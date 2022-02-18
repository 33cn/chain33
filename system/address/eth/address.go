package eth

import (
	"errors"

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

type eth struct {
	ethAddress common.Address
}

// PubKeyToAddr public key to address
func (e *eth) PubKeyToAddr(pubKey []byte) string {
	pubStr := string(pubKey)
	if value, ok := addrCache.Get(pubStr); ok {
		return value.(string)
	}
	addr := FormatEthAddr(pubKey)
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

// FormatEthAddr format eth addr
func FormatEthAddr(pubKey []byte) string {

	pub, err := crypto.DecompressPubkey(pubKey)
	// ecdsa public key, compatible with ethereum, get address from eth api
	if err == nil {
		return crypto.PubkeyToAddress(*pub).String()
	}
	// just format as eth address if pubkey not compatible
	var a common.Address
	a.SetBytes(crypto.Keccak256(pubKey[1:])[12:])
	return a.Hex()
}
