// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package types

import (
	"testing"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/system/crypto/secp256k1eth"
	"github.com/stretchr/testify/require"
)

func TestAddressID(t *testing.T) {

	var cryptoID, addressID int32

	for ; cryptoID <= crypto.MaxManualTypeID; cryptoID++ {
		addressID = 0
		for ; addressID <= address.MaxID; addressID++ {
			signID := EncodeSignID(cryptoID, addressID)
			require.Equal(t, addressID, ExtractAddressID(signID))
			require.Equal(t, cryptoID, ExtractCryptoID(signID))
			dupSignID := EncodeSignID(signID, addressID)
			require.Equal(t, signID, dupSignID)
		}
	}
}

func TestIsEthSignID(t *testing.T) {
	require.True(t, IsEthSignID(EncodeSignID(secp256k1eth.ID, EthAddressID)))
	require.False(t, IsEthSignID(EncodeSignID(secp256k1eth.ID, 0)))
	require.False(t, IsEthSignID(0))
}

func TestEncodeSignIDWithInvalidAddress(t *testing.T) {
	// Invalid address ID should fall back to default
	signID := EncodeSignID(1, address.MaxID+1)
	require.Equal(t, address.GetDefaultAddressID(), ExtractAddressID(signID))
	require.Equal(t, int32(1), ExtractCryptoID(signID))
}

func TestExtractAddressID(t *testing.T) {
	require.Equal(t, int32(0), ExtractAddressID(0))
	require.Equal(t, int32(2), ExtractAddressID(EncodeSignID(0, 2)))
}
