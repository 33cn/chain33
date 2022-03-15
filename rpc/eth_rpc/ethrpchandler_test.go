package eth_rpc

import (
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func Test_paraseDERCode(t *testing.T){
	var sigstr ="3044022040fedc49aadbdcf0cb14f42ac1b0a62d8e10e8cfadabb668017d724a7218c4ed02202c94fbb4ff94e1e52639c08acdf136e081e17b2f64609b27dc07d83c98e72de0"
	r,s,err:= paraseDERCode(common.FromHex(sigstr))
	if err!=nil{
		t.Log("err:",err)
		return
	}
	t.Log("r:",common.Bytes2Hex(r))
	t.Log("s:",common.Bytes2Hex(s),"size:",len(s))

}