package btc

import (
	"testing"

	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/require"
)

func TestBtcDriver(t *testing.T) {

	driver := &btc{}
	addr, priv := util.Genaddress()
	require.Equal(t, NormalName, driver.GetName())
	require.Equal(t, addr, driver.PubKeyToAddr(priv.PubKey().Bytes()))
	require.Nil(t, driver.ValidateAddr("12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"))
	require.Nil(t, driver.ValidateAddr(addr))
	require.Equal(t, 1, normalAddrCache.Len())
}

func TestBtcMultiSignDriver(t *testing.T) {

	driver := &btcMultiSign{}
	addr, priv := util.Genaddress()
	require.Equal(t, MultiSignName, driver.GetName())
	require.NotEqual(t, addr, driver.PubKeyToAddr(priv.PubKey().Bytes()))
	require.Equal(t, ErrInvalidAddrFormat, driver.ValidateAddr(addr))
	require.Equal(t, 1, normalAddrCache.Len())
	for i := 0; i < 100; i++ {
		_, priv = util.Genaddress()
		require.Nil(t, driver.ValidateAddr(driver.PubKeyToAddr(priv.PubKey().Bytes())))
	}
}
