package csp

import (
	"crypto/elliptic"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/tjfoc/gmsm/sm2"
)

type SM2PrivateKey struct {
	PrivKey *sm2.PrivateKey
}

func (k *SM2PrivateKey) Bytes() (raw []byte, err error) {
	return nil, errors.New("Not supported.")
}

func (k *SM2PrivateKey) SKI() (ski []byte) {
	if k.PrivKey == nil {
		return nil
	}

	raw := elliptic.Marshal(k.PrivKey.Curve, k.PrivKey.PublicKey.X, k.PrivKey.PublicKey.Y)

	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

func (k *SM2PrivateKey) Symmetric() bool {
	return false
}

func (k *SM2PrivateKey) Private() bool {
	return true
}

func (k *SM2PrivateKey) PublicKey() (Key, error) {
	return &SM2PublicKey{&k.PrivKey.PublicKey}, nil
}

type SM2PublicKey struct {
	PubKey *sm2.PublicKey
}

func (k *SM2PublicKey) Bytes() (raw []byte, err error) {
	raw, err = sm2.MarshalSm2PublicKey(k.PubKey)
	if err != nil {
		return nil, fmt.Errorf("Failed marshalling key [%s]", err)
	}
	return
}

func (k *SM2PublicKey) SKI() (ski []byte) {
	if k.PubKey == nil {
		return nil
	}

	raw := elliptic.Marshal(k.PubKey.Curve, k.PubKey.X, k.PubKey.Y)

	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

func (k *SM2PublicKey) Symmetric() bool {
	return false
}

func (k *SM2PublicKey) Private() bool {
	return false
}

func (k *SM2PublicKey) PublicKey() (Key, error) {
	return k, nil
}
