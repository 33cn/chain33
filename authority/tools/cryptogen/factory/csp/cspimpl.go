package csp

import (
	"crypto/elliptic"
	"reflect"

	"github.com/pkg/errors"
)

func New(keyStore KeyStore) (CSP, error) {
	signers := make(map[reflect.Type]Signer)
	signers[reflect.TypeOf(&ecdsaPrivateKey{})] = &ecdsaSigner{}
	signers[reflect.TypeOf(&SM2PrivateKey{})] = &sm2Signer{}

	impl := &cspimpl{
		ks:      keyStore,
		signers: signers}

	keyGenerators := make(map[int]KeyGenerator)
	keyGenerators[ECDSAP256KeyGen] = &ecdsaKeyGenerator{curve: elliptic.P256()}
	keyGenerators[SM2P256KygGen] = &sm2KeyGenerator{}
	impl.keyGenerators = keyGenerators

	return impl, nil
}

type cspimpl struct {
	ks KeyStore

	keyGenerators map[int]KeyGenerator
	signers       map[reflect.Type]Signer
}

func (csp *cspimpl) KeyGen(opts int) (k Key, err error) {
	keyGenerator, found := csp.keyGenerators[opts]
	if !found {
		return nil, errors.Errorf("Unsupported 'KeyGenOpts' provided [%v]", opts)
	}

	k, err = keyGenerator.KeyGen(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed generating key with opts [%v]", opts)
	}

	if csp.ks != nil {
		err = csp.ks.StoreKey(k)
		if err != nil {
			return nil, err
		}
	}

	return k, nil
}

func (csp *cspimpl) Sign(k Key, digest []byte, opts SignerOpts) (signature []byte, err error) {
	if k == nil {
		return nil, errors.New("Invalid Key. It must not be nil.")
	}
	if len(digest) == 0 {
		return nil, errors.New("Invalid digest. Cannot be empty.")
	}

	keyType := reflect.TypeOf(k)
	signer, found := csp.signers[keyType]
	if !found {
		return nil, errors.Errorf("Unsupported 'SignKey' provided [%s]", keyType)
	}

	signature, err = signer.Sign(k, digest, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed signing with opts [%v]", opts)
	}

	return
}
