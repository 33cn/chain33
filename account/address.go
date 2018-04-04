package account

import (
	"bytes"
	"encoding/hex"
	"errors"

	"github.com/decred/base58"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
)

var addrSeed = []byte("address seed bytes for public key")
var bname [200]byte

func ExecAddress(name string) *Address {
	if len(name) > 100 {
		panic("name too long")
	}
	buf := append(bname[:0], addrSeed...)
	buf = append(buf, []byte(name)...)
	hash := common.Sha2Sum(buf)
	return PubKeyToAddress(hash[:])
}

func PubKeyToAddress(in []byte) *Address {
	a := new(Address)
	a.Pubkey = make([]byte, len(in))
	copy(a.Pubkey[:], in[:])
	a.Version = 0
	a.Hash160 = common.Rimp160AfterSha256(in)
	return a
}

func From(tx *types.Transaction) *Address {
	return PubKeyToAddress(tx.Signature.Pubkey)
}

func CheckAddress(addr string) (e error) {
	dec := base58.Decode(addr)
	if dec == nil {
		e = errors.New("Cannot decode b58 string '" + addr + "'")
		return
	}
	if len(dec) < 25 {
		e = errors.New("Address too short " + hex.EncodeToString(dec))
		return
	}
	if len(dec) == 25 {
		sh := common.Sha2Sum(dec[0:21])
		if !bytes.Equal(sh[:4], dec[21:25]) {
			e = errors.New("Address Checksum error")
		}
	}
	return
}

func NewAddrFromString(hs string) (a *Address, e error) {
	dec := base58.Decode(hs)
	if dec == nil {
		e = errors.New("Cannot decode b58 string '" + hs + "'")
		return
	}
	if len(dec) < 25 {
		e = errors.New("Address too short " + hex.EncodeToString(dec))
		return
	}
	if len(dec) == 25 {
		sh := common.Sha2Sum(dec[0:21])
		if !bytes.Equal(sh[:4], dec[21:25]) {
			e = errors.New("Address Checksum error")
		} else {
			a = new(Address)
			a.Version = dec[0]
			copy(a.Hash160[:], dec[1:21])
			a.Checksum = make([]byte, 4)
			copy(a.Checksum, dec[21:25])
			a.Enc58str = hs
		}
	}
	return
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
		a.Enc58str = base58.Encode(ad[:])
	}
	return a.Enc58str
}
