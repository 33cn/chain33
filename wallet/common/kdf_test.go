// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"testing"
)

var testPrivkey = bytes.Repeat([]byte{0xAB}, 32)

// legacyEncryptFixedIV 复现最初格式: 口令零填充为密钥, iv = key[:16], 无前置 iv
func legacyEncryptFixedIV(password, privkey []byte) []byte {
	key := LegacyKey(password)
	block, _ := aes.NewCipher(key)
	out := make([]byte, len(privkey))
	cipher.NewCBCEncrypter(block, key[:block.BlockSize()]).CryptBlocks(out, privkey)
	return out
}

// legacyEncryptRandomIV 复现旧格式: 口令零填充为密钥, 随机 iv 前置
func legacyEncryptRandomIV(password, privkey []byte) []byte {
	key := LegacyKey(password)
	block, _ := aes.NewCipher(key)
	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	out := make([]byte, len(privkey))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, privkey)
	return append(iv, out...)
}

// 新格式写入后必须能自行读回
func TestPrivkeyNewFormatRoundTrip(t *testing.T) {
	pw := []byte("wallet-password")
	enc := CBCEncrypterPrivkey(pw, testPrivkey)
	if enc == nil {
		t.Fatal("CBCEncrypterPrivkey 返回 nil")
	}
	if !hasMagic(enc, MagicPrivKey) {
		t.Fatal("新格式密文缺少魔数")
	}
	got := CBCDecrypterPrivkey(pw, enc)
	if !bytes.Equal(got, testPrivkey) {
		t.Fatalf("新格式往返失败: got %x want %x", got, testPrivkey)
	}
}

// 新格式必须使用 KDF 派生密钥, 而不是口令零填充
func TestPrivkeyNewFormatUsesKDF(t *testing.T) {
	pw := []byte("wallet-password")
	enc := CBCEncrypterPrivkey(pw, testPrivkey)

	offset := len(MagicPrivKey) + 1
	salt := enc[offset : offset+KdfSaltLen]
	if bytes.Equal(DeriveKey(pw, salt), LegacyKey(pw)) {
		t.Fatal("派生密钥等于口令零填充, KDF 未生效")
	}

	// 用历史方式派生的密钥不应能解开新格式密文
	rest := enc[offset+KdfSaltLen:]
	block, _ := aes.NewCipher(LegacyKey(pw))
	wrong := make([]byte, len(rest)-aes.BlockSize)
	cipher.NewCBCDecrypter(block, rest[:aes.BlockSize]).CryptBlocks(wrong, rest[aes.BlockSize:])
	if bytes.Equal(wrong, testPrivkey) {
		t.Fatal("口令零填充密钥能解开新格式密文")
	}
}

// 每次加密都应使用新的随机盐, 避免相同口令产出相同密文
func TestPrivkeySaltIsRandom(t *testing.T) {
	pw := []byte("wallet-password")
	a := CBCEncrypterPrivkey(pw, testPrivkey)
	b := CBCEncrypterPrivkey(pw, testPrivkey)
	offset := len(MagicPrivKey) + 1
	if bytes.Equal(a[offset:offset+KdfSaltLen], b[offset:offset+KdfSaltLen]) {
		t.Fatal("两次加密使用了相同的盐")
	}
	if bytes.Equal(a, b) {
		t.Fatal("两次加密产出了相同密文")
	}
}

// 旧格式(随机 iv 前置)必须仍可解密
func TestPrivkeyLegacyRandomIVCompat(t *testing.T) {
	pw := []byte("wallet-password")
	old := legacyEncryptRandomIV(pw, testPrivkey)
	got := CBCDecrypterPrivkey(pw, old)
	if !bytes.Equal(got, testPrivkey) {
		t.Fatalf("旧格式(随机 iv)解密失败: got %x want %x", got, testPrivkey)
	}
}

// 最初格式(固定 iv, 无前置 iv)必须仍可解密
func TestPrivkeyLegacyFixedIVCompat(t *testing.T) {
	pw := []byte("wallet-password")
	old := legacyEncryptFixedIV(pw, testPrivkey)
	got := CBCDecrypterPrivkey(pw, old)
	if !bytes.Equal(got, testPrivkey) {
		t.Fatalf("最初格式(固定 iv)解密失败: got %x want %x", got, testPrivkey)
	}
}

