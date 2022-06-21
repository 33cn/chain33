package eth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"time"

	"github.com/33cn/chain33/system/crypto/secp256k1eth"

	"github.com/33cn/chain33/rpc/jsonclient"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"google.golang.org/grpc"

	"github.com/ethereum/go-ethereum/rpc"

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
)

type ethHandler struct {
	cli     rpcclient.ChannelClient
	cfg     *ctypes.Chain33Config
	grpcCli ctypes.Chain33Client
	chainID int64
}

var (
	log = log15.New("module", "ethrpc_eth")
)

//NewEthAPI new eth api
func NewEthAPI(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) interface{} {
	e := &ethHandler{}
	e.cli.Init(c, api)
	e.cfg = cfg
	var id struct {
		EvmChainID int64 `json:"evmChainID,omitempty"`
	}
	ctypes.MustDecode(cfg.GetSubConfig().Crypto[secp256k1eth.Name], &id)
	e.chainID = id.EvmChainID
	grpcBindAddr := e.cfg.GetModuleConfig().RPC.GrpcBindAddr
	_, port, _ := net.SplitHostPort(grpcBindAddr)
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%v", port), grpc.WithInsecure())
	if err != nil {
		log.Error("NewEthAPI", "dial local grpc server err:", err)
		return nil
	}
	e.grpcCli = ctypes.NewChain33Client(conn)

	return e
}

//GetBalance eth_getBalance  tag:"latest", "earliest" or "pending"
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
	bn = bn.Mul(bn, new(big.Int).SetUint64(1e10))
	return hexutil.Big(*bn), nil
}

//nolint
func (e *ethHandler) ChainId() (hexutil.Big, error) {
	bigID := big.NewInt(e.chainID)
	return hexutil.Big(*bigID), nil
}

//BlockNumber eth_blockNumber 获取区块高度
func (e *ethHandler) BlockNumber() (hexutil.Uint64, error) {
	log.Debug("eth_blockNumber")
	header, err := e.cli.GetLastHeader()
	if err != nil {
		log.Error("eth_blockNumber", "err", err)
		return 0, err
	}

	return hexutil.Uint64(header.Height), nil
}

//GetBlockByNumber  eth_getBlockByNumber
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

//GetBlockByHash eth_getBlockByHash 通过区块哈希获取区块交易详情
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

//GetTransactionByHash eth_getTransactionByHash
func (e *ethHandler) GetTransactionByHash(txhash common.Hash) (*types.Transaction, error) {
	log.Debug("GetTransactionByHash", "txhash", txhash)
	var req ctypes.ReqHashes
	req.Hashes = append(req.Hashes, txhash.Bytes())
	txdetails, err := e.cli.GetTransactionByHash(&req)
	if err != nil {
		return nil, err
	}
	var blockHash []byte
	if len(txdetails.GetTxs()) != 0 {
		blockNum := txdetails.GetTxs()[0].Height
		hashReply, err := e.cli.GetBlockHash(&ctypes.ReqInt{Height: blockNum})
		if err == nil {
			blockHash = hashReply.GetHash()
		}
		txs, _, err := types.TxDetailsToEthReceipts(txdetails, common.BytesToHash(blockHash), e.cfg)
		if err != nil {
			return nil, err
		}
		if len(txs) != 0 {
			hash := common.BytesToHash(blockHash)
			txs[0].BlockHash = &hash
			return txs[0], nil
		}
	}

	return nil, errors.New("transaction not exist")

}

//GetTransactionReceipt eth_getTransactionReceipt
func (e *ethHandler) GetTransactionReceipt(txhash common.Hash) (*types.Receipt, error) {
	log.Debug("GetTransactionReceipt", "txhash", txhash)
	var req ctypes.ReqHashes
	var blockHash []byte
	req.Hashes = append(req.Hashes, txhash.Bytes())
	txdetails, err := e.cli.GetTransactionByHash(&req)
	if err != nil {
		return nil, err
	}
	if len(txdetails.GetTxs()) == 0 { //如果平行链查不到，则去主链查询
		return nil, errors.New("transaction not exist")
	}
	blockNum := txdetails.GetTxs()[0].Height
	hashReply, err := e.cli.GetBlockHash(&ctypes.ReqInt{Height: blockNum})
	if err == nil {
		blockHash = hashReply.GetHash()
	}

	_, receipts, err := types.TxDetailsToEthReceipts(txdetails, common.BytesToHash(blockHash), e.cfg)
	if err != nil {
		return nil, err
	}

	if len(receipts) != 0 {
		receipts[0].BlockHash = common.BytesToHash(blockHash)
		return receipts[0], nil
	}

	return nil, errors.New("transactionReceipt not exist")

}

