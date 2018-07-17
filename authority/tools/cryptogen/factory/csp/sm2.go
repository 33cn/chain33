package csp

import (
	"math/big"
	"encoding/asn1"
	"github.com/tjfoc/gmsm/sm2"
	"fmt"
)

type sm2Signer struct{}

func (s *sm2Signer) Sign(k Key, digest []byte, opts SignerOpts) (signature []byte, err error) {
	return signSM2(k.(*SM2PrivateKey).PrivKey, digest, opts)
}

func signSM2(k *sm2.PrivateKey, digest []byte, opts SignerOpts) (signature []byte, err error) {
	r, s, err := sm2.Sign(k, digest)
	if err != nil {
		return nil, err
	}

	return MarshalSM2Signature(r, s)
}

type SM2Signature struct {
	R, S *big.Int
}

func MarshalSM2Signature(r, s *big.Int) ([]byte, error) {
	return asn1.Marshal(SM2Signature{r, s})
}

type sm2KeyGenerator struct {
}

func (kg *sm2KeyGenerator) KeyGen(opts int) (k Key, err error) {
	privKey, err := sm2.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("Failed generating SM2 key for: [%s]", err)
	}

	return &SM2PrivateKey{privKey}, nil
}
