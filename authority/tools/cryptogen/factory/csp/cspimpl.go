package csp

import (
	"crypto/elliptic"
	"reflect"

	"github.com/pkg/errors"
)

func New(keyStore KeyStore) (CSP, error) {
	signers := make(map[reflect.Type]Signer)
	signers[reflect.TypeOf(&ecdsaPrivateKey{})] = &ecdsaSigner{}

	impl := &cspimpl{
		ks:      keyStore,
		signers: signers}

	keyGenerators := make(map[int]KeyGenerator)
	keyGenerators[ECDSAP256KeyGen] = &ecdsaKeyGenerator{curve: elliptic.P256()}
	impl.keyGenerators = keyGenerators

	keyImporters := make(map[int]KeyImporter)
	keyImporters[ECDSAPKIXPublicKeyImport] = &ecdsaPKIXPublicKeyImportOptsKeyImporter{}
	keyImporters[ECDSAPrivateKeyImport] = &ecdsaPrivateKeyImportOptsKeyImporter{}
	keyImporters[ECDSAGoPublicKeyImport] = &ecdsaGoPublicKeyImportOptsKeyImporter{}

	impl.keyImporters = keyImporters

	return impl, nil
}

type cspimpl struct {
	ks KeyStore

	keyGenerators map[int]KeyGenerator
	keyImporters  map[int]KeyImporter
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

func (csp *cspimpl) KeyImport(raw interface{}, opts int) (k Key, err error) {
	if raw == nil {
		return nil, errors.New("Invalid raw. It must not be nil.")
	}

	keyImporter, found := csp.keyImporters[opts]
	if !found {
		return nil, errors.Errorf("Unsupported 'KeyImportOpts' provided [%v]", opts)
	}

	k, err = keyImporter.KeyImport(raw, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed importing key with opts [%v]", opts)
	}

	return
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

func (csp *cspimpl) GetKey(ski []byte) (k Key, err error) {
	k, err = csp.ks.GetKey(ski)
	if err != nil {
		return nil, err
	}

	return
}
