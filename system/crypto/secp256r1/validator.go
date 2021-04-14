// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secp256r1

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/33cn/chain33/system/crypto/common/authority/core"

	"github.com/33cn/chain33/system/crypto/common/authority/utils"

	"crypto/ecdsa"

	log "github.com/33cn/chain33/common/log/log15"
)

var authLogger = log.New("module", "crypto")

type ecdsaValidator struct {
	rootCerts []*x509.Certificate

	intermediateCerts []*x509.Certificate

	certificationTreeInternalNodesMap map[string]bool

	opts *x509.VerifyOptions

	CRL []*pkix.CertificateList
}

// NewEcdsaValidator 创建ecdsa校验器
func NewEcdsaValidator() core.Validator {
	return &ecdsaValidator{}
}

func (validator *ecdsaValidator) getCertFromPem(idBytes []byte) (*x509.Certificate, error) {
	if idBytes == nil {
		return nil, fmt.Errorf("getIdentityFromConf error: nil idBytes")
	}

	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, fmt.Errorf("getIdentityFromBytes error: could not decode pem bytes [%v]", idBytes)
	}

	var cert *x509.Certificate
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, fmt.Errorf("getIdentityFromBytes error: failed to parse x509 cert, err %s", err)
	}

	return cert, nil
}

func getSubjectKeyIdentifierFromCert(cert *x509.Certificate) ([]byte, error) {
	var SKI []byte

	for _, ext := range cert.Extensions {
		if reflect.DeepEqual(ext.Id, asn1.ObjectIdentifier{2, 5, 29, 14}) {
			_, err := asn1.Unmarshal(ext.Value, &SKI)
			if err != nil {
				return nil, fmt.Errorf("Failed to unmarshal Subject Key Identifier, err %s", err)
			}

			return SKI, nil
		}
	}

	return nil, errors.New("subjectKeyIdentifier not found in certificate")
}

func isCACert(cert *x509.Certificate) bool {
	_, err := getSubjectKeyIdentifierFromCert(cert)
	if err != nil {
		return false
	}

	if !cert.IsCA {
		return false
	}

	return true
}

func (validator *ecdsaValidator) Setup(conf *core.AuthConfig) error {
	if conf == nil {
		return fmt.Errorf("Setup error: nil conf reference")
	}

	if err := validator.setupCAs(conf); err != nil {
		return err
	}

	if err := validator.setupCRLs(conf); err != nil {
		return err
	}

	if err := validator.finalizeSetupCAs(conf); err != nil {
		return err
	}

	return nil
}

func (validator *ecdsaValidator) Validate(certByte []byte, pubKey []byte) error {
	authLogger.Debug("validating certificate")

	cert, err := validator.getCertFromPem(certByte)
	if err != nil {
		return fmt.Errorf("ParseCertificate failed %s", err)
	}

	certPubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("Error publick key type in transaction. expect ECDSA")
	}

	if !bytes.Equal(pubKey, SerializePublicKeyCompressed(certPubKey)) {
		return fmt.Errorf("Invalid public key")
	}

	cert, err = validator.sanitizeCert(cert)
	if err != nil {
		return fmt.Errorf("Sanitize certification failed. err %s", err)
	}

	validationChain, err := validator.getCertificationChain(cert)
	if err != nil {
		return fmt.Errorf("Could not obtain certification chain, err %s", err)
	}

	err = validator.validateCertAgainstChain(cert.SerialNumber, validationChain)
	if err != nil {
		return fmt.Errorf("Could not validate identity against certification chain, err %s", err)
	}

	return nil
}

func (validator *ecdsaValidator) getCertificationChain(cert *x509.Certificate) ([]*x509.Certificate, error) {
	if validator.opts == nil {
		return nil, errors.New("Invalid validator instance")
	}

	if cert.IsCA {
		return nil, errors.New("A CA certificate cannot be used directly")
	}

	return validator.getValidationChain(cert, false)
}

func (validator *ecdsaValidator) getUniqueValidationChain(cert *x509.Certificate, opts x509.VerifyOptions) ([]*x509.Certificate, error) {
	if validator.opts == nil {
		return nil, fmt.Errorf("The supplied identity has no verify options")
	}
	validationChains, err := cert.Verify(opts)
	if err != nil {
		return nil, fmt.Errorf("The supplied identity is not valid, Verify() returned %s", err)
	}

	if len(validationChains) != 1 {
		return nil, fmt.Errorf("Only supports a single validation chain, got %d", len(validationChains))
	}

	return validationChains[0], nil
}

