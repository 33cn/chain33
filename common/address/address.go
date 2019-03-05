// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package address 计算地址相关的函数
package address

import (
	"bytes"
	"encoding/hex"
	"errors"

	"github.com/33cn/chain33/common"
	"github.com/decred/base58"
	lru "github.com/hashicorp/golang-lru"
)

var addrSeed = []byte("address seed bytes for public key")
var addressCache *lru.Cache
var pubkeyCache *lru.Cache
var checkAddressCache *lru.Cache
var multisignCache *lru.Cache
var multiCheckAddressCache *lru.Cache

// ErrCheckVersion :
var ErrCheckVersion = errors.New("check version error")

//ErrCheckChecksum :
var ErrCheckChecksum = errors.New("Address Checksum error")

//MaxExecNameLength 执行器名最大长度
const MaxExecNameLength = 100

//NormalVer 普通地址的版本号
const NormalVer byte = 0

//MultiSignVer 多重签名地址的版本号
const MultiSignVer byte = 5

func init() {
	var err error
	multisignCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	pubkeyCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	addressCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	checkAddressCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	multiCheckAddressCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
}

//ExecPubKey 计算公钥
func ExecPubKey(name string) []byte {
	if len(name) > MaxExecNameLength {
		panic("name too long")
	}
	var bname [200]byte
	buf := append(bname[:0], addrSeed...)
	buf = append(buf, []byte(name)...)
	hash := common.Sha2Sum(buf)
	return hash[:]
}

//ExecAddress 计算量有点大，做一次cache
func ExecAddress(name string) string {
	if value, ok := addressCache.Get(name); ok {
		return value.(string)
	}
	addr := GetExecAddress(name)
	addrstr := addr.String()
	addressCache.Add(name, addrstr)
	return addrstr
}

//MultiSignAddress create a multi sign address
func MultiSignAddress(pubkey []byte) string {
	if value, ok := multisignCache.Get(string(pubkey)); ok {
		return value.(string)
	}
	addr := HashToAddress(MultiSignVer, pubkey)
	addrstr := addr.String()
	multisignCache.Add(string(pubkey), addrstr)
	return addrstr
}

//ExecPubkey 计算公钥
func ExecPubkey(name string) []byte {
	if len(name) > MaxExecNameLength {
		panic("name too long")
	}
	var bname [200]byte
	buf := append(bname[:0], addrSeed...)
	buf = append(buf, []byte(name)...)
	hash := common.Sha2Sum(buf)
	return hash[:]
}

//GetExecAddress 获取地址
func GetExecAddress(name string) *Address {
	hash := ExecPubkey(name)
	addr := PubKeyToAddress(hash[:])
	return addr
}

//PubKeyToAddress 公钥转为地址
func PubKeyToAddress(in []byte) *Address {
	return HashToAddress(NormalVer, in)
}

//PubKeyToAddr 公钥转为地址
func PubKeyToAddr(in []byte) string {
	if value, ok := pubkeyCache.Get(string(in)); ok {
		return value.(string)
	}
	addr := HashToAddress(NormalVer, in).String()
	pubkeyCache.Add(string(in), addr)
	return addr
}

//HashToAddress hash32 to address
func HashToAddress(version byte, in []byte) *Address {
	a := new(Address)
	a.Pubkey = make([]byte, len(in))
	copy(a.Pubkey[:], in[:])
	a.Version = version
	a.SetBytes(common.Rimp160(in))
	return a
}

func checkAddress(ver byte, addr string) (e error) {
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
		sh := common.Sha2Sum(dec[0:21])
		if !bytes.Equal(sh[:4], dec[21:25]) {
			e = ErrCheckChecksum
		}
	}
	if dec[0] != ver {
		e = ErrCheckVersion
	}
	return e
}

//CheckMultiSignAddress 检查多重签名地址的有效性
func CheckMultiSignAddress(addr string) (e error) {
	if value, ok := multiCheckAddressCache.Get(addr); ok {
		if value == nil {
			return nil
		}
		return value.(error)
	}
	e = checkAddress(MultiSignVer, addr)
	multiCheckAddressCache.Add(addr, e)
	return
}

//CheckAddress 检查地址
func CheckAddress(addr string) (e error) {
	if value, ok := checkAddressCache.Get(addr); ok {
		if value == nil {
			return nil
		}
		return value.(error)
	}
	e = checkAddress(NormalVer, addr)
	checkAddressCache.Add(addr, e)
	return
}

//NewAddrFromString new 地址
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
			e = ErrCheckChecksum
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

//Address 地址
type Address struct {
	Version  byte
	Hash160  [20]byte // For a stealth address: it's HASH160
	Checksum []byte   // Unused for a stealth address
	Pubkey   []byte   // Unused for a stealth address
	Enc58str string
}

//SetBytes 设置地址的bytes
func (a *Address) SetBytes(b []byte) {
	copy(a.Hash160[:], b)
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