// 错误口令不应解出正确私钥
func TestPrivkeyWrongPassword(t *testing.T) {
	enc := CBCEncrypterPrivkey([]byte("right-password"), testPrivkey)
	got := CBCDecrypterPrivkey([]byte("wrong-password"), enc)
	if bytes.Equal(got, testPrivkey) {
		t.Fatal("错误口令解出了正确私钥")
	}
}

// 超长口令(>32 字节)在新旧格式下都应可用
func TestPrivkeyLongPassword(t *testing.T) {
	pw := bytes.Repeat([]byte("x"), 100)
	enc := CBCEncrypterPrivkey(pw, testPrivkey)
	if got := CBCDecrypterPrivkey(pw, enc); !bytes.Equal(got, testPrivkey) {
		t.Fatal("超长口令新格式往返失败")
	}
	old := legacyEncryptRandomIV(pw, testPrivkey)
	if got := CBCDecrypterPrivkey(pw, old); !bytes.Equal(got, testPrivkey) {
		t.Fatal("超长口令旧格式解密失败")
	}
}

// 派生密钥缓存必须区分不同盐, 不能串号
func TestDeriveKeyCacheDistinguishesSalt(t *testing.T) {
	pw := []byte("wallet-password")
	s1, _ := NewSalt()
	s2, _ := NewSalt()
	k1 := DeriveKey(pw, s1)
	k2 := DeriveKey(pw, s2)
	if bytes.Equal(k1, k2) {
		t.Fatal("不同盐派生出相同密钥")
	}
	// 同盐重复派生应稳定返回同一结果(命中缓存)
	if !bytes.Equal(DeriveKey(pw, s1), k1) {
		t.Fatal("同盐重复派生结果不一致")
	}
}

// 派生密钥缓存必须区分不同口令
func TestDeriveKeyCacheDistinguishesPassword(t *testing.T) {
	salt, _ := NewSalt()
	if bytes.Equal(DeriveKey([]byte("pw-a"), salt), DeriveKey([]byte("pw-b"), salt)) {
		t.Fatal("不同口令派生出相同密钥")
	}
}

// TestPrivkeyForgedMagicRejected 验证旧数据即便恰好带有新格式前缀(魔数+版本),
// 也不会被误解析为新格式。新格式总长为 21(头)+16(iv)+密钥, 即 69 或 101 字节,
// 而旧格式长度必为 AES 分组倍数(32/48/64/80), 两者不相交, 长度校验会拒绝。
func TestPrivkeyForgedMagicRejected(t *testing.T) {
	password := []byte("forged-magic-password")
	for _, total := range []int{32, 48, 64, 80} {
		blob := make([]byte, total)
		copy(blob, MagicPrivKey)
		blob[len(MagicPrivKey)] = KdfVersion

		if !hasMagic(blob, MagicPrivKey) {
			t.Fatalf("len=%d 前缀构造失败", total)
		}
		if got := CBCDecrypterPrivkey(password, blob); got != nil {
			t.Errorf("len=%d 带魔数前缀的旧数据被误解析, 返回 %d 字节", total, len(got))
		}
	}
}

// TestPrivkeyNewFormatLengthDisjoint 固化"新格式长度不是 AES 分组倍数"这一前提。
// 该性质是新旧格式得以区分的基础, 若后续调整格式头长度使其变成分组倍数,
// 旧数据就可能通过新格式的长度校验, 这里作为护栏。
func TestPrivkeyNewFormatLengthDisjoint(t *testing.T) {
	header := len(MagicPrivKey) + 1 + KdfSaltLen + aes.BlockSize
	for _, keySize := range []int{32, 64} {
		if (header+keySize)%aes.BlockSize == 0 {
			t.Errorf("新格式总长 %d 为 AES 分组倍数, 与旧格式长度可能碰撞", header+keySize)
		}
	}
}

// TestPrivkeyEd25519NewFormat 覆盖 64 字节 ed25519 私钥走新格式的往返,
// 防止新格式只按 32 字节私钥实现而遗漏 privacy 钱包的密钥长度。
func TestPrivkeyEd25519NewFormat(t *testing.T) {
	password := []byte("ed25519-password")
	privkey := bytes.Repeat([]byte{0x5A}, 64)

	encrypted := CBCEncrypterPrivkey(password, privkey)
	if encrypted == nil {
		t.Fatal("加密失败")
	}
	if !bytes.Equal(CBCDecrypterPrivkey(password, encrypted), privkey) {
		t.Fatal("64 字节私钥新格式往返失败")
	}
}
