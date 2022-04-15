package eth

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/33cn/chain33/client"
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
	cli rpcclient.ChannelClient
	cfg *ctypes.Chain33Config
}

var (
	log = log15.New("module", "eth")
)

//NewEthApi new eth api
func NewEthAPI(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) interface{} {
	e := &ethHandler{}
	e.cli.Init(c, api)
	e.cfg = cfg
	return e
}

//GetBalance eth_getBalance  tag:"latest", "earliest" or "pending"
func (e *ethHandler) GetBalance(address string, tag *string) (hexutil.Uint64, error) {
	var req ctypes.ReqBalance
	req.AssetSymbol = e.cli.GetConfig().GetCoinSymbol()
	req.Execer = e.cli.GetConfig().GetCoinExec()
	req.Addresses = append(req.GetAddresses(), address)
	accounts, err := e.cli.GetBalance(&req)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(accounts[0].GetBalance()), nil

}

//ChainId eth_chainId
func (e *ethHandler) ChainId() (hexutil.Uint64, error) {
	return hexutil.Uint64(e.cfg.GetChainID()), nil
}

//BlockNumber eth_blockNumber 获取区块高度
func (e *ethHandler) BlockNumber() (hexutil.Uint64, error) {
	header, err := e.cli.GetLastHeader()
	if err != nil {
		return 0, err
	}

	return hexutil.Uint64(header.Height), nil
}

//GetBlockByNumber  eth_getBlockByNumber
func (e *ethHandler) GetBlockByNumber(number *hexutil.Big, full bool) (*types.Block, error) {
	var req ctypes.ReqBlocks
	req.Start = number.ToInt().Int64()
	req.End = req.Start
	req.IsDetail = full
	log.Debug("GetBlockByNumber", "start", req.Start)
	details, err := e.cli.GetBlocks(&req)
	if err != nil {
		log.Error("GetBlockByNumber", "err", err)
		return nil, err
	}

	fullblock := details.GetItems()[0]
	block, err := types.BlockDetailToEthBlock(&ctypes.BlockDetails{
		Items: []*ctypes.BlockDetail{fullblock},
	}, e.cfg, full)
	if err != nil {
		return nil, err
	}
	return block, nil
}

//GetBlockByHash eth_getBlockByHash 通过区块哈希获取区块交易详情
func (e *ethHandler) GetBlockByHash(txhash string, full bool) (*types.Block, error) {
	var req ctypes.ReqHashes
	req.Hashes = append(req.Hashes, common.FromHex(txhash))
	details, err := e.cli.GetBlockByHashes(&req)
	if err != nil {
		log.Error("GetBlockByNumber", "err", err)
		return nil, err
	}
	return types.BlockDetailToEthBlock(details, e.cfg, full)

}

