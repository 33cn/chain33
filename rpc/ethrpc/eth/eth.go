package eth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/33cn/chain33/system/crypto/secp256k1eth"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	"github.com/33cn/chain33/rpc/ethrpc/types"
	rpctypes "github.com/33cn/chain33/rpc/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"google.golang.org/grpc"
)

type ethHandler struct {
	cli           rpcclient.ChannelClient
	cfg           *ctypes.Chain33Config
	grpcCli       ctypes.Chain33Client
	filtersMu     sync.Mutex
	filters       map[rpc.ID]*filter
	evmChainID    int64
	filterTimeout time.Duration
}

var (
	log = log15.New("module", "ethrpc_eth")
)

// NewEthAPI new eth api
func NewEthAPI(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) interface{} {
	e := &ethHandler{}
	e.cli.Init(c, api)
	e.cfg = cfg
	e.filters = make(map[rpc.ID]*filter)
	e.evmChainID = secp256k1eth.GetEvmChainID()
	grpcBindAddr := e.cfg.GetModuleConfig().RPC.GrpcBindAddr
	_, port, _ := net.SplitHostPort(grpcBindAddr)
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%v", port), grpc.WithInsecure())
	if err != nil {
		log.Error("NewEthAPI", "dial local grpc server err:", err)
		return nil
	}
	e.filterTimeout = time.Minute * 5
	//推送用途
	e.grpcCli = ctypes.NewChain33Client(conn)
	go e.timeoutLoop(e.filterTimeout)
	return e
}

// GetBalance eth_getBalance  tag:"latest", "earliest" or "pending"
func (e *ethHandler) GetBalance(address string, tag *string) (hexutil.Big, error) {
	var req ctypes.ReqBalance
	var balance hexutil.Big
	req.AssetSymbol = e.cli.GetConfig().GetCoinSymbol()
	req.Execer = e.cli.GetConfig().GetCoinExec()
	req.Addresses = append(req.GetAddresses(), address)
	accounts, err := e.cli.GetBalance(&req)
	if err != nil {
		return balance, err
	}
	//转换成精度为18
	bn := new(big.Int).SetInt64(accounts[0].GetBalance())
	bn = bn.Mul(bn, new(big.Int).Div(big.NewInt(1e18), big.NewInt(e.cfg.GetCoinPrecision())))
	return hexutil.Big(*bn), nil
}

// nolint
func (e *ethHandler) ChainId() (hexutil.Big, error) {
	bigID := big.NewInt(e.evmChainID)
	return hexutil.Big(*bigID), nil
}

// BlockNumber eth_blockNumber 获取区块高度
func (e *ethHandler) BlockNumber() (hexutil.Uint64, error) {
	log.Debug("eth_blockNumber")
	header, err := e.cli.GetLastHeader()
	if err != nil {
		log.Error("eth_blockNumber", "err", err)
		return 0, err
	}

	return hexutil.Uint64(header.Height), nil
}

// GetBlockByNumber  eth_getBlockByNumber
func (e *ethHandler) GetBlockByNumber(in string, full bool) (*types.Block, error) {
	log.Debug("GetBlockByNumber", "param", in, "full", full)
	var num int64
	if len(common.FromHex(in)) == 0 {
		header, err := e.cli.GetLastHeader()
		if err != nil {
			return nil, err
		}
		num = header.GetHeight()
	} else {

		bn := new(big.Int).SetBytes(common.FromHex(in))
		num = bn.Int64()
	}
	var req ctypes.ReqBlocks
	req.Start = num
	req.End = req.Start
	req.IsDetail = full
	details, err := e.cli.GetBlocks(&req)
	if err != nil {
		log.Error("GetBlockByNumber", "err", err)
		return nil, err
	}

	fullblock := details.GetItems()[0]
	return types.BlockDetailToEthBlock(&ctypes.BlockDetails{
		Items: []*ctypes.BlockDetail{fullblock},
	}, e.cfg, full)

}

