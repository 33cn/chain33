package account

//package for account manger
//1. load from db
//2. save to db
//3. KVSet
//4. Transfer
//5. Add
//6. Sub
//7. Account balance query
//8. gen a private key -> private key to address (bitcoin likes)

import (
	"math/big"

	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
)

func LoadAccount(db dbm.KVDB, addr []byte) (*types.Account, error) {
	return nil, nil
}

func SaveAccount(db dbm.KVDB, acc *types.Account) {

}

func GetKVSet(acc *types.Account) (kvset []*types.KeyValue) {
	return nil
}

func PubKeyToAddress(in []byte) *Address {
	a := new(Address)
	a.Pubkey = make([]byte, len(in))
	copy(a.Pubkey[:], in[:])
	a.Version = 0
	a.Hash160 = common.Rimp160AfterSha256(in)
	return a
}

type Address struct {
	Version  byte
	Hash160  [20]byte // For a stealth address: it's HASH160
	Checksum []byte   // Unused for a stealth address
	Pubkey   []byte   // Unused for a stealth address
	Enc58str string
}

func (a *Address) String() string {
	if a.Enc58str == "" {
		var ad [25]byte
		ad[0] = a.Version
		copy(ad[1:21], a.Hash160[:])
		if a.Checksum == nil {
			sh := common.Sha2Sum(ad[0:21])
			a.Checksum = make([]byte, 4)
			copy(a.Checksum, sh[:4])
		}
		copy(ad[21:25], a.Checksum[:])
		a.Enc58str = Encodeb58(ad[:])
	}
	return a.Enc58str
}

var b58set []byte = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func b58chr2int(chr byte) int {
	for i := range b58set {
		if b58set[i] == chr {
			return i
		}
	}
	return -1
}

var bn0 *big.Int = big.NewInt(0)
var bn58 *big.Int = big.NewInt(58)

func Encodeb58(a []byte) (s string) {
	idx := len(a)*138/100 + 1
	buf := make([]byte, idx)
	bn := new(big.Int).SetBytes(a)
	var mo *big.Int
	for bn.Cmp(bn0) != 0 {
		bn, mo = bn.DivMod(bn, bn58, new(big.Int))
		idx--
		buf[idx] = b58set[mo.Int64()]
	}
	for i := range a {
		if a[i] != 0 {
			break
		}
		idx--
		buf[idx] = b58set[0]
	}

	s = string(buf[idx:])

	return
}

func Decodeb58(s string) (res []byte) {
	bn := big.NewInt(0)
	for i := range s {
		v := b58chr2int(byte(s[i]))
		if v < 0 {
			return nil
		}
		bn = bn.Mul(bn, bn58)
		bn = bn.Add(bn, big.NewInt(int64(v)))
	}

	// We want to "restore leading zeros" as satoshi's implementation does:
	var i int
	for i < len(s) && s[i] == b58set[0] {
		i++
	}
	if i > 0 {
		res = make([]byte, i)
	}
	res = append(res, bn.Bytes()...)
	return
}