//GetBlockTransactionCountByNumber eth_getBlockTransactionCountByNumber
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

//GetBlockTransactionCountByHash
//method:eth_getBlockTransactionCountByHash
//parameters: 32 Bytes - hash of a block
//Returns: integer of the number of transactions in this block.
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

//Accounts eth_accounts
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
		accounts = append(accounts, wallet.GetAcc().GetAddr())
	}

	return accounts, nil

}

//Call eth_call evm合约相关操作,合约相关信息查询
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
	param.Payload = []byte(fmt.Sprintf(`{"input":"%v","address":"%s"}`, msg.Data, msg.To))
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

//SendRawTransaction eth_sendRawTransaction
func (e *ethHandler) SendRawTransaction(rawData string) (hexutil.Bytes, error) {
	log.Debug("eth_sendRawTransaction", "rawData", rawData)
	rawhexData := common.FromHex(rawData)
	if rawhexData == nil {
		return nil, errors.New("wrong data")
	}
	ntx := new(etypes.Transaction)
	err := ntx.UnmarshalBinary(rawhexData)
	if err != nil {
		return nil, err
	}
	if ntx.ChainId().Int64() != e.chainID {
		log.Error("eth_sendRawTransaction", "this.chainID", e.chainID, "etx.ChainID", ntx.ChainId())
		return nil, errors.New("chainID no support")
	}
	signer := etypes.NewLondonSigner(ntx.ChainId())
	txSha3 := signer.Hash(ntx)
	v, r, s := ntx.RawSignatureValues()
	cv, err := types.CaculateRealV(v, ntx.ChainId().Uint64(), ntx.Type())
	if err != nil {
		return nil, err
	}
	//sig := append(r.Bytes()[:], append(s.Bytes()[:], cv)...)
	sig := make([]byte, 65)
	copy(sig[32-len(r.Bytes()):32], r.Bytes())
	copy(sig[64-len(s.Bytes()):64], s.Bytes())
	sig[64] = cv

	if !ethcrypto.ValidateSignatureValues(cv, r, s, false) {
		log.Error("etgh_SendRawTransaction", "ValidateSignatureValues", false, "RawSignatureValues v:", v, "to:", ntx.To(), "type", ntx.Type(), "sig", common.Bytes2Hex(sig))
		return nil, errors.New("wrong signature")
	}
	pubkey, err := ethcrypto.Ecrecover(txSha3.Bytes(), sig)
	if err != nil {
		log.Error("SendRawTransaction", "Ecrecover err:", err.Error(), "sig", common.Bytes2Hex(sig))
		return nil, err
	}

	if !ethcrypto.VerifySignature(pubkey, txSha3.Bytes(), sig[:64]) {
		log.Error("SendRawTransaction", "VerifySignature sig:", common.Bytes2Hex(sig), "pubkey:", common.Bytes2Hex(pubkey), "hash", txSha3.String())
		return nil, errors.New("wrong signature")
	}

	chain33Tx := types.AssembleChain33Tx(ntx, sig, pubkey, e.cfg)
	log.Debug("SendRawTransaction", "cacuHash", common.Bytes2Hex(chain33Tx.Hash()), "exec", string(chain33Tx.Execer))
	reply, err := e.cli.SendTx(chain33Tx)
	return reply.GetMsg(), err

}

//Sign method:eth_sign
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

