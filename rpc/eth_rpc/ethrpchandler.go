package eth_rpc

import (
	"fmt"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	"github.com/33cn/chain33/rpc/eth_rpc/types"
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

//eth_getTransactionByHash

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
func(e *EthApi)GetTransactionReceipt(txhash string){
	var req  ctypes.ReqHashes
	req.Hashes=append(req.Hashes,common.FromHex(txhash))
	txdetails,err:=e.cli.GetTransactionByHash(&req)
	if err!=nil{
		return nil,err
	}

}



