package eth_test

import (
	"strings"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/system/address/eth"
	"github.com/33cn/chain33/system/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatEthAddr(t *testing.T) {

	ethDriver, err := address.LoadDriver(eth.ID, -1)
	require.Nil(t, err)
	d := secp256k1.Driver{}
	for i := 0; i < 100; i++ {

		ethPriv, err := crypto.GenerateKey()
		require.Nil(t, err)
		ethAddr := crypto.PubkeyToAddress(ethPriv.PublicKey).Hex()
		chain33Priv, err := d.PrivKeyFromBytes(crypto.FromECDSA(ethPriv))
		require.Nil(t, err)
		err = ethDriver.ValidateAddr(ethAddr)
		require.Nil(t, err)
		addr := ethDriver.PubKeyToAddr(chain33Priv.PubKey().Bytes())
		require.Equal(t, address.FormatEthAddress(ethAddr), addr)
	}
	require.Equal(t, eth.Name, ethDriver.GetName())
}

func TestHexAddr(t *testing.T) {

	addr := "0x6c0d7be0d2c8350042890a77393158181716b0d6"
	upperCaseAddr := "0x6c0d7BE0d2C8350042890a77393158181716b0d6"
	ethDriver, err := address.LoadDriver(eth.ID, -1)
	require.Nil(t, err)
	err = ethDriver.ValidateAddr(upperCaseAddr)
	require.Nil(t, err)

	raw, err := ethDriver.FromString(upperCaseAddr)
	require.Nil(t, err)
	require.Equal(t, addr, ethDriver.ToString(raw))
}

func BenchmarkFormatToStandard(b *testing.B) {

	addr := "0x6c0d7BE0d2C8350042890a77393158181716b0d6"
	b.ResetTimer()
	b.Run("toLower", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			strings.ToLower(addr)
		}
	})

	b.Run("toChecksumValid", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			common.HexToAddress(addr).Hex()
		}
	})
}