// GetBlockByHash eth_getBlockByHash 通过区块哈希获取区块交易详情
func (e *ethHandler) GetBlockByHash(txhash common.Hash, full bool) (*types.Block, error) {
	log.Debug("GetBlockByHash", "txhash", txhash, "full", full)
	var req ctypes.ReqHashes
	req.Hashes = append(req.Hashes, txhash.Bytes())
	details, err := e.cli.GetBlockByHashes(&req)
	if err != nil {
		log.Error("GetBlockByNumber", "err", err)
		return nil, err
	}
	return types.BlockDetailToEthBlock(details, e.cfg, full)

}

// GetTransactionByHash eth_getTransactionByHash
func (e *ethHandler) GetTransactionByHash(txhash common.Hash) (*types.Transaction, error) {
	log.Debug("GetTransactionByHash", "txhash", txhash)
	var req ctypes.ReqHash
	req.Hash = txhash.Bytes()

	txdetail, err := e.cli.QueryTx(&req)
	if err != nil || txdetail == nil {
		log.Error("GetTransactionByHash", "QueryTx,err:", err, "txHash:", txhash.String())
		//查询不到，不返回错误，直接返回空，与ethereum 保持一致
		return nil, nil
	}
	var blockHash []byte
	if txdetail.Tx != nil {
		blockNum := txdetail.GetHeight()
		hashReply, err := e.cli.GetBlockHash(&ctypes.ReqInt{Height: blockNum})
		if err == nil {
			blockHash = hashReply.GetHash()
		}
		var txdetails ctypes.TransactionDetails
		txdetails.Txs = append(txdetails.Txs, txdetail)
		txs, _, err := types.TxDetailsToEthReceipts(&txdetails, common.BytesToHash(blockHash), e.cfg)
		if err != nil {
			return nil, err
		}
		if len(txs) != 0 {
			hash := common.BytesToHash(blockHash)
			txs[0].BlockHash = &hash
			return txs[0], nil
		}
	}
	log.Error("eth_getTransactionByHash", "transaction not exist,err", txhash.String())
	return nil, nil

}

// GetTransactionReceipt eth_getTransactionReceipt
func (e *ethHandler) GetTransactionReceipt(txhash common.Hash) (*types.Receipt, error) {
	log.Debug("GetTransactionReceipt", "txhash", txhash)
	var req ctypes.ReqHashes
	var blockHash []byte
	req.Hashes = append(req.Hashes, txhash.Bytes())
	txdetail, err := e.cli.QueryTx(&ctypes.ReqHash{Hash: txhash.Bytes()})
	if err != nil || txdetail == nil {
		log.Error("GetTransactionReceipt", "QueryTx,err:", err, "txHash:", txhash.String())
		//查询不到，不返回错误，直接返回空，与ethereum 保持一致
		return nil, nil
	}

	blockNum := txdetail.GetHeight()
	hashReply, err := e.cli.GetBlockHash(&ctypes.ReqInt{Height: blockNum})
	if err == nil {
		blockHash = hashReply.GetHash()
	}
	var txdetails ctypes.TransactionDetails
	txdetails.Txs = append(txdetails.Txs, txdetail)
	_, receipts, err := types.TxDetailsToEthReceipts(&txdetails, common.BytesToHash(blockHash), e.cfg)
	if err != nil {
		log.Error("GetTransactionReceipt", "QueryTx,err:", err)
		return nil, err
	}

	if len(receipts) != 0 {
		receipts[0].BlockHash = common.BytesToHash(blockHash)
		return receipts[0], nil
	}
	log.Error("eth_getTransactionReceipt", "err", "transactionReceipt not exist", txhash.String())
	return nil, nil
}

