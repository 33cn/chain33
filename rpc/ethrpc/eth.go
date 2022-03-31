package ethrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	"github.com/33cn/chain33/rpc/ethrpc/types"
	rpctypes "github.com/33cn/chain33/rpc/types"
	dtypes "github.com/33cn/chain33/system/dapp/coins/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"math/rand"
	"time"
)
type EthApi struct {
	cli rpcclient.ChannelClient
	cfg *ctypes.Chain33Config

}

func NewEthApi( cfg *ctypes.Chain33Config,c queue.Client,api client.QueueProtocolAPI) *EthApi {
	e:=&EthApi{}
	e.cli.Init(c,api)
	e.cfg=cfg
	return e
}


//GetBalance eth_getBalance  tag:"latest", "earliest" or "pending"
func (e *EthApi) GetBalance(address string, tag *string) ( string,  error) {
	var req ctypes.ReqBalance
	req.AssetSymbol=e.cfg.GetCoinSymbol()
	req.Execer=e.cfg.GetCoinExec()
	req.Addresses=append(req.GetAddresses(),address)
	accounts,err:=e.cli.GetBalance(&req)
	if err!=nil{
		return "",err
	}
	bf:=big.NewInt(accounts[0].GetBalance())
	return "0x"+common.Bytes2Hex(bf.Bytes()),nil
}

//ChainId eth_chainId
func (e *EthApi) ChainId() (string, error) {
	return fmt.Sprintf("0x%x",e.cfg.GetChainID()),nil
}

//BlockNumber eth_blockNumber 获取区块高度
func (e *EthApi) BlockNumber() ( string ,  error) {
	header,err:=e.cli.GetLastHeader()
	if err != nil {
		return "",err
	}

	bf:=big.NewInt(header.Height)
	return "0x"+common.Bytes2Hex(bf.Bytes()),nil

}

//GetBlockByNumber  eth_getBlockByNumber
func (e*EthApi)GetBlockByNumber(number string,full bool ) (*types.Block,error){
	nb:=common.FromHex(number)
	if nb==nil{
		return nil ,errors.New("invalid argument 0")
	}
	bn:=big.NewInt(1).SetBytes(nb)
	var req ctypes.ReqBlocks
	req.Start= bn.Int64()
	req.End= bn.Int64()
	req.IsDetail=full
	log.Info("GetBlockByNumber","start",req.Start)
	details,err:= e.cli.GetBlocks(&req)
	if err!=nil{
		log.Error("GetBlockByNumber","err",err)
		return nil,err
	}

	var block types.Block
	var header types.Header
	fullblock:=details.GetItems()[0]
	header.Time= hexutil.Uint(fullblock.GetBlock().GetBlockTime()).String()
	header.Number=hexutil.Uint(fullblock.GetBlock().Height).String() //big.NewInt(fullblock.GetBlock().Height)
	header.TxHash=common.BytesToHash(fullblock.GetBlock().GetHeader(e.cfg).TxHash).Hex()
	header.Difficulty=hexutil.Uint(fullblock.GetBlock().GetDifficulty()).String() //big.NewInt(int64(fullblock.GetBlock().GetDifficulty()))
	header.ParentHash=common.BytesToHash(fullblock.GetBlock().ParentHash).Hex()
	header.Root=common.BytesToHash(fullblock.GetBlock().GetStateHash()).Hex()
	header.Coinbase=fullblock.GetBlock().GetTxs()[0].From()
	//暂不支持ReceiptHash,UncleHash
	//header.ReceiptHash=
	//header.UncleHash

	//处理交易
	//采用BTY 默认的chainID =0如果要采用ETH的默认chainID=1,则为1
	eipSigner:= etypes.NewEIP155Signer(big.NewInt(int64(e.cfg.GetChainID())))

	var txs types.Transactions

	ftxs:=fullblock.GetBlock().GetTxs()
	for _,itx:=range ftxs{
		var tx types.Transaction
		tx.Type= fmt.Sprint(etypes.LegacyTxType)
		tx.From=itx.From()
		tx.To=itx.To
		amount,err:=itx.Amount()
		if err!=nil{
			return nil,err
		}
		tx.Value="0x"+common.Bytes2Hex(big.NewInt(amount).Bytes())
		r,s,v ,err:= types.MakeDERSigToRSV(eipSigner,itx.Signature.GetSignature())
		if err!=nil{
			log.Error("makeDERSigToRSV","err",err)
			//return nil,err
			continue
		}

		tx.V=hexutil.EncodeBig(v)
		tx.R=hexutil.EncodeBig(r)
		tx.S=hexutil.EncodeBig(s)
		//log.Info("r:",common.Bytes2Hex(tx.R.Bytes()))
		txs=append(txs,&tx)
	}
	block.Header=&header
	block.Transactions=txs
	block.Hash=common.BytesToHash(fullblock.GetBlock().Hash(e.cfg)).Hex()

	return &block,nil
}


