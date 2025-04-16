package btc

import (
	"errors"
	"github.com/33cn/chain33/common/address"
	"github.com/btcsuite/btcd/wire"
	"strings"
)

func init() {
	address.RegisterDriver(3, &utxo{}, 0)
}

var errAddressType = errors.New("ErrAddressType")

type utxo struct{}

// PubKeyToAddr public key to address
func (u *utxo) PubKeyToAddr(_ []byte) string {
	panic("implement me")
}

// ValidateAddr address validation
func (u *utxo) ValidateAddr(addr string) error {

	if !strings.Contains(addr, ":") {
		return errAddressType
	}
	_, err := wire.NewOutPointFromString(addr)
	if err != nil {
		return errAddressType
	}
	return nil
}

// GetName get driver name
func (u *utxo) GetName() string {
	return "utxo"
}

// ToString trans to string format
func (u *utxo) ToString(addr []byte) string {
	panic("implement me")
}

// FromString trans to byte format
func (u *utxo) FromString(addr string) ([]byte, error) {
	panic("implement me")
}

func (u *utxo) FormatAddr(addr string) string { return addr }
