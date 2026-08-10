// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"

	"golang.org/x/crypto/pbkdf2"
)

// 钱包加密密钥派生相关常量。
//
// 历史格式直接把用户口令零填充为 32 字节当作 AES 密钥，没有任何拉伸，
// 拿到 wallet.db 即可以「一次 AES 运算」的代价离线爆破口令。
// 新格式改用 pbkdf2-sha256 + 随机盐派生密钥，并在密文前写入魔数标识，
// 使解密侧能够区分新旧格式，从而在不破坏历史数据的前提下升级安全强度。
const (
	// KdfIterations pbkdf2 迭代次数, 参考 OWASP 对 PBKDF2-SHA256 的建议值
	KdfIterations = 210000
	// KdfSaltLen 随机盐长度
	KdfSaltLen = 16
	// KdfKeyLen 派生出的 AES-256 密钥长度
	KdfKeyLen = 32
	// KdfVersion 新格式版本号, 后续调整 KDF 参数时递增
	KdfVersion = byte(1)
)

var (
	// MagicPrivKey 新版私钥密文魔数
	MagicPrivKey = []byte("C33K")
	// MagicSeed 新版 seed 密文魔数
	MagicSeed = []byte("C33S")
)

// derivedKeyCache 缓存 (口令, 盐) -> 派生密钥。
// 私钥解密位于签名热路径上, 每次签名都重新跑一遍 pbkdf2 会带来约 20ms 的额外开销,
// 这里按口令与盐缓存派生结果, 使同一账户的重复签名只付一次派生成本。
var derivedKeyCache sync.Map

// DeriveKey 由口令与盐派生 AES 密钥, 结果带缓存。
func DeriveKey(password, salt []byte) []byte {
	sum := sha256.New()
	sum.Write(salt)
	sum.Write([]byte{0})
	sum.Write(password)
	cacheKey := hex.EncodeToString(sum.Sum(nil))

	if v, ok := derivedKeyCache.Load(cacheKey); ok {
		return v.([]byte)
	}
	key := pbkdf2.Key(password, salt, KdfIterations, KdfKeyLen, sha256.New)
	derivedKeyCache.Store(cacheKey, key)
	return key
}

// LegacyKey 复现历史的「口令零填充」密钥派生方式, 仅用于解密旧数据。
func LegacyKey(password []byte) []byte {
	key := make([]byte, KdfKeyLen)
	if len(password) > KdfKeyLen {
		copy(key, password[:KdfKeyLen])
	} else {
		copy(key, password)
	}
	return key
}

// NewSalt 生成随机盐
func NewSalt() ([]byte, error) {
	salt := make([]byte, KdfSaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// HasSeedMagic 判断 seed 密文是否为新版 KDF 格式
func HasSeedMagic(data []byte) bool {
	return hasMagic(data, MagicSeed)
}

// hasMagic 判断密文是否携带指定魔数与可识别的版本号
func hasMagic(data, magic []byte) bool {
	if len(data) < len(magic)+1 {
		return false
	}
	for i := range magic {
		if data[i] != magic[i] {
			return false
		}
	}
	return data[len(magic)] == KdfVersion
}
