package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/tjfoc/gmsm/sm2"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/cert/authority/utils"
	sm2_util "gitlab.33.cn/chain33/chain33/system/crypto/sm2"
	"gitlab.33.cn/chain33/chain33/types"
)

type gmValidator struct {
	rootCerts []*sm2.Certificate

	intermediateCerts []*sm2.Certificate

	certificationTreeInternalNodesMap map[string]bool

	opts *sm2.VerifyOptions

	CRL []*pkix.CertificateList
}

func NewGmValidator() Validator {
	return &gmValidator{}
}

func (validator *gmValidator) getCertFromPem(idBytes []byte) (*sm2.Certificate, error) {
	if idBytes == nil {
		return nil, fmt.Errorf("getIdentityFromConf error: nil idBytes")
	}

	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, fmt.Errorf("getIdentityFromBytes error: could not decode pem bytes [%v]", idBytes)
	}

	var cert *sm2.Certificate
	cert, err := sm2.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, fmt.Errorf("getIdentityFromBytes error: failed to parse sm2 cert, err %s", err)
	}

	return cert, nil
}

func (validator *gmValidator) Setup(conf *AuthConfig) error {
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

func (validator *gmValidator) Validate(certByte []byte, pubKey []byte) error {
	authLogger.Debug("validating certificate")

	cert, err := validator.getCertFromPem(certByte)
	if err != nil {
		return fmt.Errorf("ParseCertificate failed %s", err)
	}

	certPubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("Error publick key type in transaction. expect SM2")
	}

	if !bytes.Equal(pubKey, sm2_util.SerializePublicKey(ParseECDSAPubKey2SM2PubKey(certPubKey))) {
		return fmt.Errorf("Invalid public key.")
	}

	validationChain, err := validator.getCertificationChain(cert)
	if err != nil {
		return fmt.Errorf("Could not obtain certification chain, err %s", err)
	}

	err = validator.validateCertAgainstChain(cert, validationChain)
	if err != nil {
		return fmt.Errorf("Could not validate identity against certification chain, err %s", err)
	}

	return nil
}

func (validator *gmValidator) getCertificationChain(cert *sm2.Certificate) ([]*sm2.Certificate, error) {
	if validator.opts == nil {
		return nil, errors.New("Invalid validator instance")
	}

	if cert.IsCA {
		return nil, errors.New("A CA certificate cannot be used directly by this validator")
	}

	return validator.getValidationChain(cert, false)
}

func (validator *gmValidator) getUniqueValidationChain(cert *sm2.Certificate, opts sm2.VerifyOptions) ([]*sm2.Certificate, error) {
	if validator.opts == nil {
		return nil, fmt.Errorf("The supplied identity has no verify options")
	}

	validationChains, err := cert.Verify(opts)
	if err != nil {
		return nil, fmt.Errorf("The supplied identity is not valid, Verify() returned %s", err)
	}

	if len(validationChains) != 1 {
		return nil, fmt.Errorf("This validator only supports a single validation chain, got %d", len(validationChains))
	}

	return validationChains[0], nil
}

func (validator *gmValidator) getValidationChain(cert *sm2.Certificate, isIntermediateChain bool) ([]*sm2.Certificate, error) {
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
		return nil, fmt.Errorf("Invalid validation chain. Parent certificate should be a leaf of the certification tree [%v].", cert.Raw)
	}
	return validationChain, nil
}

func (validator *gmValidator) setupCAs(conf *AuthConfig) error {
	if len(conf.RootCerts) == 0 {
		return errors.New("Expected at least one CA certificate")
	}

	validator.opts = &sm2.VerifyOptions{Roots: sm2.NewCertPool(), Intermediates: sm2.NewCertPool()}
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

	validator.rootCerts = make([]*sm2.Certificate, len(conf.RootCerts))
	for i, trustedCert := range conf.RootCerts {
		cert, err := validator.getCertFromPem(trustedCert)
		if err != nil {
			return err
		}

		validator.rootCerts[i] = cert
	}

	validator.intermediateCerts = make([]*sm2.Certificate, len(conf.IntermediateCerts))
	for i, trustedCert := range conf.IntermediateCerts {
		cert, err := validator.getCertFromPem(trustedCert)
		if err != nil {
			return err
		}

		validator.intermediateCerts[i] = cert
	}

	validator.opts = &sm2.VerifyOptions{Roots: sm2.NewCertPool(), Intermediates: sm2.NewCertPool()}
	for _, cert := range validator.rootCerts {
		validator.opts.Roots.AddCert(cert)
	}
	for _, cert := range validator.intermediateCerts {
		validator.opts.Intermediates.AddCert(cert)
	}

	return nil
}

