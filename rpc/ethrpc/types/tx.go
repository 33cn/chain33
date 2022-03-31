package types

import (
	"errors"
	"fmt"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

func DecodeSignature(sig []byte) (r, s, v *big.Int) {
	if len(sig) != crypto.SignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v
}
func MakeDERSigToRSV(eipSigner etypes.EIP155Signer,sig []byte)(r,s,v *big.Int,err error){
	rb,sb,err := paraseDERCode(sig)
	if err!=nil{
		fmt.Println("MakeDERSigToRSV","err",err.Error())
		return nil,nil,nil,err
	}
	var signature []byte
	signature=append(signature,rb...)
	signature=append(signature,sb...)
	signature=append(signature,0x00)
	r,s,v =  DecodeSignature(signature)
	if eipSigner.ChainID().Sign() != 0 {
		v = big.NewInt(int64(signature[64] + 35))
		v.Add(v, new(big.Int).Mul(eipSigner.ChainID(), big.NewInt(2)))
	}
	return r,s,v,nil
}

func  paraseDERCode(sig []byte)(r,s []byte,err error){
	//0x30 [total-length] 0x02 [R-length] [R] 0x02 [S-length] [S]

	if len(sig)<70{

		return nil,nil,errors.New(fmt.Sprintf("wrong sig data size:%v,must beyound length 70 bytes",len(sig)))
	}

	fmt.Println("sig hex",common.Bytes2Hex(sig))
	//3045022100af5778b81ae8817c6ae29fad8c1113d501e521c885a65c2c4d71763c4963984b022020687b73f5c90243dc16c99427d6593a711c52c8bf09ca6331cdd42c66edee74
	if sig[0]==0x30 &&sig[2]==0x02{
		r=sig[4:int(sig[3])+4]
		if r[0]==0x0{
			r=r[1:]
		}
	}
	if sig[int(sig[3])+4]==0x02{//&&sig[int(sig[3])+5]==0x20
		s=sig[int(sig[3])+6:int(sig[3])+6+int(sig[int(sig[3])+5])]
	}

	return
}

func TxDetailsToEthTx(txdetails *ctypes.TransactionDetails,cfg *ctypes.Chain33Config )(txs Transactions, err error){
	for _,detail:=range txdetails.GetTxs(){
		var tx Transaction
		tx.To= detail.Tx.To
		tx.From=detail.Fromaddr
		tx.Value= fmt.Sprintf("0x%x",detail.GetAmount())//big.NewInt(detail.GetAmount()).
		tx.Type=fmt.Sprint(detail.Receipt.Ty)
		tx.BlockNumber=fmt.Sprintf("0x%x",detail.Height)
		tx.TransactionIndex=fmt.Sprintf("0x%x",detail.GetIndex())
		eipSigner:= etypes.NewEIP155Signer(big.NewInt(int64(cfg.GetChainID())))
		r,s,v,err:=MakeDERSigToRSV(eipSigner,detail.Tx.GetSignature().GetSignature())
		if err!=nil{
			return nil,err
		}
		tx.V=hexutil.EncodeBig(v)
		tx.R=hexutil.EncodeBig(r)
		tx.S=hexutil.EncodeBig(s)
		txs=append(txs,&tx)
	}
	return

}


func TxDetailsToEthReceipt(txdetails *ctypes.TransactionDetails,cfg *ctypes.Chain33Config )([]*Receipt,  error){
	var receipts []*Receipt
	for _,detail:=range txdetails.GetTxs(){
		var receipt Receipt
		receipt.To= detail.Tx.To
		receipt.From=detail.Fromaddr
		if detail.Receipt.Ty==2{
			receipt.Type="0x1"
		}else{
			receipt.Type="0x2"
		}
		receipt.TxHash=fmt.Sprintf("0x%x",detail.GetFullHash())
		receipt.BlockNumber=fmt.Sprintf("0x%x",detail.Height)
		receipt.TransactionIndex=fmt.Sprintf("0x%x",detail.GetIndex())
		if len(txdetails.GetTxs()[0].Tx.Payload)!=0{
			receipt.ContractAddress=detail.Tx.To
			//TODO 解析payload 提取to地址
			//receipt.To=
		}
		receipts=append(receipts,&receipt)
	}
	return receipts ,nil

}