// GetBlockTransactionCountByNumber eth_getBlockTransactionCountByNumber
func (e *ethHandler) GetBlockTransactionCountByNumber(blockNum *hexutil.Big) (hexutil.Uint64, error) {
	log.Debug("GetBlockTransactionCountByNumber", "blockNum", blockNum)
	var req ctypes.ReqBlocks
	req.Start = blockNum.ToInt().Int64()
	req.End = req.Start
	blockdetails, err := e.cli.GetBlocks(&req)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(len(blockdetails.GetItems()[0].GetBlock().GetTxs())), nil

}

// GetBlockTransactionCountByHash
// parameters: 32 Bytes - hash of a block
// Returns: integer of the number of transactions in this block.
//
//method:eth_getBlockTransactionCountByHash
func (e *ethHandler) GetBlockTransactionCountByHash(hash common.Hash) (hexutil.Uint64, error) {
	log.Debug("GetBlockTransactionCountByHash", "hash", hash)
	var req ctypes.ReqHashes
	req.Hashes = append(req.Hashes, hash.Bytes())
	blockdetails, err := e.cli.GetBlockByHashes(&req)
	if err != nil {
		log.Error("GetBlockByNumber", "err", err)
		return 0, err
	}

	return hexutil.Uint64(len(blockdetails.GetItems()[0].GetBlock().GetTxs())), nil
}

// Accounts eth_accounts
func (e *ethHandler) Accounts() ([]string, error) {
	log.Debug("Accounts", "Accounts", "")
	req := &ctypes.ReqAccountList{WithoutBalance: true}
	msg, err := e.cli.ExecWalletFunc("wallet", "WalletGetAccountList", req)
	if err != nil {
		return nil, err
	}
	accountsList := msg.(*ctypes.WalletAccounts)
	var accounts []string
	for _, wallet := range accountsList.Wallets {
		//过滤base58 格式的地址
		if common.IsHexAddress(wallet.GetAcc().GetAddr()) {
			accounts = append(accounts, wallet.GetAcc().GetAddr())
		}
	}

	return accounts, nil

}

// Call eth_call evm合约相关操作,合约相关信息查询
func (e *ethHandler) Call(msg types.CallMsg, tag *string) (interface{}, error) {
	log.Debug("eth_call", "msg", msg)
	var param rpctypes.Query4Jrpc
	var evmResult struct {
		Address  string `json:"address,omitempty"`
		Input    string `json:"input,omitempty"`
		Caller   string `json:"caller,omitempty"`
		RawData  string `json:"rawData,omitempty"`
		JSONData string `json:"jsonData,omitempty"`
	}

	//暂定evm
	param.Execer = e.cfg.ExecName("evm") //"evm"
	param.FuncName = "Query"
	param.Payload = []byte(fmt.Sprintf(`{"input":"%v","address":"%s","caller":"%s","ethquery":true}`, msg.Data, msg.To, msg.From))
	log.Debug("eth_call", "QueryCall param", param, "payload", string(param.Payload), "msg.To", msg.To)
	execty := ctypes.LoadExecutorType(param.Execer)
	if execty == nil {
		log.Error("Query", "funcname", param.FuncName, "err", ctypes.ErrNotSupport)
		return nil, ctypes.ErrNotSupport
	}
	decodePayload, err := execty.CreateQuery(param.FuncName, param.Payload)
	if err != nil {
		log.Error("eth_call", "err", err.Error(), "funcName", param.FuncName, "QueryCall param", param, "payload", string(param.Payload), "msg.To", msg.To)
		return nil, err
	}

	resp, err := e.cli.Query(e.cfg.ExecName(param.Execer), param.FuncName, decodePayload)
	if err != nil {
		log.Error("eth_call", "error", err)
		return nil, err
	}

	result, err := execty.QueryToJSON(param.FuncName, resp)
	if err != nil {
		log.Error("QueryToJSON", "error", err)
		return nil, err
	}
	err = json.Unmarshal(result, &evmResult)
	return evmResult.RawData, err

}

