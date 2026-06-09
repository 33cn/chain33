package queue

import (
	"crypto/aes"
	"crypto/cipher"
	"testing"

	"github.com/stretchr/testify/assert"
)

// F-WAL-001: wallet/seed.go:210 uses key[:12] as AES-GCM nonce.
// The nonce is deterministic (derived from password), so encrypting the same
// seed twice with the same password produces identical ciphertext.
// AES-GCM security breaks catastrophically on nonce reuse.

func TestAESGCMNonceReuseBug(t *testing.T) {
	password := []byte("test-password-for-wallet")
	seed := []byte("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about")

	key := make([]byte, 32)
	copy(key, password)

	block, err := aes.NewCipher(key)
	assert.NoError(t, err)
	aesgcm, err := cipher.NewGCM(block)
	assert.NoError(t, err)

	// Buggy: nonce = key[:12], deterministic
	nonce := key[:12]
	enc1 := aesgcm.Seal(nil, nonce, seed, nil)
	enc2 := aesgcm.Seal(nil, nonce, seed, nil)

	// BUG: same password + same seed = identical ciphertext (nonce reuse)
	assert.Equal(t, enc1, enc2,
		"buggy: deterministic nonce produces identical ciphertext — nonce reuse")

	// With a random nonce, each encryption produces different ciphertext
	// (we simulate by using different nonces)
	nonce2 := make([]byte, 12)
	nonce2[0] = 0x01 // different nonce
	enc3 := aesgcm.Seal(nil, nonce2, seed, nil)

	assert.NotEqual(t, enc1, enc3,
		"fixed: different nonce produces different ciphertext")
}

// F-WAL-002: wallet/common/crypto.go:25 uses key[:BlockSize] as AES-CBC IV.
// The IV is deterministic (derived from password), so encrypting the same
// privkey twice with the same password produces identical ciphertext.

func TestAESCBCIVReuseBug(t *testing.T) {
	password := []byte("test-password-for-wallet")
	privkey := make([]byte, 32)
	for i := range privkey {
		privkey[i] = byte(i)
	}

	key := make([]byte, 32)
	copy(key, password)

	block, err := aes.NewCipher(key)
	assert.NoError(t, err)

	// Buggy: IV = key[:BlockSize], deterministic
	iv := key[:block.BlockSize()]
	enc1 := make([]byte, len(privkey))
	enc2 := make([]byte, len(privkey))

	cipher.NewCBCEncrypter(block, iv).CryptBlocks(enc1, privkey)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(enc2, privkey)

	// BUG: same password + same privkey = identical ciphertext (IV reuse)
	assert.Equal(t, enc1, enc2,
		"buggy: deterministic IV produces identical ciphertext — IV reuse")

	// With a random IV, each encryption produces different ciphertext
	iv2 := make([]byte, block.BlockSize())
	iv2[0] = 0x01 // different IV
	enc3 := make([]byte, len(privkey))
	cipher.NewCBCEncrypter(block, iv2).CryptBlocks(enc3, privkey)

	assert.NotEqual(t, enc1, enc3,
		"fixed: different IV produces different ciphertext")
}