func (validator *gmValidator) setupCRLs(conf *AuthConfig) error {
	validator.CRL = make([]*pkix.CertificateList, len(conf.RevocationList))
	for i, crlbytes := range conf.RevocationList {
		crl, err := sm2.ParseCRL(crlbytes)
		if err != nil {
			return fmt.Errorf("Could not parse RevocationList, err %s", err)
		}

		validator.CRL[i] = crl
	}

	return nil
}

func (validator *gmValidator) finalizeSetupCAs(config *AuthConfig) error {
	for _, cert := range append(append([]*sm2.Certificate{}, validator.rootCerts...), validator.intermediateCerts...) {
		if !isSm2CACert(cert) {
			return fmt.Errorf("CA Certificate did not have the Subject Key Identifier extension, (SN: %s)", cert.SerialNumber)
		}

		if err := validator.validateCAIdentity(cert); err != nil {
			return fmt.Errorf("CA Certificate is not valid, (SN: %s) [%s]", cert.SerialNumber, err)
		}
	}

	validator.certificationTreeInternalNodesMap = make(map[string]bool)
	for _, cert := range append([]*sm2.Certificate{}, validator.intermediateCerts...) {
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

func getSubjectKeyIdentifierFromSm2Cert(cert *sm2.Certificate) ([]byte, error) {
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

func isSm2CACert(cert *sm2.Certificate) bool {
	_, err := getSubjectKeyIdentifierFromSm2Cert(cert)
	if err != nil {
		return false
	}

	if !cert.IsCA {
		return false
	}

	return true
}

func (validator *gmValidator) validateCAIdentity(cert *sm2.Certificate) error {
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

	return validator.validateCertAgainstChain(cert, validationChain)
}

func (validator *gmValidator) validateCertAgainstChain(cert *sm2.Certificate, validationChain []*sm2.Certificate) error {
	SKI, err := getSubjectKeyIdentifierFromSm2Cert(validationChain[1])
	if err != nil {
		return fmt.Errorf("Could not obtain Subject Key Identifier for signer cert, err %s", err)
	}

	for _, crl := range validator.CRL {
		aki, err := getAuthorityKeyIdentifierFromCrl(crl)
		if err != nil {
			return fmt.Errorf("Could not obtain Authority Key Identifier for crl, err %s", err)
		}

		if bytes.Equal(aki, SKI) {
			for _, rc := range crl.TBSCertList.RevokedCertificates {
				if rc.SerialNumber.Cmp(cert.SerialNumber) == 0 {
					err = validationChain[1].CheckCRLSignature(crl)
					if err != nil {
						authLogger.Warn(fmt.Sprintf("Invalid signature over the identified CRL, error %s", err))
						continue
					}

					return errors.New("The certificate has been revoked")
				}
			}
		}
	}

	return nil
}

func (validator *gmValidator) getValidityOptsForCert(cert *sm2.Certificate) sm2.VerifyOptions {
	var tempOpts sm2.VerifyOptions
	tempOpts.Roots = validator.opts.Roots

	tempOpts.DNSName = validator.opts.DNSName
	tempOpts.Intermediates = validator.opts.Intermediates
	tempOpts.KeyUsages = validator.opts.KeyUsages
	tempOpts.CurrentTime = cert.NotBefore.Add(time.Second)

	return tempOpts
}

func (Validator *gmValidator) GetCertFromSignature(signature []byte) ([]byte, error) {
	// 从proto中解码signature
	cert, _, err := utils.DecodeCertFromSignature(signature)
	if err != nil {
		authLogger.Error(fmt.Sprintf("unmashal certificate from signature failed. %s", err.Error()))
		return nil, err
	}

	if len(cert) == 0 {
		authLogger.Error("cert can not be null")
		return nil, types.ErrInvalidParam
	}

	return cert, nil
}
