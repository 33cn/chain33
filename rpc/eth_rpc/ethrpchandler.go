package eth_rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	"github.com/33cn/chain33/rpc/eth_rpc/types"
	rpctypes "github.com/33cn/chain33/rpc/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
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

/**
params: [
   '0x407d73d8a49eeb85d32cf465507dd71d507100c1',
   'latest'
]

return: {
  "id":1,
  "jsonrpc": "2.0",
  "result": "0x0234c8a3397aab58" // 158972490234375000
}
*/
//GetBalance eth_getBalance  tag:"latest", "earliest" or "pending"
func (e *EthApi) GetBalance(address string, tag *string) ( string,  error) {
	if !common.IsHexAddress(address) {
		return "", Err_AddressFormat
	}
	var req ctypes.ReqBalance
	req.AssetSymbol=e.cfg.GetCoinSymbol()
	req.Execer=e.cfg.GetCoinExec()
	req.Addresses=append(req.GetAddresses(),address)
	log.Debug("GetBalance","execer:",req.Execer,"assertsymbol:",req.AssetSymbol)
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
func (e*EthApi)GetBlockByNumber(number uint64,full bool ) *types.Block{
	var req ctypes.ReqBlocks
	req.Start= int64(number)
	req.End= int64(number)
	req.IsDetail=full
	details,err:= e.cli.GetBlocks(&req)
	if err!=nil{
		log.Error("GetBlockByNumber","err",err)
		return nil
	}

	var block types.Block
	var header types.Header
	fullblock:=details.GetItems()[0]
	header.Time= uint64(fullblock.GetBlock().GetBlockTime())
	header.Number=big.NewInt(fullblock.GetBlock().Height)
	header.TxHash=common.BytesToHash(fullblock.GetBlock().GetHeader(e.cfg).TxHash).Hex()
	header.Difficulty=big.NewInt(int64(fullblock.GetBlock().GetDifficulty()))
	header.ParentHash=common.BytesToHash(fullblock.GetBlock().ParentHash).Hex()
	header.Root=common.BytesToHash(fullblock.GetBlock().GetStateHash()).Hex()
	header.Coinbase=fullblock.GetBlock().GetTxs()[0].From()
	//暂不支持ReceiptHash,UncleHash
	//header.ReceiptHash=
	//header.UncleHash

	//处理交易
	//采用BTY 默认的chainID =0如果要采用ETH的默认chainID=1,则为1
	eipSigner:= etypes.NewEIP155Signer(big.NewInt(int64(e.cfg.GetChainID())))
	var tx types.Transaction
	var txs types.Transactions
	tx.Type= fmt.Sprint(etypes.LegacyTxType)
	for _,itx:=range fullblock.GetBlock().GetTxs(){
		tx.To=itx.To
		amount,_:=itx.Amount()
		tx.Value="0x"+common.Bytes2Hex(big.NewInt(amount).Bytes())
		r,s,v ,err:= types.MakeDERSigToRSV(eipSigner,itx.Signature.GetSignature())
		if err!=nil{
			log.Error("makeDERSigToRSV","err",err)
			return nil
		}
		tx.V=v
		tx.R=r
		tx.S=s
		txs=append(txs,&tx)
	}
	block.Header=&header
	block.Transactions=txs
	block.Hash=common.BytesToHash(fullblock.GetBlock().Hash(e.cfg)).Hex()
	return &block
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





//eth_accounts
func(e *EthApi)Accounts()(*rpctypes.WalletAccounts,error){
	req := &ctypes.ReqAccountList{WithoutBalance: true}
	msg,err:=e.cli.ExecWalletFunc("wallet","WalletGetAccountList",req)
	if err!=nil{
		return nil,err
	}
	accountsList := msg.(*ctypes.WalletAccounts)
	var accounts rpctypes.WalletAccounts
	for _, wallet := range accountsList.Wallets {
		accounts.Wallets = append(accounts.Wallets, &rpctypes.WalletAccount{Label: wallet.GetLabel(),
			Acc: &rpctypes.Account{Currency: wallet.GetAcc().GetCurrency(), Balance: wallet.GetAcc().GetBalance(),
				Frozen: wallet.GetAcc().GetFrozen(), Addr: wallet.GetAcc().GetAddr()}})
	}

	return &accounts,nil

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



//