//Syncing ...
//Returns an object with data about the sync status or false.
//Returns: FALSE:when not syncing,
//method:eth_syncing
//params:[]
func (e *ethHandler) Syncing() (interface{}, error) {
	log.Debug("eth_syncing", "eth_syncing", "")
	var syncing struct {
		StartingBlock string `json:"startingBlock,omitempty"`
		CurrentBlock  string `json:"currentBlock,omitempty"`
		HighestBlock  string `json:"highestBlock,omitempty"`
	}
	reply, err := e.cli.IsSync()
	if err == nil {
		var caughtUp ctypes.IsCaughtUp
		err = ctypes.Decode(reply.GetMsg(), &caughtUp)
		if err == nil {
			if caughtUp.Iscaughtup { // when not syncing
				return false, nil
			}
			//when syncing
			header, err := e.cli.GetLastHeader()
			if err == nil {
				syncing.CurrentBlock = hexutil.EncodeUint64(uint64(header.GetHeight()))
				syncing.StartingBlock = syncing.CurrentBlock
				replyBlockNum, err := e.cli.GetHighestBlockNum(&ctypes.ReqNil{})
				if err == nil {
					syncing.HighestBlock = hexutil.EncodeUint64(uint64(replyBlockNum.GetHeight()))
					return &syncing, nil
				}

			}

		}
	}

	return nil, err
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

//method:eth_getTransactionCount
//Returns:Returns the number of transactions sent from an address.
//Paramters: address,tag(disable):latest,pending,earliest
//GetTransactionCount 获取nonce
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

	var nonce struct {
		Nonce string `json:"nonce,omitempty"`
	}
	err = json.Unmarshal(result, &nonce)
	gitNonce, _ := new(big.Int).SetString(nonce.Nonce, 10)
	return hexutil.Uint64(gitNonce.Uint64()), err
}

//method:eth_estimateGas
//EstimateGas 获取gas
func (e *ethHandler) EstimateGas(callMsg *types.CallMsg) (hexutil.Uint64, error) {
	log.Debug("EstimateGas", "eth_estimateGas callMsg", callMsg)
	//组装tx
	exec := e.cfg.ExecName("evm")
	execty := ctypes.LoadExecutorType(exec)
	if execty == nil {
		return 0, ctypes.ErrNotSupport
	}

	if callMsg.To == "" {
		callMsg.To = address.ExecAddress(exec)
	}

	properFee, _ := e.cli.GetProperFee(&ctypes.ReqProperFee{
		TxCount: 1,
		TxSize:  int32(32 + len(*callMsg.Data)),
	})
	fee := properFee.GetProperFee()
	if callMsg.Data == nil || len(*callMsg.Data) == 0 {
		if fee < 1e5 {
			fee = 1e5
		}
		return hexutil.Uint64(fee), nil
	}

	var amount uint64
	if callMsg.Value != nil {
		amount = callMsg.Value.ToInt().Uint64()
	}
	action := &ctypes.EVMContractAction4Chain33{Amount: amount, GasLimit: 0, GasPrice: 0, Note: "", ContractAddr: callMsg.To}
	if callMsg.To == address.ExecAddress(exec) { //创建合约
		action.Code = *callMsg.Data
		action.Para = nil
	} else {
		action.Para = *callMsg.Data
		action.Code = nil
	}
	tx := &ctypes.Transaction{Execer: []byte(exec), Payload: ctypes.Encode(action), To: address.ExecAddress(exec), ChainID: e.cfg.GetChainID()}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	var p rpctypes.Query4Jrpc
	p.Execer = exec
	p.FuncName = "EstimateGas"
	p.Payload = []byte(fmt.Sprintf(`{"tx":"%v","from":"%v"}`, common.Bytes2Hex(ctypes.Encode(tx)), callMsg.From))
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
	if bigGas.Uint64() < uint64(fee) {
		bigGas = big.NewInt(fee)
	}

	//eth交易数据要存放在chain33 tx note 中，做2倍gas 处理
	return hexutil.Uint64(bigGas.Uint64() * 2), err

}

//GasPrice  eth_gasPrice default 10 gwei
func (e *ethHandler) GasPrice() (*hexutil.Big, error) {
	log.Debug("GasPrice", "eth_gasPrice ", "")
	return (*hexutil.Big)(big.NewInt(1).SetUint64(1e10)), nil
}

