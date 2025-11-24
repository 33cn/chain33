// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package address 计算地址相关的函数
package address

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/33cn/chain33/common"
	"github.com/decred/base58"
	lru "github.com/hashicorp/golang-lru"
)

var addrSeed = []byte("address seed bytes for public key")
var execAddrCache *lru.Cache
var checkAddressCache *lru.Cache
var execPubKeyCache *lru.Cache

// ErrCheckVersion :
var ErrCheckVersion = errors.New("check version error")

// ErrCheckChecksum :
var ErrCheckChecksum = errors.New("Address Checksum error")

// ErrAddressChecksum :
var ErrAddressChecksum = errors.New("address checksum error")

var (
	// ErrDecodeBase58 error decode
	ErrDecodeBase58 = errors.New("ErrDecodeBase58")
	// ErrAddressLength error length
	ErrAddressLength = errors.New("ErrAddressLength")
)

// MaxExecNameLength 执行器名最大长度
const MaxExecNameLength = 100

// NormalVer 普通地址的版本号
var NormalVer byte

// MultiSignVer 多重签名地址的版本号
var MultiSignVer byte = 5

func init() {

	var err error
	execAddrCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	checkAddressCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	execPubKeyCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
}

// ExecAddress 计算量有点大，做一次cache
// contract address
func ExecAddress(name string) string {
	if value, ok := execAddrCache.Get(name); ok {
		return value.(string)
	}
	addr, err := GetExecAddress(name, defaultAddressID)
	if err != nil {
		panic(fmt.Sprintf("load default driver err, id:%d", defaultAddressID))
	}
	execAddrCache.Add(name, addr)
	return addr
}

// ExecPubKey 计算公钥
func ExecPubKey(name string) []byte {
	if len(name) > MaxExecNameLength {
		panic("name too long")
	}
	if value, ok := execPubKeyCache.Get(name); ok {
		return value.([]byte)
	}
	var bname [200]byte
	buf := append(bname[:0], addrSeed...)
	buf = append(buf, []byte(name)...)
	hash := common.Sha2Sum(buf)
	execPubKeyCache.Add(name, hash)
	return hash[:]
}

// GetExecAddress 获取地址
func GetExecAddress(execName string, addressType int32) (string, error) {
	d, err := LoadDriver(addressType, -1)
	if err != nil {
		return "", err
	}
	pubKey := ExecPubKey(execName)
	return d.PubKeyToAddr(pubKey), nil
}

// PubKeyToAddr pubKey to specific address
// pass DefaultID for default address format
func PubKeyToAddr(addressID int32, pubKey []byte) string {

	if addressID < 0 {
		addressID = defaultAddressID
	}
	d := MustLoadDriver(addressID)
	return d.PubKeyToAddr(pubKey)
}

// CheckAddress check address validity
// blockHeight is used for enable check, pass -1 if there is no block height context
func CheckAddress(addr string, blockHeight int64) (e error) {

	if value, ok := checkAddressCache.Get(addr); ok {
		if value != nil {
			return value.(error)
		}
		return nil
	}
	for _, d := range drivers {
		if !isEnable(blockHeight, d.enableHeight) {
			continue
		}
		e = d.driver.ValidateAddr(addr)
		if e == nil {
			break
		}
	}
	checkAddressCache.Add(addr, e)
	return e
}

// GetAddressType get address type id
func GetAddressType(addr string) (int32, error) {
	for ty, d := range drivers {
		e := d.driver.ValidateAddr(addr)
		if e == nil {
			return ty, nil
		}
	}
	return -1, ErrUnknownAddressType
}

// BytesToBtcAddress hash32 to address
// Deprecated: btc address legacy
func BytesToBtcAddress(version byte, in []byte) *Address {
	a := new(Address)
	a.Pubkey = make([]byte, len(in))
	copy(a.Pubkey[:], in[:])
	a.Version = version
	a.SetBytes(common.Rimp160(in))
	return a
}

func checksum(input []byte) (cksum [4]byte) {
	h := sha256.Sum256(input)
	h2 := sha256.Sum256(h[:])
	copy(cksum[:], h2[:4])
	return
}

// CheckBase58Address check base58 format address, usually refers to
func CheckBase58Address(ver byte, addr string) (e error) {

	dec := base58.Decode(addr)
	if dec == nil {
		return ErrDecodeBase58
	}
	if len(dec) < 25 {
		return ErrAddressLength
	}
	//version 的错误优先
	if dec[0] != ver {
		return ErrCheckVersion
	}
	//需要兼容以前的错误(以前的错误，是一种特殊的情况)
	if len(dec) == 25 {
		sh := common.Sha2Sum(dec[0:21])
		if !bytes.Equal(sh[:4], dec[21:25]) {
			e = ErrCheckChecksum
			return
		}
	}
	var cksum [4]byte
	copy(cksum[:], dec[len(dec)-4:])
	//新的错误: 这个错误用一种新的错误标记
	if checksum(dec[:len(dec)-4]) != cksum {
		e = ErrAddressChecksum
	}
	return e
}

// Address btc address
// Deprecated
type Address struct {
	Version  byte
	Hash160  [20]byte // For a stealth address: it's HASH160
	Checksum []byte   // Unused for a stealth address
	Pubkey   []byte   // Unused for a stealth address
	Enc58str string
}

// SetBytes 设置地址的bytes
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

// SetNormalAddrVer 根据配置设置生成普通地址的version版本号，默认是0
func SetNormalAddrVer(ver byte) {
	if MultiSignVer == ver {
		panic("the version of the normal address conflicts with the version of the multi-signature address!")

	}
	NormalVer = ver
}

// NewBtcAddress new btc address
// Deprecated: legacy
func NewBtcAddress(addr string) (*Address, error) {
	dec := base58.Decode(addr)
	if dec == nil {
		return nil, ErrDecodeBase58
	}
	if len(dec) != 25 {
		return nil, ErrAddressLength
	}

	sh := common.Sha2Sum(dec[0:21])
	if !bytes.Equal(sh[:4], dec[21:25]) {
		return nil, ErrCheckChecksum
	}

	a := new(Address)
	a.Version = dec[0]
	copy(a.Hash160[:], dec[1:21])
	a.Checksum = make([]byte, 4)
	copy(a.Checksum, dec[21:25])
	a.Enc58str = addr
	return a, nil
}
