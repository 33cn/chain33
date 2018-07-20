package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"math/big"

	"github.com/pkg/errors"
	"github.com/tjfoc/gmsm/sm2"
)

const (
	SIGN_TYPE_AUTHECDSA = 1
	SIGN_TYPE_AUTHSM2   = 2
)

func SKI(curve elliptic.Curve, x, y *big.Int) (ski []byte) {
	raw := elliptic.Marshal(curve, x, y)

	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

func GetPublicKeySKIFromCert(cert []byte, signType int) (string, error) {
	dcert, _ := pem.Decode(cert)
	if dcert == nil {
		return "", errors.Errorf("Unable to decode cert bytes [%v]", cert)
	}

	var ski []byte
	switch signType {
	case SIGN_TYPE_AUTHECDSA:
		x509Cert, err := x509.ParseCertificate(dcert.Bytes)
		if err != nil {
			return "", errors.Errorf("Unable to parse cert from decoded bytes: %s", err)
		}
		ecdsaPk := x509Cert.PublicKey.(*ecdsa.PublicKey)
		ski = SKI(ecdsaPk.Curve, ecdsaPk.X, ecdsaPk.Y)
	case SIGN_TYPE_AUTHSM2:
		sm2Cert, err := sm2.ParseCertificate(dcert.Bytes)
		if err != nil {
			return "", errors.Errorf("Unable to parse cert from decoded bytes: %s", err)
		}
		sm2Pk := sm2Cert.PublicKey.(*ecdsa.PublicKey)
		ski = SKI(sm2Pk.Curve, sm2Pk.X, sm2Pk.Y)
	default:
		return "", errors.Errorf("unknow public key type")
	}

	return hex.EncodeToString(ski), nil
}
