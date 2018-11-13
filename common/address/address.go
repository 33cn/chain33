// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package address

import (
	"bytes"
	"encoding/hex"
	"errors"

	. "github.com/33cn/chain33/common"
	"github.com/decred/base58"
	lru "github.com/hashicorp/golang-lru"
)

var addrSeed = []byte("address seed bytes for public key")
var addressCache *lru.Cache
var checkAddressCache *lru.Cache

const MaxExecNameLength = 100

func init() {
	addressCache, _ = lru.New(10240)
	checkAddressCache, _ = lru.New(10240)
}

func ExecPubKey(name string) []byte {
	if len(name) > MaxExecNameLength {
		panic("name too long")
	}
	var bname [200]byte
	buf := append(bname[:0], addrSeed...)
	buf = append(buf, []byte(name)...)
	hash := Sha2Sum(buf)
	return hash[:]
}

//计算量有点大，做一次cache
func ExecAddress(name string) string {
	if value, ok := addressCache.Get(name); ok {
		return value.(string)
	}
	addr := PubKeyToAddress(ExecPubkey(name))
	addrstr := addr.String()
	addressCache.Add(name, addrstr)
	return addrstr
}

func ExecPubkey(name string) []byte {
	if len(name) > MaxExecNameLength {
		panic("name too long")
	}
	var bname [200]byte
	buf := append(bname[:0], addrSeed...)
	buf = append(buf, []byte(name)...)
	hash := Sha2Sum(buf)
	return hash[:]
}

func GetExecAddress(name string) *Address {
	if len(name) > MaxExecNameLength {
		panic("name too long")
	}
	var bname [200]byte
	buf := append(bname[:0], addrSeed...)
	buf = append(buf, []byte(name)...)
	hash := Sha2Sum(buf)
	addr := PubKeyToAddress(hash[:])
	return addr
}

func PubKeyToAddress(in []byte) *Address {
	a := new(Address)
	a.Pubkey = make([]byte, len(in))
	copy(a.Pubkey[:], in[:])
	a.Version = 0
	a.Hash160 = Rimp160AfterSha256(in)
	return a
}

func CheckAddress(addr string) (e error) {
	if value, ok := checkAddressCache.Get(addr); ok {
		if value == nil {
			return nil
		}
		return value.(error)
	}
	dec := base58.Decode(addr)
	if dec == nil {
		e = errors.New("Cannot decode b58 string '" + addr + "'")
		checkAddressCache.Add(addr, e)
		return
	}
	if len(dec) < 25 {
		e = errors.New("Address too short " + hex.EncodeToString(dec))
		checkAddressCache.Add(addr, e)
		return
	}
	if len(dec) == 25 {
		sh := Sha2Sum(dec[0:21])
		if !bytes.Equal(sh[:4], dec[21:25]) {
			e = errors.New("Address Checksum error")
		}
	}
	checkAddressCache.Add(addr, e)
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
		sh := Sha2Sum(dec[0:21])
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
			sh := Sha2Sum(ad[0:21])
			a.Checksum = make([]byte, 4)
			copy(a.Checksum, sh[:4])
		}
		copy(ad[21:25], a.Checksum[:])
		a.Enc58str = base58.Encode(ad[:])
	}
	return a.Enc58str
}
