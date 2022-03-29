package eth_rpc

import (
	"github.com/33cn/chain33/rpc/eth_rpc/types"
	"github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
)

func Test_paraseDERCode(t *testing.T){
	var sigstr ="3044022040fedc49aadbdcf0cb14f42ac1b0a62d8e10e8cfadabb668017d724a7218c4ed02202c94fbb4ff94e1e52639c08acdf136e081e17b2f64609b27dc07d83c98e72de0"
	eipSigner:= etypes.NewEIP155Signer(big.NewInt(int64(0)))
	r,s,v,err:= types.MakeDERSigToRSV(eipSigner,common.FromHex(sigstr))
	if err!=nil{
		t.Log("err:",err)
		return
	}
	t.Log("r:",common.Bytes2Hex(r.Bytes()))
	t.Log("s:",common.Bytes2Hex(s.Bytes()),"size:",len(s.Bytes()))
	t.Log("v",common.Bytes2Hex(v.Bytes()))

}