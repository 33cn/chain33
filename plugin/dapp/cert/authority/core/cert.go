package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"math/big"
	"time"

	"github.com/tjfoc/gmsm/sm2"
	ecdsa_util "gitlab.33.cn/chain33/chain33/plugin/crypto/ecdsa"
)

type validity struct {
	NotBefore, NotAfter time.Time
}

type publicKeyInfo struct {
	Raw       asn1.RawContent
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

type certificate struct {
	Raw                asn1.RawContent
	TBSCertificate     tbsCertificate
	SignatureAlgorithm pkix.AlgorithmIdentifier
	SignatureValue     asn1.BitString
}

type tbsCertificate struct {
	Raw                asn1.RawContent
	Version            int `asn1:"optional,explicit,default:0,tag:0"`
	SerialNumber       *big.Int
	SignatureAlgorithm pkix.AlgorithmIdentifier
	Issuer             asn1.RawValue
	Validity           validity
	Subject            asn1.RawValue
	PublicKey          publicKeyInfo
	UniqueId           asn1.BitString   `asn1:"optional,tag:1"`
	SubjectUniqueId    asn1.BitString   `asn1:"optional,tag:2"`
	Extensions         []pkix.Extension `asn1:"optional,explicit,tag:3"`
}

func isECDSASignedCert(cert *x509.Certificate) bool {
	return cert.SignatureAlgorithm == x509.ECDSAWithSHA1 ||
		cert.SignatureAlgorithm == x509.ECDSAWithSHA256 ||
		cert.SignatureAlgorithm == x509.ECDSAWithSHA384 ||
		cert.SignatureAlgorithm == x509.ECDSAWithSHA512
}

func sanitizeECDSASignedCert(cert *x509.Certificate, parentCert *x509.Certificate) (*x509.Certificate, error) {
	if cert == nil {
		return nil, errors.New("Certificate must be different from nil.")
	}
	if parentCert == nil {
		return nil, errors.New("Parent certificate must be different from nil.")
	}

	expectedSig, err := signatureToLowS(parentCert.PublicKey.(*ecdsa.PublicKey), cert.Signature)
	if err != nil {
		return nil, err
	}

	if bytes.Equal(cert.Signature, expectedSig) {
		return cert, nil
	}

	var newCert certificate
	newCert, err = certFromX509Cert(cert)
	if err != nil {
		return nil, err
	}

	newCert.SignatureValue = asn1.BitString{Bytes: expectedSig, BitLength: len(expectedSig) * 8}

	newCert.Raw = nil
	newRaw, err := asn1.Marshal(newCert)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(newRaw)
}

func signatureToLowS(k *ecdsa.PublicKey, signature []byte) ([]byte, error) {
	r, s, err := ecdsa_util.UnmarshalECDSASignature(signature)
	if err != nil {
		return nil, err
	}

	s = ecdsa_util.ToLowS(k, s)
	return ecdsa_util.MarshalECDSASignature(r, s)
}

func certFromX509Cert(cert *x509.Certificate) (certificate, error) {
	var newCert certificate
	_, err := asn1.Unmarshal(cert.Raw, &newCert)
	if err != nil {
		return certificate{}, err
	}
	return newCert, nil
}

func ParseECDSAPubKey2SM2PubKey(key *ecdsa.PublicKey) *sm2.PublicKey {
	sm2Key := &sm2.PublicKey{
		key.Curve,
		key.X,
		key.Y,
	}

	return sm2Key
}
