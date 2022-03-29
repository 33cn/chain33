package types

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func Test_parseDer(t *testing.T){
	var testSig="3045022100af5778b81ae8817c6ae29fad8c1113d501e521c885a65c2c4d71763c4963984b022020687b73f5c90243dc16c99427d6593a711c52c8bf09ca6331cdd42c66edee74"
	sig:=common.FromHex(testSig)
	t.Log("size sig:",len(sig))
	r,s,err:= paraseDERCode(sig)
	if err!=nil{
		t.Log("err",err)
		return
	}
	t.Log("r:",common.Bytes2Hex(r),"size:",len(r))
	jmstr,_:=json.Marshal(big.NewInt(1).SetBytes(r))
	t.Log("jmstr",string(jmstr))
	t.Log("s:",common.Bytes2Hex(s),"size:",len(s))
}