func (validator *ecdsaValidator) getValidationChain(cert *x509.Certificate, isIntermediateChain bool) ([]*x509.Certificate, error) {
	validationChain, err := validator.getUniqueValidationChain(cert, validator.getValidityOptsForCert(cert))
	if err != nil {
		return nil, fmt.Errorf("Failed getting validation chain %s", err)
	}

	if len(validationChain) < 2 {
		return nil, fmt.Errorf("Expected a chain of length at least 2, got %d", len(validationChain))
	}

	parentPosition := 1
	if isIntermediateChain {
		parentPosition = 0
	}
	if validator.certificationTreeInternalNodesMap[string(validationChain[parentPosition].Raw)] {
		return nil, fmt.Errorf("Invalid validation chain. Parent certificate should be a leaf of the certification tree [%v]", cert.Raw)
	}
	return validationChain, nil
}

func (validator *ecdsaValidator) setupCAs(conf *core.AuthConfig) error {
	if len(conf.RootCerts) == 0 {
		return errors.New("Expected at least one CA certificate")
	}

	validator.opts = &x509.VerifyOptions{Roots: x509.NewCertPool(), Intermediates: x509.NewCertPool()}
	for _, v := range conf.RootCerts {
		cert, err := validator.getCertFromPem(v)
		if err != nil {
			return err
		}
		validator.opts.Roots.AddCert(cert)
	}
	for _, v := range conf.IntermediateCerts {
		cert, err := validator.getCertFromPem(v)
		if err != nil {
			return err
		}
		validator.opts.Intermediates.AddCert(cert)
	}

	validator.rootCerts = make([]*x509.Certificate, len(conf.RootCerts))
	for i, trustedCert := range conf.RootCerts {
		cert, err := validator.getCertFromPem(trustedCert)
		if err != nil {
			return err
		}
		cert, err = validator.sanitizeCert(cert)
		if err != nil {
			return err
		}

		validator.rootCerts[i] = cert
	}

	validator.intermediateCerts = make([]*x509.Certificate, len(conf.IntermediateCerts))
	for i, trustedCert := range conf.IntermediateCerts {
		cert, err := validator.getCertFromPem(trustedCert)
		if err != nil {
			return err
		}
		cert, err = validator.sanitizeCert(cert)
		if err != nil {
			return err
		}

		validator.intermediateCerts[i] = cert
	}

	validator.opts = &x509.VerifyOptions{Roots: x509.NewCertPool(), Intermediates: x509.NewCertPool()}
	for _, cert := range validator.rootCerts {
		validator.opts.Roots.AddCert(cert)
	}
	for _, cert := range validator.intermediateCerts {
		validator.opts.Intermediates.AddCert(cert)
	}

	return nil
}

func (validator *ecdsaValidator) setupCRLs(conf *core.AuthConfig) error {
	validator.CRL = make([]*pkix.CertificateList, len(conf.RevocationList))
	for i, crlbytes := range conf.RevocationList {
		crl, err := x509.ParseCRL(crlbytes)
		if err != nil {
			return fmt.Errorf("Could not parse RevocationList, err %s", err)
		}
		validator.CRL[i] = crl
	}

	return nil
}

func (validator *ecdsaValidator) finalizeSetupCAs(config *core.AuthConfig) error {
	for _, cert := range append(append([]*x509.Certificate{}, validator.rootCerts...), validator.intermediateCerts...) {
		if !isCACert(cert) {
			return fmt.Errorf("CA Certificate did not have the Subject Key Identifier extension, (SN: %s)", cert.SerialNumber)
		}

		if err := validator.validateCAIdentity(cert); err != nil {
			return fmt.Errorf("CA Certificate is not valid, (SN: %s) [%s]", cert.SerialNumber, err)
		}
	}

	validator.certificationTreeInternalNodesMap = make(map[string]bool)
	for _, cert := range append([]*x509.Certificate{}, validator.intermediateCerts...) {
		chain, err := validator.getUniqueValidationChain(cert, validator.getValidityOptsForCert(cert))
		if err != nil {
			return fmt.Errorf("Failed getting validation chain, (SN: %s)", cert.SerialNumber)
		}

		for i := 1; i < len(chain); i++ {
			validator.certificationTreeInternalNodesMap[string(chain[i].Raw)] = true
		}
	}

	return nil
}

