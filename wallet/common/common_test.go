// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"testing"

	"github.com/stretchr/testify/assert"
)

// secp256k1 私钥固定 32 字节，这里的测试数据也用 32 字节对齐生产场景
func TestCBCEncryptDecryptPrivkey(t *testing.T) {
	password := []byte("test-password-12345678901234567890")
	privkey := []byte{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
	}

	encrypted := CBCEncrypterPrivkey(password, privkey)
	assert.NotNil(t, encrypted)
	// 新格式 = IV(16) + 密文(len(privkey))
	assert.Equal(t, aes.BlockSize+len(privkey), len(encrypted))
	assert.False(t, bytes.Equal(privkey, encrypted))

	decrypted := CBCDecrypterPrivkey(password, encrypted)
	assert.Equal(t, privkey, decrypted)
}

func TestCBCEncryptDecryptPrivkeyLongPassword(t *testing.T) {
	password := []byte("this-is-a-very-long-password-that-exceeds-32-bytes-length")
	privkey := []byte{
		32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17,
		16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1,
	}

	encrypted := CBCEncrypterPrivkey(password, privkey)
	assert.NotNil(t, encrypted)

	decrypted := CBCDecrypterPrivkey(password, encrypted)
	assert.Equal(t, privkey, decrypted)
}

func TestCBCEncryptDecryptPrivkeyShortPassword(t *testing.T) {
	password := []byte("short")
	privkey := []byte{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
	}

	encrypted := CBCEncrypterPrivkey(password, privkey)
	assert.NotNil(t, encrypted)

	decrypted := CBCDecrypterPrivkey(password, encrypted)
	assert.Equal(t, privkey, decrypted)
}

func TestCBCEncryptDecryptRoundTrip(t *testing.T) {
	for i := 0; i < 100; i++ {
		password := make([]byte, 16)
		privkey := make([]byte, 32)
		for j := 0; j < 16; j++ {
			password[j] = byte(j + i)
		}
		for j := 0; j < 32; j++ {
			privkey[j] = byte(255 - j - i)
		}
		encrypted := CBCEncrypterPrivkey(password, privkey)
		decrypted := CBCDecrypterPrivkey(password, encrypted)
		assert.Equal(t, privkey, decrypted, "round trip failed at iteration %d", i)
	}
}

// TestCBCEncryptDecryptPrivkey64 覆盖 ed25519 私钥（64 字节）的加解密往返。
// 回归：CBCDecrypterPrivkey 对随机 IV 新格式仅接受 32 字节明文，64 字节私钥会被
// 误判为旧格式并用 key[:16] 当 IV 解密，导致 privacy 等钱包的密钥损坏。
func TestCBCEncryptDecryptPrivkey64(t *testing.T) {
	password := []byte("test-password-12345678901234567890")
	privkey := make([]byte, 64)
	for i := range privkey {
		privkey[i] = byte(i)
	}

	encrypted := CBCEncrypterPrivkey(password, privkey)
	assert.NotNil(t, encrypted)
	assert.Equal(t, aes.BlockSize+len(privkey), len(encrypted))

	decrypted := CBCDecrypterPrivkey(password, encrypted)
	assert.Equal(t, privkey, decrypted)
}

// legacyEncrypt 复现引入随机 IV 之前的旧格式：固定 IV = key[:16]，密文不前置 IV。
func legacyEncrypt(password []byte, plain []byte) []byte {
	key := make([]byte, 32)
	if len(password) > 32 {
		key = password[0:32]
	} else {
		copy(key, password)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	iv := key[:block.BlockSize()]
	encrypted := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(encrypted, plain)
	return encrypted
}

// TestCBCDecryptLegacyFormat 验证旧格式（固定 IV、密文不前置 IV）仍能正确解密，
// 保证旧钱包持久化数据向后兼容。
func TestCBCDecryptLegacyFormat(t *testing.T) {
	password := []byte("test-password-12345678901234567890")

	legacy32 := make([]byte, 32)
	for i := range legacy32 {
		legacy32[i] = byte(255 - i)
	}
	dec32 := CBCDecrypterPrivkey(password, legacyEncrypt(password, legacy32))
	assert.Equal(t, legacy32, dec32)

	legacy64 := make([]byte, 64)
	for i := range legacy64 {
		legacy64[i] = byte(i + 1)
	}
	dec64 := CBCDecrypterPrivkey(password, legacyEncrypt(password, legacy64))
	assert.Equal(t, legacy64, dec64)
}

func TestCBCEncryptWrongDecrypt(t *testing.T) {
	password1 := []byte("password-one-1234567890123456")
	password2 := []byte("password-two-1234567890123456")
	privkey := []byte{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
	}

	encrypted := CBCEncrypterPrivkey(password1, privkey)
	decrypted := CBCDecrypterPrivkey(password2, encrypted)
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
