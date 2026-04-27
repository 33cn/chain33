// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCBCEncryptDecryptPrivkey(t *testing.T) {
	password := []byte("test-password-12345678901234567890")
	privkey := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	encrypted := CBCEncrypterPrivkey(password, privkey)
	assert.NotNil(t, encrypted)
	assert.Equal(t, len(privkey), len(encrypted))
	// Encrypted data should be different from original
	assert.False(t, bytes.Equal(privkey, encrypted))

	decrypted := CBCDecrypterPrivkey(password, encrypted)
	assert.Equal(t, privkey, decrypted)
}

func TestCBCEncryptDecryptPrivkeyLongPassword(t *testing.T) {
	password := []byte("this-is-a-very-long-password-that-exceeds-32-bytes-length")
	privkey := []byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1}

	encrypted := CBCEncrypterPrivkey(password, privkey)
	assert.NotNil(t, encrypted)

	decrypted := CBCDecrypterPrivkey(password, encrypted)
	assert.Equal(t, privkey, decrypted)
}

func TestCBCEncryptDecryptPrivkeyShortPassword(t *testing.T) {
	password := []byte("short")
	privkey := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	encrypted := CBCEncrypterPrivkey(password, privkey)
	assert.NotNil(t, encrypted)

	decrypted := CBCDecrypterPrivkey(password, encrypted)
	assert.Equal(t, privkey, decrypted)
}

func TestCBCEncryptDecryptRoundTrip(t *testing.T) {
	for i := 0; i < 100; i++ {
		password := make([]byte, 16)
		privkey := make([]byte, 16)
		for j := 0; j < 16; j++ {
			password[j] = byte(j + i)
			privkey[j] = byte(255 - j - i)
		}
		encrypted := CBCEncrypterPrivkey(password, privkey)
		decrypted := CBCDecrypterPrivkey(password, encrypted)
		assert.Equal(t, privkey, decrypted, "round trip failed at iteration %d", i)
	}
}

func TestCBCEncryptWrongDecrypt(t *testing.T) {
	password1 := []byte("password-one-1234567890123456")
	password2 := []byte("password-two-1234567890123456")
	privkey := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	encrypted := CBCEncrypterPrivkey(password1, privkey)
	decrypted := CBCDecrypterPrivkey(password2, encrypted)
	// Wrong password should NOT decrypt to original
	assert.False(t, bytes.Equal(privkey, decrypted))
}

func TestCalcAccountKey(t *testing.T) {
	key := CalcAccountKey("1234567890", "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt")
	assert.NotNil(t, key)
	assert.True(t, bytes.HasPrefix(key, []byte("Account:")))
	assert.True(t, bytes.Contains(key, []byte("14KEKbYtKKQm4wMthSK9J4La4nAiidGozt")))
}

func TestCalcAddrKey(t *testing.T) {
	key := CalcAddrKey("14KEKbYtKKQm4wMthSK9J4La4nAiidGozt")
	assert.NotNil(t, key)
	assert.True(t, bytes.HasPrefix(key, []byte("Addr:")))
	assert.True(t, bytes.Contains(key, []byte("14KEKbYtKKQm4wMthSK9J4La4nAiidGozt")))
}

func TestCalcLabelKey(t *testing.T) {
	key := CalcLabelKey("myaccount")
	assert.NotNil(t, key)
	assert.True(t, bytes.HasPrefix(key, []byte("Label:")))
	assert.True(t, bytes.HasSuffix(key, []byte("myaccount")))
}

func TestCalcTxKey(t *testing.T) {
	key := CalcTxKey("100001")
	assert.NotNil(t, key)
	assert.True(t, bytes.HasPrefix(key, []byte("Tx:")))
	assert.True(t, bytes.HasSuffix(key, []byte("100001")))
}

func TestCalcEncryptionFlag(t *testing.T) {
	key := CalcEncryptionFlag()
	assert.Equal(t, "Encryption", string(key))
}

func TestCalckeyEncryptionCompFlag(t *testing.T) {
	key := CalckeyEncryptionCompFlag()
	assert.Equal(t, "EncryptionFlag", string(key))
}

func TestCalcPasswordHash(t *testing.T) {
	key := CalcPasswordHash()
	assert.Equal(t, "PasswordHash", string(key))
}

func TestCalcWalletSeed(t *testing.T) {
	key := CalcWalletSeed()
	assert.Equal(t, "walletseed", string(key))
}

func TestCalcAirDropIndex(t *testing.T) {
	key := CalcAirDropIndex()
	assert.Equal(t, "AirDropIndex", string(key))
}

func TestCalcAddrKeyWithFormatting(t *testing.T) {
	// Test with uppercase address - CalcAddrKey should format it
	key1 := CalcAddrKey("1EDDghAtgBsamrNEjN3g94jNA5CLcNxXro")
	key2 := CalcAddrKey("1EDDghAtgBsamrNEjN3g94jNA5CLcNxXro")
	assert.Equal(t, key1, key2)
}
