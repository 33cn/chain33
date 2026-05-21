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

// CBCEncrypterPrivkey 使用钱包的password对私钥进行aes cbc加密,返回加密后的privkey
func CBCEncrypterPrivkey(password []byte, privkey []byte) []byte {
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

	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil
	}

	Encrypted := make([]byte, len(privkey))
	encrypter := cipher.NewCBCEncrypter(block, iv)
	encrypter.CryptBlocks(Encrypted, privkey)

	return append(iv, Encrypted...)
}

// CBCDecrypterPrivkey 使用钱包的password对私钥进行aes cbc解密,返回解密后的privkey
func CBCDecrypterPrivkey(password []byte, privkey []byte) []byte {
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

	blockSize := block.BlockSize()

	// New format: IV(16) + ciphertext
	if len(privkey) > blockSize && len(privkey)%blockSize == 0 {
		iv := privkey[:blockSize]
		ciphertext := privkey[blockSize:]
		decrypted := make([]byte, len(ciphertext))
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, ciphertext)
		// Validate: if decryption looks valid, return it
		if len(decrypted) == 32 {
			return decrypted
		}
	}

	// Legacy format: IV = key[:BlockSize]
	iv := key[:blockSize]
	decryptered := make([]byte, len(privkey))
	decrypter := cipher.NewCBCDecrypter(block, iv)
	decrypter.CryptBlocks(decryptered, privkey)
	return decryptered
}