//NewHeads ...
//eth_subscribe
//params:["newHeads"]
func (e *ethHandler) NewHeads(ctx context.Context) (*rpc.Subscription, error) {
	log.Debug("eth_subscribe", "NewHeads ", "")
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}
	subscription := notifier.CreateSubscription()
	//通过Grpc 客户端
	var in ctypes.ReqSubscribe
	in.Name = string(subscription.ID)
	in.Type = 1
	stream, err := e.grpcCli.SubEvent(context.Background(), &in)
	if err != nil {
		return nil, err
	}
	go func() {

		for {
			select {
			case <-subscription.Err():
				//取消订阅
				return
			default:
				msg, err := stream.Recv()
				if err != nil {
					log.Error("NewHeads read", "err", err)
					return
				}
				eheader, _ := types.BlockHeaderToEthHeader(msg.GetHeaderSeqs().GetSeqs()[0].GetHeader())
				if err := notifier.Notify(subscription.ID, eheader); err != nil {
					log.Error("notify", "err", err)
					return

				}
			}

		}
	}()

	return subscription, nil
}

//Logs ...
//eth_subscribe
//params:["logs",{"address":"","topics":[""]}]
//address：要监听日志的源地址或地址数组，可选
//topics：要监听日志的主题匹配条件，可选
func (e *ethHandler) Logs(ctx context.Context, options *types.SubLogs) (*rpc.Subscription, error) {
	log.Debug("eth_subscribe", "eth_subscribe ", options)
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}
	subscription := notifier.CreateSubscription()
	addrObj := new(address.Address)
	addrObj.SetBytes(common.FromHex(options.Address))
	options.Address = addrObj.String()

	//通过Grpc 客户端
	var in ctypes.ReqSubscribe
	in.Name = string(subscription.ID)
	in.Contract = make(map[string]bool)
	in.Contract[options.Address] = true
	in.Type = 4

	stream, err := e.grpcCli.SubEvent(context.Background(), &in)
	if err != nil {
		return nil, err
	}
	go func() {

		for {
			select {
			case <-subscription.Err():
				//取消订阅
				return
			default:
				msg, err := stream.Recv()
				if err != nil {
					log.Error("Logs read", "err", err)
					return
				}
				var evmlogs []*types.EvmLogInfo
				for _, item := range msg.GetEvmLogs().GetLogs4EVMPerBlk() {
					logs := types.FilterEvmLogs(item, options)
					evmlogs = append(evmlogs, logs...)
				}
				//推送到订阅者
				if err := notifier.Notify(subscription.ID, evmlogs); err != nil {
					log.Error("notify", "err", err)
					return

				}

				log.Info("eth_subscribe", "logs:", evmlogs)
			}

		}
	}()

	return subscription, nil
}

//Hashrate
//method: eth_hashrate
func (e *ethHandler) Hashrate() (hexutil.Uint64, error) {
	log.Debug("eth_hashrate", "eth_hashrate ", "")
	header, err := e.cli.GetLastHeader()
	if err != nil {
		return 0, err
	}

	return hexutil.Uint64(header.Difficulty), nil
}

//GetContractorAddress   eth_getContractorAddress
func (e *ethHandler) GetContractorAddress(from common.Address, txhash string) (*common.Address, error) {
	log.Debug("eth_getContractorAddress", "addr", from, "txhash", txhash)
	var res string
	_, port, err := net.SplitHostPort(e.cfg.GetModuleConfig().RPC.JrpcBindAddr)
	if err != nil {
		return nil, errors.New("inner error")
	}
	httpStr := "http://"
	if e.cfg.GetModuleConfig().RPC.EnableTLS {
		httpStr = "https://"
	}

	rpcLaddr := fmt.Sprintf("%slocalhost:%v", httpStr, port)
	var param struct {
		Caller string `json:"caller,omitempty"`
		Txhash string `json:"txhash,omitempty"`
	}
	param.Caller = from.String()
	param.Txhash = txhash
	jcli, err := jsonclient.New("evm", rpcLaddr, false)
	if err != nil {
		return nil, errors.New("inner error")
	}

	err = jcli.Call("CalcNewContractAddr", &param, &res)
	if err != nil {
		return nil, err
	}
	c := common.HexToAddress(res)
	return &c, nil
}

//GetCode eth_getCode 获取部署合约的合约代码
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

//HistoryParam ...
type HistoryParam struct {
	BlockCount  hexutil.Uint64
	NewestBlock string
	//reward_percentiles []int
}

//FeeHistory eth_feeHistory feehistory
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
	result.GasUsedRatio = []float64{0.5, 0.8, 0.1, 0.4, 0.2}
	return &result, nil
}
