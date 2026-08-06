// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"testing"

	wcom "github.com/33cn/chain33/wallet/common"
)

const testSeedText = "hurt vacuum bird rely gold sad decline moon vast prosper spend rich"

// legacySeedFixedNonce 复现最初格式: 口令零填充为密钥, nonce = key[:12], 无前置 nonce
func legacySeedFixedNonce(password, seed []byte) []byte {
	key := wcom.LegacyKey(password)
	block, _ := aes.NewCipher(key)
	aesgcm, _ := cipher.NewGCM(block)
	return aesgcm.Seal(nil, key[:12], seed, nil)
}

// legacySeedRandomNonce 复现旧格式: 口令零填充为密钥, 随机 nonce 前置
func legacySeedRandomNonce(password, seed []byte) []byte {
	key := wcom.LegacyKey(password)
	block, _ := aes.NewCipher(key)
	aesgcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err)
	}
	return append(nonce, aesgcm.Seal(nil, nonce, seed, nil)...)
}

// 新格式写入后必须能自行读回
func TestSeedNewFormatRoundTrip(t *testing.T) {
	pw := []byte("wallet-password")
	enc, err := AesgcmEncrypter(pw, []byte(testSeedText))
	if err != nil {
		t.Fatalf("AesgcmEncrypter err: %v", err)
	}
	if !wcom.HasSeedMagic(enc) {
		t.Fatal("新格式密文缺少魔数")
	}
	got, err := AesgcmDecrypter(pw, enc)
	if err != nil {
		t.Fatalf("AesgcmDecrypter err: %v", err)
	}
	if string(got) != testSeedText {
		t.Fatalf("新格式往返失败: got %q", got)
	}
}

// 新格式必须使用 KDF 派生密钥
func TestSeedNewFormatUsesKDF(t *testing.T) {
	pw := []byte("wallet-password")
	enc, err := AesgcmEncrypter(pw, []byte(testSeedText))
	if err != nil {
		t.Fatal(err)
	}
	offset := len(wcom.MagicSeed) + 1
	salt := enc[offset : offset+wcom.KdfSaltLen]
	if bytes.Equal(wcom.DeriveKey(pw, salt), wcom.LegacyKey(pw)) {
		t.Fatal("派生密钥等于口令零填充, KDF 未生效")
	}
}

// 每次加密都应使用新的随机盐
func TestSeedSaltIsRandom(t *testing.T) {
	pw := []byte("wallet-password")
	a, _ := AesgcmEncrypter(pw, []byte(testSeedText))
	b, _ := AesgcmEncrypter(pw, []byte(testSeedText))
	offset := len(wcom.MagicSeed) + 1
	if bytes.Equal(a[offset:offset+wcom.KdfSaltLen], b[offset:offset+wcom.KdfSaltLen]) {
		t.Fatal("两次加密使用了相同的盐")
	}
}

// 旧格式(随机 nonce 前置)必须仍可解密
func TestSeedLegacyRandomNonceCompat(t *testing.T) {
	pw := []byte("wallet-password")
	old := legacySeedRandomNonce(pw, []byte(testSeedText))
	got, err := AesgcmDecrypter(pw, old)
	if err != nil {
		t.Fatalf("旧格式(随机 nonce)解密失败: %v", err)
	}
	if string(got) != testSeedText {
		t.Fatalf("旧格式解密内容不符: got %q", got)
	}
}

// 最初格式(固定 nonce, 无前置 nonce)必须仍可解密
func TestSeedLegacyFixedNonceCompat(t *testing.T) {
	pw := []byte("wallet-password")
	old := legacySeedFixedNonce(pw, []byte(testSeedText))
	got, err := AesgcmDecrypter(pw, old)
	if err != nil {
		t.Fatalf("最初格式(固定 nonce)解密失败: %v", err)
	}
	if string(got) != testSeedText {
		t.Fatalf("最初格式解密内容不符: got %q", got)
	}
}

// 错误口令必须返回错误(GCM 有认证标签)
func TestSeedWrongPassword(t *testing.T) {
	enc, _ := AesgcmEncrypter([]byte("right-password"), []byte(testSeedText))
	if _, err := AesgcmDecrypter([]byte("wrong-password"), enc); err == nil {
		t.Fatal("错误口令未返回错误")
	}
}

// 超长口令(>32 字节)在新旧格式下都应可用
func TestSeedLongPassword(t *testing.T) {
	pw := bytes.Repeat([]byte("x"), 100)
	enc, err := AesgcmEncrypter(pw, []byte(testSeedText))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := AesgcmDecrypter(pw, enc); err != nil || string(got) != testSeedText {
		t.Fatalf("超长口令新格式往返失败: err=%v", err)
	}
	old := legacySeedRandomNonce(pw, []byte(testSeedText))
	if got, err := AesgcmDecrypter(pw, old); err != nil || string(got) != testSeedText {
		t.Fatalf("超长口令旧格式解密失败: err=%v", err)
	}
}