// SendRawTransaction eth_sendRawTransaction
func (e *ethHandler) SendRawTransaction(rawData string) (hexutil.Bytes, error) {
	log.Info("eth_sendRawTransaction", "rawData", rawData)
	rawhexData := common.FromHex(rawData)
	if rawhexData == nil {
		return nil, errors.New("wrong data")
	}
	ntx := new(etypes.Transaction)
	err := ntx.UnmarshalBinary(rawhexData)
	if err != nil {
		log.Error("eth_sendRawTransaction", "UnmarshalBinary", err)
		return nil, err
	}
	if ntx.ChainId().Int64() != e.evmChainID {
		log.Error("eth_sendRawTransaction", "this.chainID:", e.evmChainID, "etx.ChainID", ntx.ChainId())
		return nil, errors.New("chainID no support")
	}

	signer := etypes.NewLondonSigner(ntx.ChainId())
	txSha3 := signer.Hash(ntx)
	v, r, s := ntx.RawSignatureValues()
	cv, err := types.CaculateRealV(v, ntx.ChainId().Uint64(), ntx.Type())
	if err != nil {
		return nil, err
	}

	sig := make([]byte, 65)
	copy(sig[32-len(r.Bytes()):32], r.Bytes())
	copy(sig[64-len(s.Bytes()):64], s.Bytes())
	sig[64] = cv

	if !ethcrypto.ValidateSignatureValues(cv, r, s, false) {
		log.Error("eth_SendRawTransaction", "ValidateSignatureValues", false, "RawSignatureValues v:", v, "to:", ntx.To(), "type", ntx.Type(), "sig", common.Bytes2Hex(sig))
		return nil, errors.New("wrong signature")
	}
	pubkey, err := ethcrypto.Ecrecover(txSha3.Bytes(), sig)
	if err != nil {
		log.Error("eth_sendRawTransaction", "Ecrecover err:", err.Error(), "sig", common.Bytes2Hex(sig))
		return nil, err
	}
	// check tx nonce
	txFrom := address.PubKeyToAddr(2, pubkey)
	nonce, err := e.GetTransactionCount(txFrom, "")
	if err != nil {
		log.Error("eth_sendRawTransaction", "GetTransactionCount", err)
		return nil, err
	}
	//考虑到支持平行链和主链，平行链没有mempool，无法进行校验，只能在发送前校验
	if ntx.Nonce() < uint64(nonce) {
		log.Error("eth_sendRawTransaction", "nonce too low,tx.From", txFrom, "txnonce", ntx.Nonce(), "stateNonce", nonce)
		return nil, fmt.Errorf("nonce too low")
	}
	if ntx.Nonce() > uint64(nonce) { //非平行链下 允许更高的nonce 通过，在mempool中排序等待
		log.Warn("eth_sendRawTransaction", "nonce too high,tx.From", txFrom, "txnonce", ntx.Nonce(), "stateNonce", nonce)
		if e.cfg.IsPara() { //平行链架构下，交易是要发到主链共识的，无法校验交易的nonce是否正确
			return nil, fmt.Errorf("nonce too high")
		}
	}

	if !ethcrypto.VerifySignature(pubkey, txSha3.Bytes(), sig[:64]) {
		log.Error("SendRawTransaction", "VerifySignature sig:", common.Bytes2Hex(sig), "pubkey:", common.Bytes2Hex(pubkey), "hash", txSha3.String())
		return nil, errors.New("wrong signature")
	}

	chain33Tx := types.AssembleChain33Tx(ntx, sig, pubkey, e.cfg)
	reply, err := e.cli.SendTx(chain33Tx)
	log.Info("SendRawTransaction", "cacuHash", common.Bytes2Hex(chain33Tx.Hash()),
		"ethHash:", ntx.Hash().String(), "exec", string(chain33Tx.Execer), "mempool reply:", common.Bytes2Hex(reply.GetMsg()), "err:", err)
	//调整为返回eth 交易哈希，之前是reply.GteMsg() chain33 哈希
	conf := ctypes.Conf(e.cfg, "config.rpc.sub.eth")
	//打开 enableRlpTxHash 返回 eth 交易哈希 , 通过此hash 查询交易详情需要配合enableTxQuickIndex =false 使用
	if conf.IsEnable("enableRlpTxHash") {
		return ntx.Hash().Bytes(), err
	}
	return reply.GetMsg(), err

}

