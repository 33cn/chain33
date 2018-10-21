package signer

import (
	"crypto"
	"errors"
	"fmt"
	"io"

	lccsp "gitlab.33.cn/chain33/chain33/plugin/dapp/cert/authority/tools/cryptogen/factory/csp"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/cert/authority/tools/cryptogen/factory/utils"
)

type cspCryptoSigner struct {
	csp lccsp.CSP
	key lccsp.Key
	pk  interface{}
}

func New(csp lccsp.CSP, key lccsp.Key) (crypto.Signer, error) {
	if csp == nil {
		return nil, errors.New("bccsp instance must be different from nil.")
	}
	if key == nil {
		return nil, errors.New("key must be different from nil.")
	}
	if key.Symmetric() {
		return nil, errors.New("key must be asymmetric.")
	}

	pub, err := key.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed getting public key [%s]", err)
	}

	raw, err := pub.Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed marshalling public key [%s]", err)
	}

	pk, err := utils.DERToPublicKey(raw)
	if err != nil {
		return nil, fmt.Errorf("failed marshalling der to public key [%s]", err)
	}

	return &cspCryptoSigner{csp, key, pk}, nil
}

func (s *cspCryptoSigner) Public() crypto.PublicKey {
	return s.pk
}

func (s *cspCryptoSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	return s.csp.Sign(s.key, digest, opts)
}
