// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package address_test

import (
	"testing"

	"github.com/33cn/chain33/system/address/eth"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/system/address/btc"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {

	config := &address.Config{EnableHeight: make(map[string]int64)}

	address.Init(config)
	require.Equal(t, btc.NormalName, config.DefaultDriver)
	f := func() {
		address.Init(config)
	}
	unknownName := "unknown"
	config.EnableHeight[unknownName] = 0
	require.Panics(t, f)
	delete(config.EnableHeight, unknownName)
	config.DefaultDriver = unknownName
	require.Panics(t, f)
	config.DefaultDriver = btc.NormalName
	config.EnableHeight[btc.NormalName] = 1
	require.Panics(t, f)
	config.EnableHeight[btc.NormalName] = 0
	require.NotPanics(t, f)
}

type mockDriver struct{}

func (m *mockDriver) PubKeyToAddr(pubKey []byte) string { return "" }

func (m *mockDriver) ValidateAddr(addr string) error { return nil }

func (m *mockDriver) GetName() string { return "mock" }

func (m *mockDriver) ToString([]byte) string { return "" }

func (m *mockDriver) FromString(string) ([]byte, error) { return nil, nil }

func (m *mockDriver) FormatAddr(addr string) string { return "" }

func Test_RegisterDriver(t *testing.T) {

	m := &mockDriver{}
	require.Panics(t, func() { address.RegisterDriver(-1, m, 0) })
	require.Panics(t, func() { address.RegisterDriver(0, m, 0) })
	require.NotPanics(t, func() { address.RegisterDriver(address.MaxID, m, 0) })
	require.Panics(t, func() { address.RegisterDriver(address.MaxID-1, m, 0) })
}

func Test_LoadDriver(t *testing.T) {

	_, err := address.LoadDriver(-1, 0)
	require.Equal(t, address.ErrUnknownAddressDriver, err)
	_, err = address.LoadDriver(eth.ID, 0)
	require.Equal(t, nil, err)
	d, err := address.LoadDriver(btc.NormalAddressID, 0)
	require.Equal(t, btc.NormalName, d.GetName())
	require.Nil(t, err)
}

func Test_util(t *testing.T) {

	id := address.GetDefaultAddressID()
	require.Equal(t, int32(0), id)
	list := address.GetDriverList()

	for id, driver := range list {
		ty, err := address.GetDriverType(driver.GetName())
		require.Nil(t, err)
		require.Equal(t, id, ty)
	}
}
