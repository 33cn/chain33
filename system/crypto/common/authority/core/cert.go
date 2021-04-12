// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"crypto/ecdsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/tjfoc/gmsm/sm2"
)

// Validity Validity
type Validity struct {
	NotBefore, NotAfter time.Time
}

// PublicKeyInfo PublicKeyInfo
type PublicKeyInfo struct {
	Raw       asn1.RawContent
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

// Certificate Certificate
type Certificate struct {
	Raw                asn1.RawContent
	TBSCertificate     TbsCertificate
	SignatureAlgorithm pkix.AlgorithmIdentifier
	SignatureValue     asn1.BitString
}

// TbsCertificate TbsCertificate
type TbsCertificate struct {
	Raw                asn1.RawContent
	Version            int `asn1:"optional,explicit,default:0,tag:0"`
	SerialNumber       *big.Int
	SignatureAlgorithm pkix.AlgorithmIdentifier
	Issuer             asn1.RawValue
	Validity           Validity
	Subject            asn1.RawValue
	PublicKey          PublicKeyInfo
	UniqueID           asn1.BitString   `asn1:"optional,tag:1"`
	SubjectUniqueID    asn1.BitString   `asn1:"optional,tag:2"`
	Extensions         []pkix.Extension `asn1:"optional,explicit,tag:3"`
}

// CertFromX509Cert x509格式转换
func CertFromX509Cert(cert *x509.Certificate) (Certificate, error) {
	var newCert Certificate
	_, err := asn1.Unmarshal(cert.Raw, &newCert)
	if err != nil {
		return Certificate{}, err
	}
	return newCert, nil
}

// IsECDSASignedCert 是否ecdsa证书
func IsECDSASignedCert(cert *x509.Certificate) bool {
	return cert.SignatureAlgorithm == x509.ECDSAWithSHA1 ||
		cert.SignatureAlgorithm == x509.ECDSAWithSHA256 ||
		cert.SignatureAlgorithm == x509.ECDSAWithSHA384 ||
		cert.SignatureAlgorithm == x509.ECDSAWithSHA512
}

// ParseECDSAPubKey2SM2PubKey 将ECDSA的公钥转成SM2公钥
func ParseECDSAPubKey2SM2PubKey(key *ecdsa.PublicKey) *sm2.PublicKey {
	sm2Key := &sm2.PublicKey{
		Curve: key.Curve,
		X:     key.X,
		Y:     key.Y,
	}

	return sm2Key
}

type authorityKeyIdentifier struct {
	KeyIdentifier             []byte  `asn1:"optional,tag:0"`
	AuthorityCertIssuer       []byte  `asn1:"optional,tag:1"`
	AuthorityCertSerialNumber big.Int `asn1:"optional,tag:2"`
}

// GetAuthorityKeyIdentifierFromCrl  Crl
func GetAuthorityKeyIdentifierFromCrl(crl *pkix.CertificateList) ([]byte, error) {
	aki := authorityKeyIdentifier{}

	for _, ext := range crl.TBSCertList.Extensions {
		if reflect.DeepEqual(ext.Id, asn1.ObjectIdentifier{2, 5, 29, 35}) {
			_, err := asn1.Unmarshal(ext.Value, &aki)
			if err != nil {
				return nil, fmt.Errorf("Failed to unmarshal AKI, error %s", err)
			}

			return aki.KeyIdentifier, nil
		}
	}

	return nil, errors.New("authorityKeyIdentifier not found in certificate")
}