func (validator *ecdsaValidator) sanitizeCert(cert *x509.Certificate) (*x509.Certificate, error) {
	if core.IsECDSASignedCert(cert) {
		var parentCert *x509.Certificate
		if cert.IsCA {
			chain, err := validator.getUniqueValidationChain(cert, validator.getValidityOptsForCert(cert))
			if err != nil {
				return nil, err
			}
			if len(chain) == 1 {
				parentCert = cert
			} else {
				parentCert = chain[1]
			}
		} else {
			chain, err := validator.getUniqueValidationChain(cert, validator.getValidityOptsForCert(cert))
			if err != nil {
				return nil, err
			}
			parentCert = chain[1]
		}

		var err error
		cert, err = sanitizeECDSASignedCert(cert, parentCert)
		if err != nil {
			return nil, err
		}
	}
	return cert, nil
}

func (validator *ecdsaValidator) validateCAIdentity(cert *x509.Certificate) error {
	if !cert.IsCA {
		return errors.New("Only CA identities can be validated")
	}

	validationChain, err := validator.getUniqueValidationChain(cert, validator.getValidityOptsForCert(cert))
	if err != nil {
		return fmt.Errorf("Could not obtain certification chain, err %s", err)
	}
	if len(validationChain) == 1 {
		return nil
	}

	return validator.validateCertAgainstChain(cert.SerialNumber, validationChain)
}

func (validator *ecdsaValidator) validateCertAgainstChain(serialNumber *big.Int, validationChain []*x509.Certificate) error {
	SKI, err := getSubjectKeyIdentifierFromCert(validationChain[1])
	if err != nil {
		return fmt.Errorf("Could not obtain Subject Key Identifier for signer cert, err %s", err)
	}

	for _, crl := range validator.CRL {
		aki, err := core.GetAuthorityKeyIdentifierFromCrl(crl)
		if err != nil {
			return fmt.Errorf("Could not obtain Authority Key Identifier for crl, err %s", err)
		}

		if bytes.Equal(aki, SKI) {
			for _, rc := range crl.TBSCertList.RevokedCertificates {
				if rc.SerialNumber.Cmp(serialNumber) == 0 {
					err = validationChain[1].CheckCRLSignature(crl)
					if err != nil {
						authLogger.Warn("Invalid signature over the identified CRL, error %s", err)
						continue
					}

					return errors.New("The certificate has been revoked")
				}
			}
		}
	}

	return nil
}

func (validator *ecdsaValidator) getValidityOptsForCert(cert *x509.Certificate) x509.VerifyOptions {
	var tempOpts x509.VerifyOptions
	tempOpts.Roots = validator.opts.Roots
	tempOpts.DNSName = validator.opts.DNSName
	tempOpts.Intermediates = validator.opts.Intermediates
	tempOpts.KeyUsages = validator.opts.KeyUsages
	tempOpts.CurrentTime = cert.NotBefore.Add(time.Second)

	return tempOpts
}

func (validator *ecdsaValidator) GetCertFromSignature(signature []byte) ([]byte, error) {
	certSign, err := utils.DecodeCertFromSignature(signature)
	if err != nil {
		authLogger.Error(fmt.Sprintf("unmashal certificate from signature failed. %s", err.Error()))
		return nil, err
	}

	if len(certSign.Cert) == 0 {
		authLogger.Error("cert can not be null")
		return nil, errors.New("ErrInvalidParam")
	}

	return certSign.Cert, nil
}

func sanitizeECDSASignedCert(cert *x509.Certificate, parentCert *x509.Certificate) (*x509.Certificate, error) {
	if cert == nil {
		return nil, errors.New("Certificate must be different from nil")
	}
	if parentCert == nil {
		return nil, errors.New("Parent certificate must be different from nil")
	}

	expectedSig, err := signatureToLowS(parentCert.PublicKey.(*ecdsa.PublicKey), cert.Signature)
	if err != nil {
		return nil, err
	}

	if bytes.Equal(cert.Signature, expectedSig) {
		return cert, nil
	}

	var newCert core.Certificate
	newCert, err = core.CertFromX509Cert(cert)
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
	r, s, err := UnmarshalECDSASignature(signature)
	if err != nil {
		return nil, err
	}

	s = ToLowS(k, s)
	return MarshalECDSASignature(r, s)
}
