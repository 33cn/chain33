// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

// CBCEncrypterPrivkey 使用钱包的password对私钥进行aes cbc加密,返回加密后的privkey。
//
// 新格式: MagicPrivKey(4) + version(1) + salt(16) + iv(16) + ciphertext,
// 密钥由 pbkdf2 派生, 不再直接使用口令明文。旧数据仍可由 CBCDecrypterPrivkey 读取。
func CBCEncrypterPrivkey(password []byte, privkey []byte) []byte {
	salt, err := NewSalt()
	if err != nil {
		return nil
	}
	key := DeriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}

	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil
	}

	Encrypted := make([]byte, len(privkey))
	encrypter := cipher.NewCBCEncrypter(block, iv)
	encrypter.CryptBlocks(Encrypted, privkey)

	out := make([]byte, 0, len(MagicPrivKey)+1+len(salt)+len(iv)+len(Encrypted))
	out = append(out, MagicPrivKey...)
	out = append(out, KdfVersion)
	out = append(out, salt...)
	out = append(out, iv...)
	out = append(out, Encrypted...)
	return out
}

// CBCDecrypterPrivkey 使用钱包的password对私钥进行aes cbc解密,返回解密后的privkey。
//
// 依次尝试三种格式, 保证历史钱包数据可以继续读取:
//  1. 新格式 MagicPrivKey + version + salt + iv + ciphertext (pbkdf2 派生密钥)
//  2. 旧格式 iv(16) + ciphertext        (口令零填充为密钥, 随机 iv)
//  3. 最初格式 ciphertext, iv = key[:16] (口令零填充为密钥, 固定 iv)
func CBCDecrypterPrivkey(password []byte, privkey []byte) []byte {
	// 1. 新格式
	if hasMagic(privkey, MagicPrivKey) {
		offset := len(MagicPrivKey) + 1
		if len(privkey) < offset+KdfSaltLen+aes.BlockSize {
			return nil
		}
		salt := privkey[offset : offset+KdfSaltLen]
		rest := privkey[offset+KdfSaltLen:]
		block, err := aes.NewCipher(DeriveKey(password, salt))
		if err != nil {
			return nil
		}
		iv := rest[:aes.BlockSize]
		ciphertext := rest[aes.BlockSize:]
		if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
			return nil
		}
		decrypted := make([]byte, len(ciphertext))
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, ciphertext)
		return decrypted
	}

	// 旧格式统一使用口令零填充后的密钥
	key := LegacyKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	blockSize := block.BlockSize()

	// 2. 旧格式: iv(16) + ciphertext
	if len(privkey) > blockSize && len(privkey)%blockSize == 0 {
		iv := privkey[:blockSize]
		ciphertext := privkey[blockSize:]
		decrypted := make([]byte, len(ciphertext))
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, ciphertext)
		// 校验: 私钥固定 32 字节, 命中则认为格式判断正确
		if len(decrypted) == 32 {
			return decrypted
		}
	}

	// 3. 最初格式: iv = key[:BlockSize]
	iv := key[:blockSize]
	decryptered := make([]byte, len(privkey))
	decrypter := cipher.NewCBCDecrypter(block, iv)
	decrypter.CryptBlocks(decryptered, privkey)
	return decryptered
}