// Sign method:eth_sign
func (e *ethHandler) Sign(address string, digestHash *hexutil.Bytes) (string, error) {
	//导出私钥
	log.Debug("Sign", "eth_sign,hash", digestHash, "addr", address)
	reply, err := e.cli.ExecWalletFunc("wallet", "DumpPrivkey", &ctypes.ReqString{Data: address})
	if err != nil {
		log.Error("SignWalletRecoverTx", "execWalletFunc err", err)
		return "", err
	}
	key := reply.(*ctypes.ReplyString).GetData()
	signKey, err := ethcrypto.ToECDSA(common.FromHex(key))
	if err != nil {
		return "", err
	}

	sig, err := ethcrypto.Sign(*digestHash, signKey)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(sig), nil
}

// Syncing ...
// Returns an object with data about the sync status or false.
// Returns: FALSE:when not syncing,
// params:[]
//
//method:eth_syncing
func (e *ethHandler) Syncing() (interface{}, error) {
	log.Debug("eth_syncing", "eth_syncing", "")
	var syncing struct {
		StartingBlock string `json:"startingBlock,omitempty"`
		CurrentBlock  string `json:"currentBlock,omitempty"`
		HighestBlock  string `json:"highestBlock,omitempty"`
	}
	reply, err := e.cli.IsSync()
	if err != nil {
		return nil, err

	}
	if !reply.IsOk { //未同步
		//when syncing
		header, err := e.cli.GetLastHeader()
		if err == nil {
			blockHeight, _ := e.cli.GetHighestBlockNum(&ctypes.ReqNil{})
			syncing.CurrentBlock = hexutil.EncodeUint64(uint64(header.GetHeight()))
			syncing.StartingBlock = syncing.CurrentBlock
			syncing.HighestBlock = hexutil.EncodeUint64(uint64(blockHeight.GetHeight()))
			return &syncing, nil
		}

	}

	return !reply.IsOk, err
}

//Mining...
//method:eth_mining
//Paramtesrs:none
//Returns:Returns true if client is actively mining new blocks.

func (e *ethHandler) Mining() (bool, error) {
	log.Debug("eth_mining", "call", "")
	msg, err := e.cli.ExecWalletFunc("wallet", "GetWalletStatus", &ctypes.ReqNil{})
	if err == nil {
		status := msg.(*ctypes.WalletStatus)
		if status.IsAutoMining {
			return true, nil
		}
		return false, nil
	}
	return false, err
}

// GetTransactionCount Returns:Returns the number of transactions sent from an address.
// Parameters: address,tag(disable):latest,pending,earliest
// GetTransactionCount 获取nonce
//
//method:eth_getTransactionCount
func (e *ethHandler) GetTransactionCount(address, tag string) (hexutil.Uint64, error) {
	log.Debug("GetTransactionCount", "eth_getTransactionCount address", address)
	exec := e.cfg.ExecName("evm")
	execty := ctypes.LoadExecutorType(exec)
	if execty == nil {
		return 0, ctypes.ErrNotSupport
	}

	var param rpctypes.Query4Jrpc
	param.FuncName = "GetNonce"
	param.Execer = exec
	param.Payload = []byte(fmt.Sprintf(`{"address":"%v"}`, address))
	queryparam, err := execty.CreateQuery(param.FuncName, param.Payload)
	if err != nil {
		return 0, err
	}
	resp, err := e.cli.Query(param.Execer, param.FuncName, queryparam)
	if err != nil {
		return 0, err
	}

	result, err := execty.QueryToJSON(param.FuncName, resp)
	if err != nil {
		return 0, err
	}
	log.Info("eth_getTransactionCount", "nonce", string(result))
	var nonce struct {
		Nonce string `json:"nonce,omitempty"`
	}
	err = json.Unmarshal(result, &nonce)
	bigNonce, _ := new(big.Int).SetString(nonce.Nonce, 10)

	return hexutil.Uint64(bigNonce.Uint64()), err
}

