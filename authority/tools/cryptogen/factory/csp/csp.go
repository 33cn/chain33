package csp

import "crypto"

type Key interface {
	Bytes() ([]byte, error)
	SKI() []byte
	Symmetric() bool
	Private() bool
	PublicKey() (Key, error)
}

type SignerOpts interface {
	crypto.SignerOpts
}

type CSP interface {
	KeyGen(opts int) (k Key, err error)
	KeyImport(raw interface{}, opts int) (k Key, err error)
	GetKey(ski []byte) (k Key, err error)
	Sign(k Key, digest []byte, opts SignerOpts) (signature []byte, err error)
}

type KeyStore interface {

	ReadOnly() bool

	GetKey(ski []byte) (k Key, err error)

	StoreKey(k Key) (err error)
}

type Signer interface {
	Sign(k Key, digest []byte, opts SignerOpts) (signature []byte, err error)
}

type KeyImporter interface {
	KeyImport(raw interface{}, opts int) (k Key, err error)
}

type KeyGenerator interface {
	KeyGen(opts int) (k Key, err error)
}