package crypto

import (
	"fmt"
	"sync"
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

var driverMutex sync.Mutex

const (
	SignTypeInvalid        = 0
	SignTypeSecp256k1      = 1
	SignTypeED25519        = 2
	SignTypeSM2            = 3
	SignTypeOnetimeED25519 = 4
	SignTypeRing           = 5
)

const (
	SignNameSecp256k1      = "secp256k1"
	SignNameED25519        = "ed25519"
	SignNameSM2            = "sm2"
	SignNameOnetimeED25519 = "onetimeed25519"
	SignNameRing           = "RingSignatue"
)

var MapSignType2name = map[int]string{
	SignTypeSecp256k1:      SignNameSecp256k1,
	SignTypeED25519:        SignNameED25519,
	SignTypeSM2:            SignNameSM2,
	SignTypeOnetimeED25519: SignNameOnetimeED25519,
	SignTypeRing:           SignNameRing,
}

var MapSignName2Type = map[string]int{
	SignNameSecp256k1:      SignTypeSecp256k1,
	SignNameED25519:        SignTypeED25519,
	SignNameSM2:            SignTypeSM2,
	SignNameOnetimeED25519: SignTypeOnetimeED25519,
	SignNameRing:           SignTypeRing,
}

func Register(name string, driver Crypto) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	if driver == nil {
		panic("crypto: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("crypto: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

func New(name string) (c Crypto, err error) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	c, ok := drivers[name]
	if !ok {
		err = fmt.Errorf("unknown driver %q", name)
		return
	}

	return c, nil
}