// EstimateGas 获取gas
//
//method:eth_estimateGas
func (e *ethHandler) EstimateGas(callMsg *types.CallMsg) (hexutil.Uint64, error) {
	log.Info("EstimateGas", "callMsg.From", callMsg.From, "callMsg.To:", callMsg.To, "callMsg.Value:", callMsg.Value)
	if callMsg == nil {
		return 0, errors.New("callMsg empty")
	}
	//组装tx
	exec := e.cfg.ExecName("evm")
	execty := ctypes.LoadExecutorType(exec)
	if execty == nil {
		return 0, ctypes.ErrNotSupport
	}

	if callMsg.To == "" {
		callMsg.To = address.ExecAddress(exec)
	}

	var amount uint64
	if callMsg.Value != nil && callMsg.Value.ToInt() != nil {
		ethUnit := big.NewInt(1e18)
		bigAmount := new(big.Int).Div(callMsg.Value.ToInt(), ethUnit.Div(ethUnit, big.NewInt(1).SetInt64(e.cfg.GetCoinPrecision())))
		amount = bigAmount.Uint64()
	}

	action := &ctypes.EVMContractAction4Chain33{Amount: amount, GasLimit: 0, GasPrice: 0, Note: "", ContractAddr: callMsg.To}
	if callMsg.Data != nil {
		etx := etypes.NewTx(&etypes.LegacyTx{
			Nonce:    0,
			Value:    big.NewInt(int64(amount)),
			Gas:      1e5,
			GasPrice: big.NewInt(1e9),
			Data:     *callMsg.Data,
		})
		etxBs, _ := etx.MarshalBinary()
		action.Note = common.Bytes2Hex(etxBs)
		if callMsg.To == address.ExecAddress(exec) { //创建合约
			action.Code = *callMsg.Data
			action.Para = nil
		} else {
			action.Para = *callMsg.Data
			action.Code = nil
		}
	}

	tx := ctypes.Transaction{Execer: []byte(exec), Payload: ctypes.Encode(action), To: address.ExecAddress(exec), ChainID: e.cfg.GetChainID()}
	nonce, err := e.GetTransactionCount(callMsg.From, "")
	if err != nil {
		log.Error("eth_EstimateGas", "GetTransactionCount", err)
		return 0, err
	}
	tx.Nonce = int64(nonce)
	estimateTxSize := int32(tx.Size() + 300)
	properFee, _ := e.cli.GetProperFee(&ctypes.ReqProperFee{
		TxCount: 1,
		TxSize:  estimateTxSize,
	})

	fee := properFee.GetProperFee()
	//GetMinTxFeeRate 默认1e5
	realFee, _ := tx.GetRealFee(e.cfg.GetMinTxFeeRate())
	//判断是否是代理执行的地址，如果是，则直接返回，不会交给evm执行器去模拟执行计算gas.
	if callMsg.To == e.cfg.GetModuleConfig().Exec.ProxyExecAddress {
		rightFee := realFee
		if realFee < fee {
			rightFee = fee
		}
		return hexutil.Uint64(rightFee), nil
	}

	var minimumGas int64 = 21000
	if callMsg.Data == nil || len(*callMsg.Data) == 0 {
		if fee < e.cfg.GetMinTxFeeRate() {
			fee = e.cfg.GetMinTxFeeRate()
		}
		if fee < minimumGas { //不能低于21000
			fee = minimumGas
		}
		return hexutil.Uint64(fee), nil
	}

	var p rpctypes.Query4Jrpc
	p.Execer = exec
	p.FuncName = "EstimateGas"
	p.Payload = []byte(fmt.Sprintf(`{"tx":"%v","from":"%v","ethquery":true}`, common.Bytes2Hex(ctypes.Encode(&tx)), callMsg.From))
	queryparam, err := execty.CreateQuery(p.FuncName, p.Payload)
	if err != nil {
		return 0, err
	}
	resp, err := e.cli.Query(p.Execer, p.FuncName, queryparam)
	if err != nil {
		return 0, err
	}

	result, err := execty.QueryToJSON(p.FuncName, resp)
	if err != nil {
		return 0, err
	}
	var gas struct {
		Gas string `json:"gas,omitempty"`
	}
	err = json.Unmarshal(result, &gas)
	if err != nil {
		return 0, err
	}

	bigGas, _ := new(big.Int).SetString(gas.Gas, 10)

	var finalFee = realFee
	if bigGas.Uint64() > uint64(realFee) {
		finalFee = bigGas.Int64()
	}
	if finalFee < minimumGas {
		finalFee = minimumGas
	}
	log.Info("EstimateGas", "evmNeedGas:", bigGas, "properFee:", fee, "realFee:", realFee, "finalFee:", finalFee, "tx.size", tx.Size())
	return hexutil.Uint64(finalFee), err

}

