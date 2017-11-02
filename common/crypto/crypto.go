package crypto

import (
	"fmt"
)

type PrivKey interface {
	Bytes() []byte
	Sign(msg []byte) Signature
	PubKey() PubKey
	Equals(PrivKey) bool
}

type Signature interface {
	Bytes() []byte
	IsZero() bool
	String() string
	Equals(Signature) bool
}

type PubKey interface {
	Bytes() []byte
	KeyString() string
	VerifyBytes(msg []byte, sig Signature) bool
	Equals(PubKey) bool
}

type Crypto interface {
	GenKey() (PrivKey, error)
	SignatureFromBytes([]byte) (Signature, error)
	PrivKeyFromBytes([]byte) (PrivKey, error)
	PubKeyFromBytes([]byte) (PubKey, error)
}

var (
	drivers = make(map[string]Crypto)
)

func Register(name string, driver Crypto) {
	if driver == nil {
		panic("crypto: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("crypto: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

func New(name string) (c Crypto, err error) {
	c, ok := drivers[name]
	if !ok {
		err = fmt.Errorf("unknown driver %q", name)
		return
	}

	return c, nil
}