//GetBlockByHash eth_getBlockByHash 通过区块哈希获取区块交易详情
func(e*EthApi) GetBlockByHash(txhash string ,full bool ) (*types.Block,error){
	var req ctypes.ReqHashes
	req.Hashes=append(req.Hashes,common.FromHex(txhash))
	details,err:= e.cli.GetBlockByHashes(&req)
	if err!=nil{
		log.Error("GetBlockByNumber","err",err)
		return nil, err
	}
	return types.BlockDetailToEthBlock(details,e.cfg)

}

//GetTransactionByHash eth_getTransactionByHash
func(e *EthApi)GetTransactionByHash(txhash string)(*types.Transaction,error){
	var req  ctypes.ReqHashes
	req.Hashes=append(req.Hashes,common.FromHex(txhash))
	txdetails,err:=e.cli.GetTransactionByHash(&req)
	if err!=nil{
		return nil,err
	}
	txs,err:= types.TxDetailsToEthTx(txdetails,e.cfg)
	if err!=nil{
		return nil,err
	}
	return txs[0],nil
}


//GetTransactionReceipt eth_getTransactionReceipt
func(e *EthApi)GetTransactionReceipt(txhash string)(*types.Receipt,error){
	var req  ctypes.ReqHashes
	req.Hashes=append(req.Hashes,common.FromHex(txhash))
	txdetails,err:=e.cli.GetTransactionByHash(&req)
	if err!=nil{
		return nil,err
	}
	receipts,err:= types.TxDetailsToEthReceipt(txdetails,e.cfg)
	if err!=nil{
		return nil,err
	}
	return receipts[0],nil

}
//GetBlockTransactionCountByNumber eth_getBlockTransactionCountByNumber
func(e *EthApi)GetBlockTransactionCountByNumber(blockNum string )(string,error){
	var blockHeight int64
	if common.FromHex(blockNum)!=nil{
		bn:=big.NewInt(1)
		var ok bool
		bn,ok=bn.SetString(blockNum,16)
		if !ok{
			return "",errors.New("wrong param")
		}
		blockHeight = bn.Int64()

	}else{
		bn,ok:= big.NewInt(1).SetString(blockNum,10)
		if !ok{
			return "",errors.New("wrong param")
		}
		blockHeight = bn.Int64()
	}
	var req ctypes.ReqBlocks
	req.Start= blockHeight
	req.End= blockHeight
	blockdetails,err:= e.cli.GetBlocks(&req)
	if err!=nil{
		return "",err
	}
	return fmt.Sprintf("0x%x",len(blockdetails.GetItems()[0].Block.GetTxs())),nil

}

