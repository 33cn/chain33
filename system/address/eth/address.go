package eth

import (
	"errors"

	"github.com/33cn/chain33/common/crypto/client"

	"github.com/33cn/chain33/common/address"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	lru "github.com/hashicorp/golang-lru"
)

const (
	// ID normal address id
	ID = 2
	// Name driver name
	Name = "eth"
)

var (
	addrCache *lru.Cache
	// ErrInvalidEthAddr invalid ethereum address
	ErrInvalidEthAddr = errors.New("ErrInvalidEthAddr")
)

func init() {
	address.RegisterDriver(ID, &eth{}, 0)

	var err error
	addrCache, err = lru.New(10240)
	if err != nil {
		panic(err)
	}
}

// eth地址驱动, chain33中统一采用小写格式
type eth struct{}

// PubKeyToAddr public key to address
func (e *eth) PubKeyToAddr(pubKey []byte) string {
	pubStr := string(pubKey)
	if value, ok := addrCache.Get(pubStr); ok {
		return value.(string)
	}
	addr := pubKey2EthAddr(pubKey)
	addrCache.Add(pubStr, addr)
	return addr
}

// ValidateAddr address validation
func (e *eth) ValidateAddr(addr string) error {
	if common.IsHexAddress(addr) {
		return nil
	}
	return ErrInvalidEthAddr
}

// GetName get driver name
func (e *eth) GetName() string {
	return Name
}

// ToString trans to string format
func (e *eth) ToString(addr []byte) string {
	return formatAddr(common.BytesToAddress(addr).String())
}

// FromString trans to byte format
func (e *eth) FromString(addr string) ([]byte, error) {
	if err := e.ValidateAddr(addr); err != nil {
		return nil, err
	}
	return common.HexToAddress(addr).Bytes(), nil
}

func (e *eth) FormatAddr(addr string) string {

	return formatAddr(addr)
}

func formatAddr(addr string) string {
	ctx := client.GetCryptoContext()
	if ctx.API == nil || ctx.API.GetConfig().IsFork(ctx.CurrBlockHeight, address.ForkFormatAddressKey) {
		return address.FormatEthAddress(addr)
	}
	return addr
}

// pubKey2EthAddr format eth addr
func pubKey2EthAddr(pubKey []byte) string {

	pub, err := crypto.DecompressPubkey(pubKey)
	// ecdsa public key, compatible with ethereum, get address from eth api
	if err == nil {
		return formatAddr(crypto.PubkeyToAddress(*pub).String())
	}
	// just format as eth address if pubkey not compatible
	var a common.Address
	a.SetBytes(crypto.Keccak256(pubKey[1:])[12:])
	return formatAddr(a.String())
}