// GasPrice  eth_gasPrice default 10 gwei
func (e *ethHandler) GasPrice() (*hexutil.Big, error) {
	log.Debug("GasPrice", "eth_gasPrice ", "")
	return (*hexutil.Big)(new(big.Int).Div(big.NewInt(1e18), big.NewInt(e.cfg.GetCoinPrecision()))), nil
}

// Hashrate
// method: eth_hashrate
func (e *ethHandler) Hashrate() (hexutil.Uint64, error) {
	log.Debug("eth_hashrate", "eth_hashrate ", "")
	header, err := e.cli.GetLastHeader()
	if err != nil {
		return 0, err
	}

	return hexutil.Uint64(header.Difficulty), nil
}

// GetContractorAddress   eth_getContractorAddress
func (e *ethHandler) GetContractorAddress(from common.Address, nonce hexutil.Uint64) (*common.Address, error) {
	log.Debug("eth_getContractorAddress", "addr", from, "nonce", nonce)

	contractorAddr := ethcrypto.CreateAddress(from, uint64(nonce))
	return &contractorAddr, nil

}

// GetCode eth_getCode 获取部署合约的合约代码
func (e *ethHandler) GetCode(addr *common.Address, tag string) (*hexutil.Bytes, error) {

	exec := e.cfg.ExecName("evm")
	log.Info("eth_GetCode", "addr", addr, "exec", exec)
	execty := ctypes.LoadExecutorType(exec)
	if execty == nil { //for test case
		return nil, ctypes.ErrNotSupport
	}
	var code []byte
	var p rpctypes.Query4Jrpc
	p.Execer = exec
	p.FuncName = "GetCode"
	p.Payload = []byte(fmt.Sprintf(`{"addr":"%v"}`, addr.String()))
	queryparam, err := execty.CreateQuery(p.FuncName, p.Payload)
	if err != nil {
		log.Error("eth_GetCode", "CreateQuery err", err)
		return nil, err
	}
	resp, err := e.cli.Query(p.Execer, p.FuncName, queryparam)
	if err != nil {
		log.Error("eth_GetCode", "Query err", err)
		return (*hexutil.Bytes)(&code), nil
	}

	result, err := execty.QueryToJSON(p.FuncName, resp)
	if err != nil {
		log.Error("eth_GetCode", "QueryToJSON err", err)
		return nil, err
	}
	log.Debug("GetCode", "resp", string(result))
	var ret struct {
		Creator  string         ` json:"creator,omitempty"`
		Name     string         ` json:"name,omitempty"`
		Alias    string         ` json:"alias,omitempty"`
		Addr     string         ` json:"addr,omitempty"`
		Code     *hexutil.Bytes ` json:"code,omitempty"`
		CodeHash []byte         ` json:"codeHash,omitempty"`
		// 绑定ABI数据 ForkEVMABI
		Abi string `json:"abi,omitempty"`
	}

	err = json.Unmarshal(result, &ret)
	if err != nil {
		log.Error("GetCode", "unmarshal err", err)
		return nil, err
	}

	return ret.Code, nil

}

