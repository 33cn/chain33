package btc

import (
	"bytes"
	"errors"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/decred/base58"
	lru "github.com/hashicorp/golang-lru"
)

const (
	// NormalAddressID normal address id
	NormalAddressID = 0
	// NormalName driver name
	NormalName = "btc"
	// MultiSignAddressID multi sign address id
	MultiSignAddressID = 1
	// MultiSignName multi sign driver name
	MultiSignName = "btcMultiSign"
)

var (
	normalAddrCache    *lru.Cache
	multiSignAddrCache *lru.Cache
	// ErrInvalidAddrFormat invalid address format
	ErrInvalidAddrFormat = errors.New("ErrInvalidAddrFormat")
)

func init() {
	address.RegisterDriver(NormalAddressID, &btc{}, 0)
	address.RegisterDriver(MultiSignAddressID, &btcMultiSign{}, 0)

	var err error
	multiSignAddrCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
	normalAddrCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
}

type btc struct{}

// PubKeyToAddr public key to address
func (b *btc) PubKeyToAddr(pubKey []byte) string {
	pubStr := string(pubKey)
	if value, ok := normalAddrCache.Get(pubStr); ok {
		return value.(string)
	}
	addr := FormatBtcAddr(address.NormalVer, pubKey)
	normalAddrCache.Add(pubStr, addr)
	return addr
}

// ValidateAddr address validation
func (b *btc) ValidateAddr(addr string) error {
	return address.CheckBase58Address(address.NormalVer, addr)
}

// GetName get driver name
func (b *btc) GetName() string {
	return NormalName
}

// ToString trans to string format
func (b *btc) ToString(addr []byte) string {
	return encodeToString(address.NormalVer, addr)
}

// FromString trans to byte format
func (b *btc) FromString(addr string) ([]byte, error) {

	return decodeFromString(address.NormalVer, addr)
}

func (b *btc) FormatAddr(addr string) string { return addr }

type btcMultiSign struct{}

// PubKeyToAddr public key to address
func (b *btcMultiSign) PubKeyToAddr(pubKey []byte) string {

	pubStr := string(pubKey)
	if value, ok := multiSignAddrCache.Get(pubStr); ok {
		return value.(string)
	}
	addr := FormatBtcAddr(address.MultiSignVer, pubKey)
	multiSignAddrCache.Add(pubStr, addr)
	return addr
}

// ValidateAddr address validation
func (b *btcMultiSign) ValidateAddr(addr string) error {
	return address.CheckBase58Address(address.MultiSignVer, addr)
}

// GetName get driver name
func (b *btcMultiSign) GetName() string {
	return MultiSignName
}

// ToString trans to string format
func (b *btcMultiSign) ToString(addr []byte) string {
	return encodeToString(address.MultiSignVer, addr)
}

// FromString trans to byte format
func (b *btcMultiSign) FromString(addr string) ([]byte, error) {
	return decodeFromString(address.MultiSignVer, addr)
}

func (b *btcMultiSign) FormatAddr(addr string) string { return addr }

func encodeToString(version byte, raw []byte) string {
	var ad [25]byte
	ad[0] = version
	if len(raw) > 20 {
		raw = raw[:20]
	}
	// ad[1:21], left padding with zero if len(raw) < 20
	copy(ad[21-len(raw):], raw)
	// ad[21:25]  checksum
	copy(ad[21:25], common.Sha2Sum(ad[0:21])[:4])
	return base58.Encode(ad[:])
}

func decodeFromString(version byte, addr string) ([]byte, error) {

	raw := base58.Decode(addr)
	if raw == nil {
		return nil, address.ErrDecodeBase58
	}
	if len(raw) != 25 {
		return nil, address.ErrAddressLength
	}

	sh := common.Sha2Sum(raw[:21])
	if !bytes.Equal(sh[:4], raw[21:25]) {
		return nil, address.ErrCheckChecksum
	}
	if raw[0] != version {
		return nil, address.ErrCheckVersion
	}
	return raw[1:21], nil
}

// FormatBtcAddr format bitcoin address
func FormatBtcAddr(version byte, pubKey []byte) string {

	var ad [25]byte
	ad[0] = version
	copy(ad[1:21], common.Rimp160(pubKey))
	copy(ad[21:25], common.Sha2Sum(ad[0:21])[:4])
	return base58.Encode(ad[:])
}

//
