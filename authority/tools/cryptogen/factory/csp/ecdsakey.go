package csp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
)

type ecdsaPrivateKey struct {
	privKey *ecdsa.PrivateKey
}

func (k *ecdsaPrivateKey) Bytes() (raw []byte, err error) {
	return nil, errors.New("Not supported.")
}

func (k *ecdsaPrivateKey) SKI() (ski []byte) {
	if k.privKey == nil {
		return nil
	}

	raw := elliptic.Marshal(k.privKey.Curve, k.privKey.PublicKey.X, k.privKey.PublicKey.Y)

	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

func (k *ecdsaPrivateKey) Symmetric() bool {
	return false
}

func (k *ecdsaPrivateKey) Private() bool {
	return true
}

func (k *ecdsaPrivateKey) PublicKey() (Key, error) {
	return &ecdsaPublicKey{&k.privKey.PublicKey}, nil
}

type ecdsaPublicKey struct {
	pubKey *ecdsa.PublicKey
}

func (k *ecdsaPublicKey) Bytes() (raw []byte, err error) {
	raw, err = x509.MarshalPKIXPublicKey(k.pubKey)
	if err != nil {
		return nil, fmt.Errorf("Failed marshalling key [%s]", err)
	}
	return
}

func (k *ecdsaPublicKey) SKI() (ski []byte) {
	if k.pubKey == nil {
		return nil
	}

	raw := elliptic.Marshal(k.pubKey.Curve, k.pubKey.X, k.pubKey.Y)

	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

func (k *ecdsaPublicKey) Symmetric() bool {
	return false
}

func (k *ecdsaPublicKey) Private() bool {
	return false
}

func (k *ecdsaPublicKey) PublicKey() (Key, error) {
	return k, nil
}