//GetBlockTransactionCountByHash
//method:eth_getBlockTransactionCountByHash
//parameters: 32 Bytes - hash of a block
//Returns: integer of the number of transactions in this block.
func (e *EthApi)GetBlockTransactionCountByHash(hash string)(string,error){
	var req ctypes.ReqHashes
	req.Hashes=append(req.Hashes,common.FromHex(hash))
	details,err:= e.cli.GetBlockByHashes(&req)
	if err!=nil{
		log.Error("GetBlockByNumber","err",err)
		return "", err
	}
	txNum:=len(details.GetItems()[0].GetBlock().GetTxs())
	bn:=big.NewInt(int64(txNum))
	return "0x"+common.Bytes2Hex(bn.Bytes()),nil
}



//eth_accounts
func(e *EthApi)Accounts()([]string ,error){
	req := &ctypes.ReqAccountList{WithoutBalance: true}
	msg,err:=e.cli.ExecWalletFunc("wallet","WalletGetAccountList",req)
	if err!=nil{
		return nil,err
	}
	accountsList := msg.(*ctypes.WalletAccounts)
	var accounts []string
	for _, wallet := range accountsList.Wallets {
		accounts=append(accounts,wallet.GetAcc().GetAddr())
	}

	return accounts,nil

}

//eth_call evm合约相关操作
func(e *EthApi)Call(msg types.CallMsg,tag *string )(interface{},error){
	var param rpctypes.Query4Jrpc
	type EvmQueryReq struct {
		Address              string
		Input                string
	}

	param.Execer = e.cfg.GetCoinExec()
	param.FuncName = "Query"
	jsonData:=fmt.Sprintf(`{"input:%v","address":"%s"}`,msg.Data,msg.To)
	param.Payload = []byte(jsonData)
	log.Info("eth_call", "QueryCall param", param, "payload", string(param.Payload))

	execty := ctypes.LoadExecutorType(param.Execer)
	if execty == nil {
		log.Error("Query", "funcname", param.FuncName, "err", ctypes.ErrNotSupport)
		return nil,ctypes.ErrNotSupport
	}
	decodePayload, err := execty.CreateQuery(param.FuncName, param.Payload)
	if err != nil {
		log.Error("EventQuery1", "err", err.Error(), "funcName", param.FuncName)
		return nil,err
	}

	resp,err := e.cli.Query(e.cfg.ExecName(param.Execer), 	param.FuncName , decodePayload)
	if err != nil {
		log.Error("eth_call", "error", err)
		return nil, err
	}

	log.Debug("eth_call", "QueryCall resp", resp)
	var jsonmsg json.RawMessage
	jsonmsg, err = execty.QueryToJSON(param.FuncName, resp)
	return jsonmsg,nil
}



//SendRawTransaction eth_sendRawTransaction 发送交易
func(e *EthApi)SendRawTransaction(rawData string )(string,error){
	hexData:= common.FromHex(rawData)
	if hexData==nil{
		return "",errors.New("wrong data")
	}
	var parm ctypes.Transaction
	//暂按照Chain33交易格式进行解析
	err := ctypes.Decode(hexData, &parm)
	if err != nil {
		return "",err
	}
	log.Debug("SendTransaction", "param", parm.String())
	reply,err:= e.cli.SendTx(&parm)
	if err != nil {
		return "",err
	}

	return  "0x"+common.Bytes2Hex(reply.GetMsg()),nil

}


//TODO 后面实现
//Sign
//method:eth_sign
func(e *EthApi)Sign(address,message string)(string,error){
	//导出私钥
	reply, err := e.cli.ExecWalletFunc("wallet", "DumpPrivkey", &ctypes.ReqString{Data: address})
	if err != nil {
		log.Error("SignWalletRecoverTx", "execWalletFunc err", err)
		return "", err
	}
	privKeyHex := reply.(*ctypes.ReplyString).GetData()

	msg:= common.FromHex(message)
	if len(msg)==0{
		return "",errors.New("invalid argument 1: must hex string")
	}

	c, err := crypto.Load("secp256k1", -1)
	signKey, err := c.PrivKeyFromBytes(common.FromHex(privKeyHex))
	if err!=nil{
		return "",err
	}

	return "0x"+common.Bytes2Hex(signKey.Sign(msg).Bytes()),nil
}



