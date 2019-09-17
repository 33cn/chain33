// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bip32 A fully compliant implementation of the BIP0032 spec for Hierarchical Deterministic Bitcoin addresses
package bip32

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
)

const (
	// FirstHardenedChild FirstHardenedChild
	FirstHardenedChild = uint32(0x80000000)
	// PublicKeyCompressedLength 公钥压缩长度
	PublicKeyCompressedLength = 33
)

var (
	// PrivateWalletVersion 私钥钱包版本
	PrivateWalletVersion, _ = hex.DecodeString("0488ADE4")
	// PublicWalletVersion 公钥钱包版本
	PublicWalletVersion, _ = hex.DecodeString("0488B21E")
)

// Key Represents a bip32 extended key containing key data, chain code, parent information, and other meta data
type Key struct {
	Version     []byte // 4 bytes
	Depth       byte   // 1 bytes
	ChildNumber []byte // 4 bytes
	FingerPrint []byte // 4 bytes
	ChainCode   []byte // 32 bytes
	Key         []byte // 33 bytes
	IsPrivate   bool   // unserialized
}

// NewMasterKey Creates a new master extended key from a seed
func NewMasterKey(seed []byte) (*Key, error) {
	// Generate key and chaincode
	hmac := hmac.New(sha512.New, []byte("Bitcoin seed"))
	_, err := hmac.Write(seed)
	if err != nil {
		return nil, err
	}
	intermediary := hmac.Sum(nil)

	// Split it into our key and chain code
	keyBytes := intermediary[:32]
	chainCode := intermediary[32:]

	// Validate key
	err = validatePrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}

	// Create the key struct
	key := &Key{
		Version:     PrivateWalletVersion,
		ChainCode:   chainCode,
		Key:         keyBytes,
		Depth:       0x0,
		ChildNumber: []byte{0x00, 0x00, 0x00, 0x00},
		FingerPrint: []byte{0x00, 0x00, 0x00, 0x00},
		IsPrivate:   true,
	}

	return key, nil
}

// NewChildKey Derives a child key from a given parent as outlined by bip32
func (key *Key) NewChildKey(childIdx uint32) (*Key, error) {
	hardenedChild := childIdx >= FirstHardenedChild
	childIndexBytes := uint32Bytes(childIdx)

	// Fail early if trying to create hardned child from public key
	if !key.IsPrivate && hardenedChild {
		return nil, errors.New("Can't create hardened child for public key")
	}

	// Get intermediary to create key and chaincode from
	// Hardened children are based on the private key
	// NonHardened children are based on the public key
	var data []byte
	if hardenedChild {
		data = append([]byte{0x0}, key.Key...)
	} else if key.IsPrivate {
		data = publicKeyForPrivateKey(key.Key)
	} else {
		data = key.Key
	}
	data = append(data, childIndexBytes...)

	hmac := hmac.New(sha512.New, key.ChainCode)
	_, err := hmac.Write(data)
	if err != nil {
		return nil, err
	}
	intermediary := hmac.Sum(nil)

	// Create child Key with data common to all both scenarios
	childKey := &Key{
		ChildNumber: childIndexBytes,
		ChainCode:   intermediary[32:],
		Depth:       key.Depth + 1,
		IsPrivate:   key.IsPrivate,
	}

	// Bip32 CKDpriv
	if key.IsPrivate {
		childKey.Version = PrivateWalletVersion
		childKey.FingerPrint = hash160(publicKeyForPrivateKey(key.Key))[:4]
		childKey.Key = addPrivateKeys(intermediary[:32], key.Key)

		// Validate key
		err := validatePrivateKey(childKey.Key)
		if err != nil {
			return nil, err
		}
		// Bip32 CKDpub
	} else {
		keyBytes := publicKeyForPrivateKey(intermediary[:32])
		// Validate key
		err := validateChildPublicKey(keyBytes)
		if err != nil {
			return nil, err
		}

		childKey.Version = PublicWalletVersion
		childKey.FingerPrint = hash160(key.Key)[:4]
		childKey.Key = addPublicKeys(keyBytes, key.Key)
	}

	return childKey, nil
}

// PublicKey Create public version of key or return a copy; 'Neuter' function from the bip32 spec
func (key *Key) PublicKey() *Key {
	keyBytes := key.Key

	if key.IsPrivate {
		keyBytes = publicKeyForPrivateKey(keyBytes)
	}

	return &Key{
		Version:     PublicWalletVersion,
		Key:         keyBytes,
		Depth:       key.Depth,
		ChildNumber: key.ChildNumber,
		FingerPrint: key.FingerPrint,
		ChainCode:   key.ChainCode,
		IsPrivate:   false,
	}
}

// Serialize Serialized an Key to a 78 byte byte slice
func (key *Key) Serialize() []byte {
	// Private keys should be prepended with a single null byte
	keyBytes := key.Key
	if key.IsPrivate {
		keyBytes = append([]byte{0x0}, keyBytes...)
	}

	// Write fields to buffer in order
	buffer := new(bytes.Buffer)
	_, err := buffer.Write(key.Version)
	if err != nil {
		return nil
	}
	err = buffer.WriteByte(key.Depth)
	if err != nil {
		return nil
	}
	_, err = buffer.Write(key.FingerPrint)
	if err != nil {
		return nil
	}
	_, err = buffer.Write(key.ChildNumber)
	if err != nil {
		return nil
	}
	_, err = buffer.Write(key.ChainCode)
	if err != nil {
		return nil
	}
	_, err = buffer.Write(keyBytes)
	if err != nil {
		return nil
	}

	// Append the standard doublesha256 checksum
	serializedKey := addChecksumToBytes(buffer.Bytes())

	return serializedKey
}

// String Encode the Key in the standard Bitcoin base58 encoding
func (key *Key) String() string {
	return string(base58Encode(key.Serialize()))
}

// NewSeed Cryptographically secure seed
func NewSeed() ([]byte, error) {
	// Well that easy, just make go read 256 random bytes into a slice
	s := make([]byte, 256)
	_, err := rand.Read(s)
	return s, err
}