//GetTransactionByHash eth_getTransactionByHash
func (e *ethHandler) GetTransactionByHash(txhash string) (*types.Transaction, error) {
	var req ctypes.ReqHashes
	req.Hashes = append(req.Hashes, common.FromHex(txhash))
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
func (e *ethHandler) GetTransactionReceipt(txhash string) (*types.Receipt, error) {
	var req ctypes.ReqHashes
	req.Hashes = append(req.Hashes, common.FromHex(txhash))
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
func (e *ethHandler) GetBlockTransactionCountByHash(hash string) (hexutil.Uint64, error) {
	var req ctypes.ReqHashes
	req.Hashes = append(req.Hashes, common.FromHex(hash))
	blockdetails, err := e.cli.GetBlockByHashes(&req)
	if err != nil {
		log.Error("GetBlockByNumber", "err", err)
		return 0, err
	}

	return hexutil.Uint64(len(blockdetails.GetItems()[0].GetBlock().GetTxs())), nil
}

//Accounts eth_accounts
func (e *ethHandler) Accounts() ([]string, error) {
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
	var param rpctypes.Query4Jrpc
	type EvmQueryReq struct {
		Address string
		Input   string
	}

	//暂定evm
	param.Execer = "evm"
	param.FuncName = "Query"
	param.Payload = []byte(fmt.Sprintf(`{"input:%v","address":"%s"}`, msg.Data, msg.To))
	log.Debug("eth_call", "QueryCall param", param, "payload", string(param.Payload))
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

	log.Debug("Call", "QueryCall resp", resp)
	return execty.QueryToJSON(param.FuncName, resp)

}

//SendRawTransaction eth_sendRawTransaction 发送交易
func (e *ethHandler) SendRawTransaction(rawData string) (string, error) {
	hexData := common.FromHex(rawData)
	if hexData == nil {
		return "", errors.New("wrong data")
	}
	var parm ctypes.Transaction
	//按照Chain33交易格式进行解析
	err := ctypes.Decode(hexData, &parm)
	if err != nil {
		log.Error("SendRawTransaction", "param", parm.String(), "err", err.Error())
		return "", err
	}
	log.Debug("SendTransaction", "param", parm.String())
	reply, err := e.cli.SendTx(&parm)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(reply.GetMsg()), nil
}

//Sign method:eth_sign
func (e *ethHandler) Sign(address, message string) (string, error) {
	//导出私钥
	reply, err := e.cli.ExecWalletFunc("wallet", "DumpPrivkey", &ctypes.ReqString{Data: address})
	if err != nil {
		log.Error("SignWalletRecoverTx", "execWalletFunc err", err)
		return "", err
	}
	privKeyHex := reply.(*ctypes.ReplyString).GetData()
	msg := common.FromHex(message)
	if len(msg) == 0 {
		return "", errors.New("invalid argument 1: must hex string")
	}

	c, _ := crypto.Load("secp256k1", -1)
	signKey, err := c.PrivKeyFromBytes(common.FromHex(privKeyHex))
	if err != nil {
		return "", err
	}
	return hexutil.Encode(signKey.Sign(msg).Bytes()), nil
}

//SignTransaction method:eth_signTransaction
func (e *ethHandler) SignTransaction(msg *types.CallMsg) (string, error) {
	var tx *ctypes.Transaction
	var data []byte
	if msg.Data == "" {
		//普通的coins 转账
		exec := e.cfg.ExecName("coins") //e.cfg.GetParaName() +"coins"
		v := &dtypes.CoinsAction_Transfer{Transfer: &ctypes.AssetsTransfer{Cointoken: e.cfg.GetCoinSymbol(), Amount: msg.Value.ToInt().Int64(), Note: []byte("")}}
		transfer := &dtypes.CoinsAction{Value: v, Ty: dtypes.CoinsActionTransfer}
		data = ctypes.Encode(transfer)
		tx = &ctypes.Transaction{Execer: []byte(exec), Payload: data, Fee: 0, To: msg.To, Nonce: rand.New(rand.NewSource(time.Now().UnixNano())).Int63()}
	} else {
		//evm tx
		action := ctypes.EVMContractAction4Chain33{
			Amount:       msg.Value.ToInt().Uint64(),
			GasLimit:     0,
			GasPrice:     0,
			Code:         nil,
			Para:         common.FromHex(msg.Data),
			Alias:        "",
			Note:         "",
			ContractAddr: msg.To,
		}

		exec := e.cfg.ExecName("evm") // e.cfg.GetParaName() +"evm"
		tx = &ctypes.Transaction{Execer: []byte(exec), Payload: ctypes.Encode(&action), Fee: 0, To: address.ExecAddress(exec)}
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
func (e *ethHandler) GetTransactionCount(address, tag string) (interface{}, error) {
	exec := e.cfg.ExecName("evm")
	execty := ctypes.LoadExecutorType(exec)
	if execty == nil {
		return "", ctypes.ErrNotSupport
	}
	var param rpctypes.Query4Jrpc
	param.FuncName = "GetNonce"
	param.Execer = exec
	param.Payload = []byte(fmt.Sprintf(`{"address":"%v"}`, address))
	queryparam, err := execty.CreateQuery(param.FuncName, param.Payload)
	if err != nil {
		return nil, err
	}
	resp, err := e.cli.Query(param.Execer, param.FuncName, queryparam)
	if err != nil {
		return "", err
	}

	return execty.QueryToJSON(param.FuncName, resp)
}

//method:eth_estimateGas
//EstimateGas 获取gas
func (e *ethHandler) EstimateGas(callMsg *types.CallMsg) (interface{}, error) {
	//组装tx
	exec := e.cfg.ExecName("evm")
	execty := ctypes.LoadExecutorType(exec)
	if execty == nil {
		return "", ctypes.ErrNotSupport
	}

	action := &ctypes.EVMContractAction4Chain33{Amount: callMsg.Value.ToInt().Uint64(), GasLimit: 0, GasPrice: 0, Note: "", Para: common.FromHex(callMsg.Data), ContractAddr: callMsg.To}
	tx := &ctypes.Transaction{Execer: []byte(exec), Payload: ctypes.Encode(action), To: address.ExecAddress(exec), ChainID: e.cfg.GetChainID()}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	var p rpctypes.Query4Jrpc
	p.Execer = exec
	p.FuncName = "EstimateGas"
	p.Payload = []byte(fmt.Sprintf(`{"tx":"%v","from":"%v"}`, common.Bytes2Hex(ctypes.Encode(tx)), callMsg.From))
	queryparam, err := execty.CreateQuery(p.FuncName, p.Payload)
	if err != nil {
		return nil, err
	}
	resp, err := e.cli.Query(p.Execer, p.FuncName, queryparam)
	if err != nil {
		return "", err
	}

	return execty.QueryToJSON(p.FuncName, resp)
}