//SignTransaction
//method:eth_signTransaction
func(e *EthApi)SignTransaction(msg *types.CallMsg)(string,error){
	//把
	var tx *ctypes.Transaction
	var data []byte
	if msg.Data==""{
		//普通的coins 转账
		v := &dtypes.CoinsAction_Transfer{Transfer: &ctypes.AssetsTransfer{Cointoken: e.cfg.GetCoinSymbol(), Amount: msg.Value.ToInt().Int64(), Note: []byte("")}}
		transfer := &dtypes.CoinsAction{Value: v, Ty: dtypes.CoinsActionTransfer}
		data=ctypes.Encode(transfer)
		tx = &ctypes.Transaction{Execer: []byte(e.cfg.GetCoinExec()), Payload: data, Fee: 1e5, To: msg.To, Nonce: rand.New(rand.NewSource(time.Now().UnixNano())).Int63()}
	}else{
		//evm tx
		//TOOD 暂时不支持，因为不知道evm具体数据要执行何种执行
		return "",errors.New("no support unpack data")
	}

	//对TX 进行签名
	unsigned := &ctypes.ReqSignRawTx{
		Addr:   msg.From,
		TxHex:  common.Bytes2Hex(ctypes.Encode(tx)),
		Expire: "0",
	}
	signedTx,err:= e.cli.ExecWalletFunc("wallet","SignRawTx",unsigned)
	if err!=nil{
		return "",err
	}

	return signedTx.(*ctypes.ReplySignRawTx).TxHex,nil

}

//Syncing :Returns an object with data about the sync status or false.
//Returns: FALSE:when not syncing,
//method:eth_syncing
//params:[]
func (e *EthApi)Syncing()(interface{},error){
	var syncing struct{
		StartingBlock string `json:"startingBlock,omitempty"`
		CurrentBlock string `json:"currentBlock,omitempty"`
		HighestBlock string `json:"highestBlock,omitempty"`
	}
	reply,err:= e.cli.IsSync()
	if err ==nil{
		var caughtUp ctypes.IsCaughtUp
		err= ctypes.Decode(reply.GetMsg(),&caughtUp)
		if err==nil{
			if caughtUp.Iscaughtup{// when not syncing
				return false,nil
			}else{//when suncing
				header,err:= e.cli.GetLastHeader()
				if err==nil{
					bn:=big.NewInt(header.Height)
					syncing.CurrentBlock="0x"+common.Bytes2Hex(bn.Bytes())
					syncing.StartingBlock=syncing.CurrentBlock
					replyBlockNum,err:= e.cli.GetHighestBlockNum(&ctypes.ReqNil{})
					if err==nil{
						bn= big.NewInt(replyBlockNum.GetHeight())
						syncing.HighestBlock="0x"+common.Bytes2Hex(bn.Bytes())
						return &syncing,nil
					}

				}

			}
		}
	}

	return nil,err
}

//GasPrice
//method:eth_gasPrice
//Parameters: none
//Returns:Returns the current price per gas in wei.
func (e *EthApi)GasPrice()(interface{},error){
	//TODO 支持gasprice的获取
	return nil,errors.New("no support")
}

//Mining
//method:eth_mining
//Paramtesrs:none
//Returns:Returns true if client is actively mining new blocks.
func (e *EthApi)Mining()(bool,error){

	msg,err:=e.cli.ExecWalletFunc("wallet","GetWalletStatus",&ctypes.ReqNil{})
	if err==nil{
		status:=msg.(*ctypes.WalletStatus)
		if status.IsAutoMining{
			return true ,nil
		}
		return false ,nil
	}
	return false,err
}

//GetTransactionCount
//methpd:eth_getTransactionCount
//Returns:Returns the number of transactions sent from an address.
//Paramters: address,tag(disable):latest,pending,earliest
func (e *EthApi)GetTransactionCount(address ,tag string)(string,error){
	return "0x0",nil
}





