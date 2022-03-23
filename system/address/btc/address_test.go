package btc_test

import (
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/system/address/btc"
	"github.com/33cn/chain33/system/crypto/sm2"

	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/require"
)

func TestBtcDriver(t *testing.T) {

	driver, err := address.LoadDriver(btc.NormalAddressID, -1)
	require.Nil(t, err)
	addr, priv := util.Genaddress()
	sm2driver := &sm2.Driver{}
	sm2Priv, err := sm2driver.PrivKeyFromBytes(priv.Bytes())
	require.Nil(t, err)
	pub, err := sm2driver.PubKeyFromBytes(priv.PubKey().Bytes())
	require.Nil(t, err)
	println(common.ToHex(pub.Bytes()))
	println(common.ToHex(sm2Priv.PubKey().Bytes()))
	require.Equal(t, btc.NormalName, driver.GetName())
	println(driver.PubKeyToAddr(priv.PubKey().Bytes()))
	println(driver.PubKeyToAddr(sm2Priv.PubKey().Bytes()))
	require.Equal(t, addr, driver.PubKeyToAddr(priv.PubKey().Bytes()))
	require.Nil(t, driver.ValidateAddr("12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"))
	require.Nil(t, driver.ValidateAddr(addr))
}

func TestBtcMultiSignDriver(t *testing.T) {

	driver, err := address.LoadDriver(btc.MultiSignAddressID, -1)
	require.Nil(t, err)
	addr, priv := util.Genaddress()
	require.Equal(t, btc.MultiSignName, driver.GetName())
	require.NotEqual(t, addr, driver.PubKeyToAddr(priv.PubKey().Bytes()))
	require.Equal(t, btc.ErrInvalidAddrFormat, driver.ValidateAddr(addr))
	for i := 0; i < 100; i++ {
		_, priv = util.Genaddress()
		require.Nil(t, driver.ValidateAddr(driver.PubKeyToAddr(priv.PubKey().Bytes())))
	}
}

func Test_ErrAddr(t *testing.T) {
	addr := "DsYQcck3QFK9Wt1UWd5eoskWjk8JdYSCMoK"
	err := address.CheckAddress(addr, 0)
	require.Equal(t, address.ErrCheckVersion, err)
}
