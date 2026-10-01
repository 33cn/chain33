// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package address_test

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/system/address/btc"
	"github.com/33cn/chain33/types"

	"github.com/33cn/chain33/common/crypto"
	_ "github.com/33cn/chain33/system/address"
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

func TestCheckAddressAPI(t *testing.T) {
	c, err := crypto.Load("secp256k1", -1)
	require.NoError(t, err)
	key, err := c.GenKey()
	require.NoError(t, err)

	addr := address.PubKeyToAddr(address.DefaultID, key.PubKey().Bytes())
	err = address.CheckAddress(addr, -1)
	assert.Nil(t, err)

	err = address.CheckAddress(addr+"invalid", -1)
	assert.NotNil(t, err)
}

func TestGetAddressType(t *testing.T) {
	c, err := crypto.Load("secp256k1", -1)
	require.NoError(t, err)
	key, err := c.GenKey()
	require.NoError(t, err)

	addr := address.PubKeyToAddr(address.DefaultID, key.PubKey().Bytes())
	ty, err := address.GetAddressType(addr)
	assert.Nil(t, err)
	assert.GreaterOrEqual(t, ty, int32(0))
}

func TestAddressStructMethods(t *testing.T) {
	addr := &address.Address{Version: address.NormalVer}
	addr.SetBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	s := addr.String()
	assert.NotEmpty(t, s)
	assert.Contains(t, s, "1")
}

func TestCheckBase58AddressEdgeCases(t *testing.T) {
	assert.NotNil(t, address.CheckBase58Address(address.NormalVer, ""))
}

// CheckAddress's answer depends on the height, because the height decides which drivers are
// enabled -- so the cache cannot be keyed by the address alone. The callers that pass -1
// (RPC, CLI, wallet) share a process with the ones that pass a real height, and -1 means
// "every driver is enabled", including the ones the config disables. Whichever of the two
// ran first used to decide the answer for the other, so a node that had answered such a
// query would accept a transaction at a real height that the other nodes reject.
func TestCheckAddressCacheFollowsHeight(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	address.Init(cfg.GetModuleConfig().Address)
	t.Cleanup(func() {
		// back to the height the eth driver registers itself with, so the rest of the
		// package sees the state it expects
		address.Init(&address.Config{EnableHeight: map[string]int64{"eth": 0}})
	})

	// [address.enableHeight] eth=-2 in types/defaultcfg.go: disabled at every real height.
	ethAddr := "0x" + strings.Repeat("11", 20)
	require.NoError(t, address.CheckAddress(ethAddr, -1), "no block context: every driver is enabled")
	require.Error(t, address.CheckAddress(ethAddr, 0),
		"disabled at height 0 -- the cached -1 answer must not be reused")
}

// The error returned for an address that several drivers reject has to be the same every
// time. Ranging over the driver map made it depend on Go's randomized iteration order, and
// system/dapp.CheckAddress decides whether a legacy address may be tolerated below a fork
// by comparing the error against specific values -- so an arbitrary winner meant an
// arbitrary accept/reject decision, and two nodes could disagree.
func TestCheckAddressErrorIsDeterministic(t *testing.T) {
	const valid = "1HUiTRFvp6HvW6eacgV9EoBSgroRDiUsMs"
	const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// A valid address with its last character replaced: the decoded length and the version
	// byte stay the same, so the base58 driver reports a checksum mismatch (ErrCheckChecksum)
	// while the multi-sign driver reports a version mismatch (ErrCheckVersion). Those two are
	// treated differently by the fork gate, so the arbitrary winner was visible downstream.
	checked := 0
	for i := 0; i < len(base58Alphabet); i++ {
		variant := valid[:len(valid)-1] + string(base58Alphabet[i])
		if variant == valid {
			continue
		}
		require.Equal(t, address.ErrCheckChecksum, address.CheckAddress(variant, 0),
			"variant %q", variant)
		checked++
	}
	require.Greater(t, checked, 40)
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
