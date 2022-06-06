package eth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/33cn/chain33/rpc/jsonclient"
	"math/big"
	"math/rand"
	"net"
	"time"

	"github.com/33cn/chain33/system/address/eth"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"google.golang.org/grpc"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/33cn/chain33/client"
	chain33Common "github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	"github.com/33cn/chain33/rpc/ethrpc/types"
	rpctypes "github.com/33cn/chain33/rpc/types"
	dtypes "github.com/33cn/chain33/system/dapp/coins/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type ethHandler struct {
	cli     rpcclient.ChannelClient
	cfg     *ctypes.Chain33Config
	grpcCli ctypes.Chain33Client
}

var (
	log = log15.New("module", "ethrpc_eth")
)

//NewEthAPI new eth api
func NewEthAPI(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) interface{} {
	e := &ethHandler{}
	e.cli.Init(c, api)
	e.cfg = cfg
	grpcBindAddr := e.cfg.GetModuleConfig().RPC.GrpcBindAddr
	_, port, _ := net.SplitHostPort(grpcBindAddr)
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%v", port), grpc.WithInsecure())
	if err != nil {
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
	//check address type
	if common.IsHexAddress(address) {
		address = common.HexToAddress(address).String()
	}
	req.Addresses = append(req.GetAddresses(), address)
	accounts, err := e.cli.GetBalance(&req)
	if err != nil {
		return balance, err
	}
	//转换成精度为18
	bn := new(big.Int).SetInt64(accounts[0].GetBalance())
	bn = bn.Mul(bn, new(big.Int).SetInt64(1e10))
	balance = hexutil.Big(*bn)
	log.Info("GetBalance", "param address:", address, "balance:", balance)
	return balance, nil

}

//nolint
func (e *ethHandler) ChainId() (hexutil.Big, error) {
	return hexutil.Big(*new(big.Int).SetInt64(int64(e.cfg.GetChainID()))), nil
}

//BlockNumber eth_blockNumber 获取区块高度
func (e *ethHandler) BlockNumber() (hexutil.Uint64, error) {
	log.Info("BlockNumber", "...")
	header, err := e.cli.GetLastHeader()
	if err != nil {
		return 0, err
	}

	return hexutil.Uint64(header.Height), nil
}

//GetBlockByNumber  eth_getBlockByNumber
func (e *ethHandler) GetBlockByNumber(in string, full bool) (*types.Block, error) {
	log.Info("GetBlockByNumber", "param", in, "full", full)
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
	log.Debug("GetBlockByNumber", "start", req.Start)
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
	log.Info("GetBlockByHash", "txhash", txhash, "full", full)
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
	log.Info("GetTransactionByHash", "txhash", txhash)
	var req ctypes.ReqHashes
	req.Hashes = append(req.Hashes, txhash.Bytes())
	txdetails, err := e.cli.GetTransactionByHash(&req)
	if err != nil {
		return nil, err
	}
	txs, _, err := types.TxDetailsToEthTx(txdetails, e.cfg)
	if err != nil {
		return nil, err
	}
	return txs[0], nil
}

//GetTransactionReceipt eth_getTransactionReceipt
func (e *ethHandler) GetTransactionReceipt(txhash common.Hash) (*types.Receipt, error) {
	log.Info("GetTransactionReceipt", "txhash", txhash)
	var req ctypes.ReqHashes
	req.Hashes = append(req.Hashes, txhash.Bytes())
	txdetails, err := e.cli.GetTransactionByHash(&req)
	if err != nil {
		return nil, err
	}
	_, receipts, err := types.TxDetailsToEthTx(txdetails, e.cfg)
	if err != nil {
		return nil, err
	}
	return receipts[0], nil

}

//GetBlockTransactionCountByNumber eth_getBlockTransactionCountByNumber
func (e *ethHandler) GetBlockTransactionCountByNumber(blockNum *hexutil.Big) (hexutil.Uint64, error) {
	log.Info("GetBlockTransactionCountByNumber", "blockNum", blockNum)
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
	log.Info("GetBlockTransactionCountByHash", "hash", hash)
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
	log.Info("Accounts", "Accounts", "")
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
	log.Info("Call", "eth_call", msg)
	var param rpctypes.Query4Jrpc
	var evmResult struct {
		Address  string `json:"address,omitempty"`
		Input    string `json:"input,omitempty"`
		Caller   string `json:"caller,omitempty"`
		RawData  string `json:"rawData,omitempty"`
		JSONData string `json:"jsonData,omitempty"`
	}

	if common.IsHexAddress(msg.To) {
		msg.To = common.HexToAddress(msg.To).String()
		//临时转换为BTY地址格式
		if e.cfg.GetModuleConfig().Address.DefaultDriver == "btc" {
			addrObj := new(address.Address)
			addrObj.SetBytes(common.FromHex(msg.To))
			msg.To = addrObj.String()
		}

	}

	if common.IsHexAddress(msg.From) {
		msg.From = common.HexToAddress(msg.From).String()
	}

	//暂定evm
	param.Execer = e.cfg.ExecName("evm") //"evm"
	param.FuncName = "Query"
	param.Payload = []byte(fmt.Sprintf(`{"input":"%v","address":"%s"}`, msg.Data, msg.To))
	log.Info("eth_call", "QueryCall param", param, "payload", string(param.Payload), "msg.To", msg.To)

	execty := ctypes.LoadExecutorType(param.Execer)
	if execty == nil {
		log.Error("Query", "funcname", param.FuncName, "err", ctypes.ErrNotSupport)
		return nil, ctypes.ErrNotSupport
	}
	decodePayload, err := execty.CreateQuery(param.FuncName, param.Payload)
	if err != nil {
		log.Error("EventQuery1", "err", err.Error(), "funcName", param.FuncName)
		return nil, err
	}

	resp, err := e.cli.Query(e.cfg.ExecName(param.Execer), param.FuncName, decodePayload)
	if err != nil {
		log.Error("eth_call", "error", err)
		return nil, err
	}

	//log.Info("Eth_Call", "QueryCall resp", resp.String(),"execer",e.cfg.ExecName(param.Execer),"json ",string(jmb))
	result, err := execty.QueryToJSON(param.FuncName, resp)
	if err != nil {
		log.Error("QueryToJSON", "error", err)
		return nil, err
	}
	err = json.Unmarshal(result, &evmResult)
	//log.Info("result",hexutil.Encode(result),"str result",string(result))
	return evmResult.RawData, err

}

//SendTransaction  eth_sendTransaction
func (e *ethHandler) SendTransaction(msg *types.CallMsg) (string, error) {
	log.Info("SendTransaction", "eth_sendTransaction", msg)
	reply, err := e.cli.ExecWalletFunc("wallet", "DumpPrivkey", &ctypes.ReqString{Data: msg.From})
	if err != nil {
		log.Error("SignWalletRecoverTx", "execWalletFunc err", err)
		return "", err
	}

	key := reply.(*ctypes.ReplyString).GetData()
	var data []byte
	var tx *ctypes.Transaction
	if msg.Data != nil {
		exec := e.cfg.ExecName("evm")
		action := ctypes.EVMContractAction4Chain33{
			Amount:       0,
			GasLimit:     uint64(*msg.Gas),
			GasPrice:     1,
			Code:         nil,
			Para:         *msg.Data,
			Alias:        "",
			Note:         "",
			ContractAddr: msg.To,
		}
		if msg.To == "" { // 部署合约
			action.Para = nil
			action.Code = *msg.Data
			msg.To = address.ExecAddress(exec)
			action.ContractAddr = msg.To
		}

		tx = &ctypes.Transaction{Execer: []byte(exec), Payload: ctypes.Encode(&action), Fee: 0, To: msg.To, Nonce: rand.New(rand.NewSource(time.Now().UnixNano())).Int63()}
	} else {
		exec := e.cfg.ExecName("coins") //e.cfg.GetParaName() +"coins"
		bn := msg.Value.ToInt()
		bn = bn.Div(bn, big.NewInt(1).SetUint64(1e10))
		v := &dtypes.CoinsAction_Transfer{Transfer: &ctypes.AssetsTransfer{Cointoken: e.cfg.GetCoinSymbol(), Amount: bn.Int64(), Note: []byte("")}}
		transfer := &dtypes.CoinsAction{Value: v, Ty: dtypes.CoinsActionTransfer}
		data = ctypes.Encode(transfer)
		tx = &ctypes.Transaction{Execer: []byte(exec), Payload: data, Fee: 0, To: msg.To, Nonce: rand.New(rand.NewSource(time.Now().UnixNano())).Int63()}
	}

	txCache := ctypes.NewTransactionCache(tx)
	fee, err := txCache.GetRealFee(e.cfg.GetMinTxFeeRate())
	if err != nil {
		return "", err
	}
	tx.Fee = fee
	if tx.Fee < int64(*msg.Gas) {
		tx.Fee = int64(*msg.Gas)
	}

	c, err := crypto.Load("secp256k1sha3", -1)
	if err != nil {
		return "", err
	}
	signKey, err := c.PrivKeyFromBytes(common.FromHex(key))
	if err != nil {
		return "", err
	}

	sig := signKey.Sign(ctypes.Encode(tx)).Bytes()
	return e.AssembleSign(common.Bytes2Hex(ctypes.Encode(tx)), common.Bytes2Hex(sig))

}

//SendRawTransaction eth_sendRawTransaction 发送交易
func (e *ethHandler) SendRawTransaction(rawData string) (string, error) {
	log.Info("SendRawTransaction", "eth_sendRawTransaction", rawData)
	hexData := common.FromHex(rawData)
	if hexData == nil {
		return "", errors.New("wrong data")
	}
	var tx ctypes.Transaction
	//按照Chain33交易格式进行解析
	err := ctypes.Decode(hexData, &tx)
	if err != nil {
		log.Error("SendRawTransaction", "param", tx.String(), "err", err.Error())
		return "", err
	}
	log.Debug("SendTransaction", "param", tx.String())
	reply, err := e.cli.SendTx(&tx)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(reply.GetMsg()), nil
}

//Sign method:eth_sign
func (e *ethHandler) Sign(address string, digestHash *hexutil.Bytes) (string, error) {
	//导出私钥
	log.Info("Sign", "eth_sign,hash", digestHash, "addr", address)
	if common.IsHexAddress(address) {
		address = common.HexToAddress(address).String()
	}
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

//AssembleSign eth_assembleSign
func (e *ethHandler) AssembleSign(unSignTx, sigData string) (string, error) {
	log.Info("AssembleSign", "eth_assembleSign,unSignTx", unSignTx, "sigData", sigData)
	var tx ctypes.Transaction
	err := ctypes.Decode(common.FromHex(unSignTx), &tx)
	if err != nil {
		return "", err
	}

	sig := common.FromHex(sigData)
	sig, err = types.ParaseEthSigData(sig, e.cfg.GetChainID())
	if err != nil {
		return "", err
	}
	pubkey, err := ethcrypto.Ecrecover(chain33Common.Sha3SigHash(common.FromHex(unSignTx)), sig)
	if err != nil {
		log.Error("SendRawTransaction", "Ecrecover err:", err.Error())
		return "", err
	}

	epub, err := ethcrypto.UnmarshalPubkey(pubkey)
	if err != nil {
		return "", err
	}

	sginTy := ctypes.EncodeSignID(ctypes.SECP256K1SHA3, eth.ID)
	/*if !common.IsHexAddress(tx.From()) {
		sginTy = ctypes.EncodeSignID(ctypes.SECP256K1SHA3, btc.NormalAddressID)
	}*/

	tx.Signature = &ctypes.Signature{
		Signature: sig,
		Ty:        sginTy,
		Pubkey:    ethcrypto.CompressPubkey(epub),
	}

	return hexutil.Encode(ctypes.Encode(&tx)), nil

}

//SignTransaction method:eth_signTransaction
func (e *ethHandler) SignTransaction(msg *types.CallMsg) (string, error) {
	log.Info("SignTransaction", "eth_signTransaction,unSignTx", msg)
	var tx *ctypes.Transaction
	var data []byte

	if msg.Data == nil {
		//普通的coins 转账
		//if len(common.FromHex(msg.Value)) == 0 {
		//	return "", errors.New("invalid hex string callMsg.Value")
		//}

		exec := e.cfg.ExecName("coins") //e.cfg.GetParaName() +"coins"
		v := &dtypes.CoinsAction_Transfer{Transfer: &ctypes.AssetsTransfer{Cointoken: e.cfg.GetCoinSymbol(), Amount: msg.Value.ToInt().Int64(), Note: []byte("")}}
		transfer := &dtypes.CoinsAction{Value: v, Ty: dtypes.CoinsActionTransfer}
		data = ctypes.Encode(transfer)
		tx = &ctypes.Transaction{Execer: []byte(exec), Payload: data, Fee: 0, To: msg.To, Nonce: rand.New(rand.NewSource(time.Now().UnixNano())).Int63()}
	} else {
		//evm tx
		//if len(common.FromHex(msg.Data)) == 0 {
		//	return "", errors.New("invalid hex string callMsg.Data")
		//}

		action := ctypes.EVMContractAction4Chain33{
			Amount:       0,
			GasLimit:     uint64(*msg.Gas),
			GasPrice:     1,
			Code:         nil,
			Para:         *msg.Data,
			Alias:        "",
			Note:         "",
			ContractAddr: msg.To,
		}

		exec := e.cfg.ExecName("evm")
		tx = &ctypes.Transaction{Execer: []byte(exec), Payload: ctypes.Encode(&action), Fee: 0, To: msg.To}
	}
	tx.Fee, _ = tx.GetRealFee(e.cfg.GetMinTxFeeRate())
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	tx.ChainID = e.cfg.GetChainID()
	//对TX 进行签名
	unsigned := &ctypes.ReqSignRawTx{
		Addr:   msg.From,
		TxHex:  common.Bytes2Hex(ctypes.Encode(tx)),
		Expire: "0",
	}
	signedTx, err := e.cli.ExecWalletFunc("wallet", "SignRawTx", unsigned)
	if err != nil {
		return "", err
	}

	return signedTx.(*ctypes.ReplySignRawTx).TxHex, nil

}

//Syncing ...
//Returns an object with data about the sync status or false.
//Returns: FALSE:when not syncing,
//method:eth_syncing
//params:[]
func (e *ethHandler) Syncing() (interface{}, error) {
	log.Info("Syncing", "eth_syncing", "")
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
	log.Info("Mining", "eth_mining", "")
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
	log.Info("GetTransactionCount", "eth_getTransactionCount address", address)
	exec := e.cfg.ExecName("evm")
	execty := ctypes.LoadExecutorType(exec)
	if execty == nil {
		return 0, ctypes.ErrNotSupport
	}

	if common.IsHexAddress(address) {
		address = common.HexToAddress(address).String()
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

	//log.Info("result", hexutil.Encode(result), "str result", string(result))
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
	log.Info("EstimateGas", "eth_estimateGas callMsg", callMsg)
	//组装tx
	exec := e.cfg.ExecName("evm")
	execty := ctypes.LoadExecutorType(exec)
	if execty == nil {
		return 0, ctypes.ErrNotSupport
	}

	if callMsg.To == "" {
		callMsg.To = address.ExecAddress(exec)
	}
	if callMsg.Data == nil {
		return 1e5, nil
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
	bigGas, _ := new(big.Int).SetString(gas.Gas, 10)
	return hexutil.Uint64(bigGas.Uint64()), err

}

//GasPrice
//funcname: eth_gasPrice
func (e *ethHandler) GasPrice() (*hexutil.Big, error) {
	log.Info("GasPrice", "eth_gasPrice ", "")
	bn, _ := big.NewInt(1).SetString("10000000000", 10) //10gwei
	return (*hexutil.Big)(bn), nil
}

//NewHeads ...
//eth_subscribe
//params:["newHeads"]
func (e *ethHandler) NewHeads(ctx context.Context) (*rpc.Subscription, error) {
	log.Info("NewHeads", "eth_subscribe ", "")
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
				ehead, _ := types.BlockHeaderToEthHeader(msg.GetHeaderSeqs().GetSeqs()[0].GetHeader())
				if err := notifier.Notify(subscription.ID, ehead); err != nil {
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
	log.Info("Logs", "eth_subscribe ", "")
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}
	subscription := notifier.CreateSubscription()

	if common.IsHexAddress(options.Address) {
		options.Address = common.HexToAddress(options.Address).String()
		//临时处理
		addrObj := new(address.Address)
		addrObj.SetBytes(common.FromHex(options.Address))
		options.Address = addrObj.String()
	}
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

//CreateRawTransaction eth_createRawTransaction
func (e *ethHandler) CreateRawTransaction(msg *types.CallMsg) (*types.HexRawTx, error) {
	//log.Info("CreateRawTransaction", "eth_createRawTransaction ", msg)
	log.Info("CreateRawTransaction", "msg", msg, "value", msg.Value, "gas:", msg.Gas, "msg.Data", msg.Data)
	var execer = e.cfg.ExecName("evm")
	var payload []byte
	var evmdata *ctypes.EVMContractAction4Chain33
	var gas uint64
	if msg.Data != nil {
		var code, para []byte

		if msg.To == "" || len(common.FromHex(msg.To)) == 0 {
			msg.To = address.ExecAddress(execer)
			code = *msg.Data
			para = nil
			log.Info("CreateRawTransaction", "code size", len(code), "para", len(para))
		} else {
			para = *msg.Data
			code = nil
		}
		gas = uint64(*msg.Gas)
		evmdata = &ctypes.EVMContractAction4Chain33{Amount: 0,
			GasLimit:     gas,
			GasPrice:     1, //uint32(msg.GasPrice.ToInt().Int64()),
			ContractAddr: msg.To,
			Alias:        "ERC20:" + "token",
			Para:         para,
			Code:         code}

		payload = ctypes.Encode(evmdata)
	} else {
		var bn *big.Int
		if msg.Value != nil {
			bn = msg.Value.ToInt()
			bn = bn.Div(bn, big.NewInt(1).SetUint64(1e10))
		} else {
			return nil, errors.New("invalid hex string  msg.Value")
		}

		execer = e.cfg.ExecName("coins")
		action := dtypes.CoinsAction_Transfer{
			Transfer: &ctypes.AssetsTransfer{
				Amount: bn.Int64(),
			},
		}
		payload = ctypes.Encode(&dtypes.CoinsAction{Value: &action, Ty: dtypes.CoinsActionTransfer})
	}

	tx := &ctypes.Transaction{
		Execer:  []byte(execer),
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      msg.To,
		Payload: payload,
		ChainID: e.cfg.GetChainID(),
	}

	txCache := ctypes.NewTransactionCache(tx)
	fee, err := txCache.GetRealFee(e.cfg.GetMinTxFeeRate())
	if err != nil {
		return nil, err
	}
	tx.Fee = fee
	if tx.Fee < int64(gas) {
		tx.Fee = int64(gas)
	}

	rawHex := &types.HexRawTx{
		RawTx:      ctypes.Encode(tx),
		Sha256Hash: chain33Common.Sha3SigHash(ctypes.Encode(tx)),
	}
	log.Info("CreateRawTransaction", "rawHex", common.Bytes2Hex(rawHex.RawTx), "\nsha3hash:", common.Bytes2Hex(chain33Common.Sha3(ctypes.Encode(tx))), "RealFee:", tx.Fee)
	return rawHex, nil
}

//SendSignedTransaction eth_sendSignedTransaction 发送交易
func (e *ethHandler) SendSignedTransaction(msg *types.HexRawTx) (hexutil.Bytes, error) {
	log.Info("SendSignedTransaction", "eth_sendSignedTransaction ", msg)
	var tx ctypes.Transaction
	//按照Chain33交易格式进行解析
	err := ctypes.Decode(msg.RawTx, &tx)
	if err != nil {
		log.Error("SendRawTransaction", "param", tx.String(), "err", err.Error())
		return nil, err
	}
	log.Info("SendSignedTransaction", "param", tx.String(), "sig size", len(msg.Signature), "msg:", msg)
	sig, err := types.ParaseEthSigData(msg.Signature, e.cfg.GetChainID())
	if err != nil {
		return nil, err
	}
	pubkey, err := ethcrypto.Ecrecover(chain33Common.Sha3(msg.RawTx), sig)
	if err != nil {
		log.Error("SendRawTransaction", "Ecrecover err:", err.Error())
		return nil, err
	}

	sginTy := ctypes.EncodeSignID(ctypes.SECP256K1SHA3, eth.ID)
	//if !common.IsHexAddress(tx.From()) && tx.From() != "" {
	//	sginTy = ctypes.EncodeSignID(ctypes.SECP256K1SHA3, btc.NormalAddressID)
	//}
	tx.Signature = &ctypes.Signature{
		Signature: msg.Signature,
		Ty:        sginTy,
		Pubkey:    pubkey, //ethcrypto.CompressPubkey(epub),
	}

	log.Info("SendSignedTransaction", "rawtx:", common.Bytes2Hex(ctypes.Encode(&tx)), "sig", msg.Signature)
	reply, err := e.cli.SendTx(&tx)
	if err != nil {
		return nil, err
	}
	log.Info("SendSignedTransaction", "hash", hexutil.Encode(tx.Hash()), "sendhash", common.Bytes2Hex(reply.GetMsg()))
	return reply.GetMsg(), nil
}

//Hashrate
//method: eth_hashrate
func (e *ethHandler) Hashrate() (hexutil.Uint64, error) {
	log.Info("Hashrate", "eth_hashrate ", "")
	header, err := e.grpcCli.GetLastHeader(context.Background(), &ctypes.ReqNil{})
	if err != nil {
		return 0, err
	}

	return hexutil.Uint64(header.Difficulty), nil
}

//GetContractorAddress   eth_getContractorAddress
func (e *ethHandler) GetContractorAddress(from common.Address, txhash string) (*common.Address, error) {

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

//GetCode eth_getCode
func (e *ethHandler) GetCode(addr *common.Address, tag string) (*hexutil.Bytes, error) {
	exec := e.cfg.ExecName("evm")
	execty := ctypes.LoadExecutorType(exec)
	if execty == nil {
		return nil, ctypes.ErrNotSupport
	}

	var p rpctypes.Query4Jrpc
	p.Execer = exec
	p.FuncName = "GetCode"
	p.Payload = []byte(fmt.Sprintf(`{"addr":"%v"}`, addr.String()))
	queryparam, err := execty.CreateQuery(p.FuncName, p.Payload)
	if err != nil {
		return nil, err
	}
	resp, err := e.cli.Query(p.Execer, p.FuncName, queryparam)
	if err != nil {
		return nil, err
	}

	result, err := execty.QueryToJSON(p.FuncName, resp)
	if err != nil {
		return nil, err
	}
	log.Info("GetCode", "resp", string(result))
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
