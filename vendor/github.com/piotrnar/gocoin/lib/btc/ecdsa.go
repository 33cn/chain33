package btc

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"github.com/piotrnar/gocoin/lib/secp256k1"
	"math/big"
	"sync/atomic"
)

var (
	EcdsaVerifyCnt uint64
	EC_Verify      func(k, s, h []byte) bool
)

func EcdsaVerify(kd []byte, sd []byte, hash []byte) bool {
	atomic.AddUint64(&EcdsaVerifyCnt, 1)
	if len(kd) == 0 || len(sd) == 0 {
		return false
	}
	if EC_Verify != nil {
		return EC_Verify(kd, sd, hash)
	}
	return secp256k1.Verify(kd, sd, hash)
}

func EcdsaSign(priv, hash []byte) (r, s *big.Int, err error) {
	var sig secp256k1.Signature
	var sec, msg, nonce secp256k1.Number

	sec.SetBytes(priv)
	msg.SetBytes(hash)

	sha := sha256.New()
	sha.Write(priv)
	sha.Write(hash)
	for {
		var buf [32]byte
		rand.Read(buf[:])
		sha.Write(buf[:])
		nonce.SetBytes(sha.Sum(nil))
		if nonce.Sign() > 0 && nonce.Cmp(&secp256k1.TheCurve.Order.Int) < 0 {
			break
		}
	}

	if sig.Sign(&sec, &msg, &nonce, nil) != 1 {
		err = errors.New("ESCDS Sign error()")
	}
	return &sig.R.Int, &sig.S.Int, nil
}
