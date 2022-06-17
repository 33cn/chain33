// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package address_test

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/system/address/btc"

	"github.com/33cn/chain33/common/crypto"
	_ "github.com/33cn/chain33/system/crypto/init"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func genkey() crypto.PrivKey {
	c, err := crypto.Load("secp256k1", -1)
	if err != nil {
		panic(err)
	}
	key, err := c.GenKey()
	if err != nil {
		panic(err)
	}
	return key
}
func TestAddress(t *testing.T) {
	key := genkey()
	t.Logf("%X", key.Bytes())
	addr := address.PubKeyToAddr(address.DefaultID, key.PubKey().Bytes())
	t.Log(addr)
}

func TestMultiSignAddress(t *testing.T) {
	key := genkey()
	addr := address.PubKeyToAddr(btc.MultiSignAddressID, key.PubKey().Bytes())
	err := address.CheckBase58Address(address.NormalVer, addr)
	assert.Equal(t, address.ErrCheckVersion, err)
	err = address.CheckBase58Address(address.MultiSignVer, addr)
	assert.Nil(t, err)
	t.Log(addr)
}

func TestPubkeyToAddress(t *testing.T) {
	pubkey := "024a17b0c6eb3143839482faa7e917c9b90a8cfe5008dff748789b8cea1a3d08d5"
	b, err := hex.DecodeString(pubkey)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%X", b)
	addr := address.PubKeyToAddr(address.DefaultID, b)
	t.Log(addr)
}

func TestCheckAddress(t *testing.T) {
	c, err := crypto.Load("secp256k1", -1)
	if err != nil {
		t.Error(err)
		return
	}
	key, err := c.GenKey()
	if err != nil {
		t.Error(err)
		return
	}
	addr := address.PubKeyToAddr(address.DefaultID, key.PubKey().Bytes())
	err = address.CheckBase58Address(address.NormalVer, addr)
	require.NoError(t, err)

	err = address.CheckBase58Address(address.NormalVer, addr+addr)
	require.Equal(t, err, address.ErrAddressChecksum)
}

func TestExecAddress(t *testing.T) {
	assert.Equal(t, "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp", address.ExecAddress("ticket"))
	err := address.CheckBase58Address(address.NormalVer, "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp")
	assert.Nil(t, err)
}

func BenchmarkExecAddress(b *testing.B) {
	start := time.Now().UnixNano() / 1000000
	fmt.Println(start)
	for i := 0; i < b.N; i++ {
		address.ExecAddress("ticket")
	}
	end := time.Now().UnixNano() / 1000000
	fmt.Println(end)
	duration := end - start
	fmt.Println("duration with cache:", strconv.FormatInt(duration, 10))

	start = time.Now().UnixNano() / 1000000
	fmt.Println(start)
	for i := 0; i < b.N; i++ {
		address.ExecAddress("ticket")
	}
	end = time.Now().UnixNano() / 1000000
	fmt.Println(end)
	duration = end - start
	fmt.Println("duration without cache:", strconv.FormatInt(duration, 10))
}
