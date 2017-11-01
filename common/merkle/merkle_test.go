package merkle

import (
	"testing"

	"code.aliyun.com/chain33/chain33/common"
)

func TestMerkleRoot(t *testing.T) {
	RootHash := "0x5140e5972f672bf8e81bc189894c55a410723b095716eaeec845490aed785f0e"
	tx0string := "0xb86f5ef1da8ddbdb29ec269b535810ee61289eeac7bf2b2523b494551f03897c"
	tx1string := "0x80c6f121c3e9fe0a59177e49874d8c703cbadee0700a782e4002e87d862373c6"

	var hashlist [][]byte

	hashlist = append(hashlist, reverse(common.FromHex(tx0string)))
	hashlist = append(hashlist, reverse(common.FromHex(tx1string)))

	data, mutated := CalcMerkle(hashlist)
	t.Log(mutated)
	calcroot := common.ToHex(reverse(data))
	if calcroot != RootHash {
		t.Error("root not match")
		t.Log(calcroot)
	}
}

func reverse(r []byte) []byte {
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return r
}
