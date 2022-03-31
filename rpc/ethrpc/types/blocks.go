package types

import (
	"fmt"
	"github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

func BlockDetailToEthBlock(details *types.BlockDetails,cfg *types.Chain33Config )(*Block,error){
	var block Block
	var header Header
	fullblock:=details.GetItems()[0]
	header.Time= hexutil.Uint(fullblock.GetBlock().GetBlockTime()).String()
	header.Number=hexutil.Uint(fullblock.GetBlock().Height).String()
	header.TxHash=common.BytesToHash(fullblock.GetBlock().GetHeader(cfg).TxHash).Hex()
	header.Difficulty=hexutil.Uint(int64(fullblock.GetBlock().GetDifficulty())).String()
	header.ParentHash=common.BytesToHash(fullblock.GetBlock().ParentHash).Hex()
	header.Root=common.BytesToHash(fullblock.GetBlock().GetStateHash()).Hex()
	header.Coinbase=fullblock.GetBlock().GetTxs()[0].From()
	//暂不支持ReceiptHash,UncleHash
	//header.ReceiptHash=
	//header.UncleHash

	//处理交易
	//采用BTY 默认的chainID =0如果要采用ETH的默认chainID=1,则为1
	eipSigner:= etypes.NewEIP155Signer(big.NewInt(int64(cfg.GetChainID())))
	var tx Transaction
	var txs Transactions

	tx.Type= fmt.Sprint(etypes.LegacyTxType)
	for _,itx:=range fullblock.GetBlock().GetTxs(){
		tx.To=itx.To
		amount,_:=itx.Amount()
		tx.Value="0x"+common.Bytes2Hex(big.NewInt(amount).Bytes())
		r,s,v ,err:= MakeDERSigToRSV(eipSigner,itx.Signature.GetSignature())
		if err!=nil{
			//log.Error("makeDERSigToRSV","err",err)
			return nil,err
		}
		tx.V=hexutil.EncodeBig(v)
		tx.R=hexutil.EncodeBig(r)
		tx.S=hexutil.EncodeBig(s)
		txs=append(txs,&tx)
	}
	block.Header=&header
	block.Transactions=txs
	block.Hash=common.BytesToHash(fullblock.GetBlock().Hash(cfg)).Hex()
	return &block,nil
}