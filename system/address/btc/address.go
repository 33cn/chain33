package btc

import (
	"errors"
	"strings"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/decred/base58"
	lru "github.com/hashicorp/golang-lru"
)

const (
	// NormalAddressID normal address id
	NormalAddressID = 0
	normalName      = "btc"
	// MultiSignAddressID multi sign address id
	MultiSignAddressID = 1
	multiSignName      = "btc-multi-sign"
)

var (
	normalAddrCache     *lru.Cache
	multiSignAddrCache  *lru.Cache
	normalAddrPrefix    string
	multiSignAddrPrefix string
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
	return address.CheckBase58Address(address.MultiSignVer, addr)
}

// GetName get driver name
func (b *btc) GetName() string {
	return normalName
}

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
	if !strings.HasPrefix(addr, "3") {
		return ErrInvalidAddrFormat
	}
	return address.CheckBase58Address(address.MultiSignVer, addr)
}

// GetName get driver name
func (b *btcMultiSign) GetName() string {
	return multiSignName
}

// FormatBtcAddr
func FormatBtcAddr(version byte, pubKey []byte) string {

	var ad [25]byte
	ad[0] = version
	copy(ad[1:21], common.Rimp160(pubKey))
	copy(ad[21:25], common.Sha2Sum(ad[0:21])[:4])
	return base58.Encode(ad[:])
}
