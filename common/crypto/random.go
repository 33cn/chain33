// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"encoding/hex"
	"io"
	"sync"
)

var gRandInfo *randInfo

func init() {
	gRandInfo = &randInfo{}
	gRandInfo.MixEntropy(randBytes(32)) // Init
}

// MixEntropy Mix additional bytes of randomness, e.g. from hardware, user-input, etc.
// It is OK to call it multiple times.  It does not diminish security.
func MixEntropy(seedBytes []byte) {
	gRandInfo.MixEntropy(seedBytes)
}

// This only uses the OS's randomness
func randBytes(numBytes int) []byte {
	b := make([]byte, numBytes)
	_, err := crand.Read(b)
	if err != nil {
		panic("Panic on a Crisis" + err.Error())
	}
	return b
}

// CRandBytes This uses the OS and the Seed(s).
func CRandBytes(numBytes int) []byte {
	b := make([]byte, numBytes)
	_, err := gRandInfo.Read(b)
	if err != nil {
		panic("Panic on a Crisis" + err.Error())
	}
	return b
}

// CRandHex RandHex(24) gives 96 bits of randomness, strong enough for most purposes.
func CRandHex(numDigits int) string {
	return hex.EncodeToString(CRandBytes(numDigits / 2))
}

// CReader Returns a crand.Reader mixed with user-supplied entropy
func CReader() io.Reader {
	return gRandInfo
}

//--------------------------------------------------------------------------------

type randInfo struct {
	mtx          sync.Mutex
	seedBytes    [32]byte
	cipherAES256 cipher.Block
	streamAES256 cipher.Stream
	reader       io.Reader
}

// MixEntropy You can call this as many times as you'd like.
// XXX TODO review
func (ri *randInfo) MixEntropy(seedBytes []byte) {
	ri.mtx.Lock()
	defer ri.mtx.Unlock()
	// Make new ri.seedBytes
	hashBytes := Sha256(seedBytes)
	hashBytes32 := [32]byte{}
	copy(hashBytes32[:], hashBytes)
	ri.seedBytes = xorBytes32(ri.seedBytes, hashBytes32)
	// Create new cipher.Block
	var err error
	ri.cipherAES256, err = aes.NewCipher(ri.seedBytes[:])
	if err != nil {
		panic("Error creating AES256 cipher: " + err.Error())
	}
	// Create new stream
	ri.streamAES256 = cipher.NewCTR(ri.cipherAES256, randBytes(aes.BlockSize))
	// Create new reader
	ri.reader = &cipher.StreamReader{S: ri.streamAES256, R: crand.Reader}
}

func (ri *randInfo) Read(b []byte) (n int, err error) {
	ri.mtx.Lock()
	defer ri.mtx.Unlock()
	return ri.reader.Read(b)
}

func xorBytes32(bytesA [32]byte, bytesB [32]byte) (res [32]byte) {
	for i, b := range bytesA {
		res[i] = b ^ bytesB[i]
	}
	return res
}