// HistoryParam ...
type HistoryParam struct {
	BlockCount  hexutil.Uint64
	NewestBlock string
	//reward_percentiles []int
}

// FeeHistory eth_feeHistory feehistory
func (e *ethHandler) FeeHistory(BlockCount, tag string, options []interface{}) (interface{}, error) {
	log.Debug("eth_feeHistory", "FeeHistory blockcout", BlockCount)
	header, err := e.cli.GetLastHeader()
	if err != nil {
		return nil, err
	}
	latestBlockNum := header.GetHeight()
	var result struct {
		OldestBlock   hexutil.Uint64 `json:"oldestBlock,omitempty"`
		Reward        []interface{}  `json:"reward,omitempty"`
		BaseFeePerGas []string       `json:"baseFeePerGas,omitempty"`
		GasUsedRatio  []float64      `json:"gasUsedRatio,omitempty"`
	}
	result.OldestBlock = hexutil.Uint64(latestBlockNum)
	result.BaseFeePerGas = []string{"0x12", "0x10", "0x10", "0x10", "0x10"}
	result.GasUsedRatio = []float64{0.99, 0.99, 0.99, 0.99, 0.99}
	return &result, nil
}

// GetLogs eth_getLogs
func (e *ethHandler) GetLogs(options *types.FilterQuery) ([]*types.EvmLog, error) {
	//通过Grpc 客户端
	log.Info("GetLogs", "Logs,options:", options)
	filter, err := types.NewFilter(e.grpcCli, e.cfg, options)
	if err != nil {
		return nil, err
	}
	evmlogs := []*types.EvmLog{}
	if options.BlockHash != nil {
		//直接查询对应的block 的交易回执信息
		if options.FromBlock != "" || options.ToBlock != "" {
			return nil, fmt.Errorf("cannot specify both BlockHash and FromBlock/ToBlock")
		}

		return filter.FilterBlockDetail([][]byte{options.BlockHash.Bytes()})
	}
	var fromBlock, toBlock uint64
	header, err := e.cli.GetLastHeader()
	if err != nil {
		return nil, err
	}

	fromBlock, err = hexutil.DecodeUint64(options.FromBlock)
	if err != nil {
		fromBlock = uint64(header.GetHeight())
		toBlock = fromBlock
	} else {
		if options.ToBlock == "latest" || options.ToBlock == "" {
			toBlock = uint64(header.GetHeight())
		} else {
			toBlock, err = hexutil.DecodeUint64(options.ToBlock)
			if err != nil {
				return nil, err
			}
		}
	}
	if fromBlock > uint64(header.GetHeight()) {
		fromBlock = uint64(header.GetHeight())
	}
	itemNum := toBlock - fromBlock
	if itemNum > 10000 {
		return nil, errors.New("query returned more than 10000 results")
	}
	var hashes [][]byte
	for i := fromBlock; i <= toBlock; i++ {
		replyHash, err := e.grpcCli.GetBlockHash(context.Background(), &ctypes.ReqInt{
			Height: int64(i),
		})
		if err == nil {
			hashes = append(hashes, replyHash.Hash)
		}

	}
	logs, err := filter.FilterBlockDetail(hashes)
	if err == nil {
		evmlogs = append(evmlogs, logs...)
	}

	return evmlogs, nil
}
