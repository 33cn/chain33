package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"math/big"

	"encoding/asn1"

	"fmt"

	"github.com/pkg/errors"
	"github.com/tjfoc/gmsm/sm2"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	ecdsa_util "gitlab.33.cn/chain33/chain33/plugin/crypto/ecdsa"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/cert/types"
	sm2_util "gitlab.33.cn/chain33/chain33/system/crypto/sm2"
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
	case ty.AUTH_ECDSA:
		x509Cert, err := x509.ParseCertificate(dcert.Bytes)
		if err != nil {
			return "", errors.Errorf("Unable to parse cert from decoded bytes: %s", err)
		}
		ecdsaPk := x509Cert.PublicKey.(*ecdsa.PublicKey)
		ski = SKI(ecdsaPk.Curve, ecdsaPk.X, ecdsaPk.Y)
	case ty.AUTH_SM2:
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

func EncodeCertToSignature(signByte []byte, cert []byte) ([]byte, error) {
	certSign := crypto.CertSignature{}
	certSign.Signature = append(certSign.Signature, signByte...)
	certSign.Cert = append(certSign.Cert, cert...)
	return asn1.Marshal(certSign)
}

func DecodeCertFromSignature(signByte []byte) ([]byte, []byte, error) {
	var certSignature crypto.CertSignature
	_, err := asn1.Unmarshal(signByte, &certSignature)
	if err != nil {
		return nil, nil, err
	}

	return certSignature.Cert, certSignature.Signature, nil
}

func PrivKeyByteFromRaw(raw []byte, signType int) ([]byte, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("Failed decoding PEM. Block must be different from nil. [% x]", raw)
	}

	switch signType {
	case ty.AUTH_ECDSA:
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return ecdsa_util.SerializePrivateKey(key.(*ecdsa.PrivateKey)), nil
	case ty.AUTH_SM2:
		key, err := sm2.ParsePKCS8PrivateKey(block.Bytes, nil)
		if err != nil {
			return nil, err
		}
		return sm2_util.SerializePrivateKey(key), nil
	}

	return nil, errors.Errorf("unknow public key type")
}
