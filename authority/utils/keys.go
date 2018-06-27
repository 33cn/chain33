package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"crypto/sha256"
	"encoding/hex"
	"github.com/pkg/errors"
)

func SKI(curve elliptic.Curve, x, y *big.Int) (ski []byte) {
	raw := elliptic.Marshal(curve, x, y)

	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

func GetPublicKeySKIFromCert(cert []byte) (string, error) {
	dcert, _ := pem.Decode(cert)
	if dcert == nil {
		return "", errors.Errorf("Unable to decode cert bytes [%v]", cert)
	}

	x509Cert, err := x509.ParseCertificate(dcert.Bytes)
	if err != nil {
		return "", errors.Errorf("Unable to parse cert from decoded bytes: %s", err)
	}

	var ski []byte
	pk := x509Cert.PublicKey
	switch pk.(type) {
	case *ecdsa.PublicKey:
		ecdsaPk := pk.(*ecdsa.PublicKey)
		ski = SKI(ecdsaPk.Curve, ecdsaPk.X, ecdsaPk.Y)
	default:
		return "", errors.Errorf("unknow public key type")
	}

	return hex.EncodeToString(ski), nil
}
