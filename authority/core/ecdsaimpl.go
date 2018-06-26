/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

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

	log "github.com/inconshreveable/log15"
	"crypto/sha256"
)

var authLogger = log.New("module", "autority")

type ecdsaValidator struct {
	// list of CA certs we trust
	rootCerts []*x509.Certificate

	// list of intermediate certs we trust
	intermediateCerts []*x509.Certificate

	// certificationTreeInternalNodesMap whose keys correspond to the raw material
	// (DER representation) of a certificate casted to a string, and whose values
	// are boolean. True means that the certificate is an internal node of the certification tree.
	// False means that the certificate corresponds to a leaf of the certification tree.
	certificationTreeInternalNodesMap map[string]bool

	// verification options
	opts *x509.VerifyOptions

	// list of certificate revocation lists
	CRL []*pkix.CertificateList
}

// NewBccspMsp returns an MSP instance backed up by a BCCSP
// crypto provider. It handles x.509 certificates and can
// generate identities and signing identities backed by
// certificates and keypairs
func NewEcdsaValidator() (Validator, error) {
	theValidator := &ecdsaValidator{}

	return theValidator, nil
}

func (validator *ecdsaValidator) getCertFromPem(idBytes []byte) (*x509.Certificate, error) {
	if idBytes == nil {
		return nil, fmt.Errorf("getIdentityFromConf error: nil idBytes")
	}

	// Decode the pem bytes
	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, fmt.Errorf("getIdentityFromBytes error: could not decode pem bytes [%v]", idBytes)
	}

	// get a cert
	var cert *x509.Certificate
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, fmt.Errorf("getIdentityFromBytes error: failed to parse x509 cert, err %s", err)
	}

	return cert, nil
}

type authorityKeyIdentifier struct {
	KeyIdentifier             []byte  `asn1:"optional,tag:0"`
	AuthorityCertIssuer       []byte  `asn1:"optional,tag:1"`
	AuthorityCertSerialNumber big.Int `asn1:"optional,tag:2"`
}

// getAuthorityKeyIdentifierFromCrl returns the Authority Key Identifier
// for the supplied CRL. The authority key identifier can be used to identify
// the public key corresponding to the private key which was used to sign the CRL.
func getAuthorityKeyIdentifierFromCrl(crl *pkix.CertificateList) ([]byte, error) {
	aki := authorityKeyIdentifier{}

	for _, ext := range crl.TBSCertList.Extensions {
		// Authority Key Identifier is identified by the following ASN.1 tag
		// authorityKeyIdentifier (2 5 29 35) (see https://tools.ietf.org/html/rfc3280.html)
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

// getSubjectKeyIdentifierFromCert returns the Subject Key Identifier for the supplied certificate
// Subject Key Identifier is an identifier of the public key of this certificate
func getSubjectKeyIdentifierFromCert(cert *x509.Certificate) ([]byte, error) {
	var SKI []byte

	for _, ext := range cert.Extensions {
		// Subject Key Identifier is identified by the following ASN.1 tag
		// subjectKeyIdentifier (2 5 29 14) (see https://tools.ietf.org/html/rfc3280.html)
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

// isCACert does a few checks on the certificate,
// assuming it's a CA; it returns true if all looks good
// and false otherwise
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

// Setup sets up the internal data structures
// for this MSP, given an MSPConfig ref; it
// returns nil in case of success or an error otherwise
func (validator *ecdsaValidator) Setup(conf *AuthConfig) error {
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

func (validator *ecdsaValidator) Validate(certByte []byte) error {
	authLogger.Debug("validating certificate")

	cert, err := validator.getCertFromPem(certByte)
	if err != nil {
		return fmt.Errorf("ParseCertificate failed %s", err)
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

// getCertificationChainForBCCSPIdentity returns the certification chain of the passed bccsp identity within this msp
func (validator *ecdsaValidator) getCertificationChain(cert *x509.Certificate) ([]*x509.Certificate, error) {
	// we expect to have a valid VerifyOptions instance
	if validator.opts == nil {
		return nil, errors.New("Invalid msp instance")
	}

	// CAs cannot be directly used as identities..
	if cert.IsCA {
		return nil, errors.New("A CA certificate cannot be used directly by this MSP")
	}

	return validator.getValidationChain(cert, false)
}

func (validator *ecdsaValidator) getUniqueValidationChain(cert *x509.Certificate, opts x509.VerifyOptions) ([]*x509.Certificate, error) {
	// ask golang to validate the cert for us based on the options that we've built at setup time
	if validator.opts == nil {
		return nil, fmt.Errorf("The supplied identity has no verify options")
	}
	validationChains, err := cert.Verify(opts)
	if err != nil {
		return nil, fmt.Errorf("The supplied identity is not valid, Verify() returned %s", err)
	}

	// we only support a single validation chain;
	// if there's more than one then there might
	// be unclarity about who owns the identity
	if len(validationChains) != 1 {
		return nil, fmt.Errorf("This MSP only supports a single validation chain, got %d", len(validationChains))
	}

	return validationChains[0], nil
}

func (validator *ecdsaValidator) getValidationChain(cert *x509.Certificate, isIntermediateChain bool) ([]*x509.Certificate, error) {
	validationChain, err := validator.getUniqueValidationChain(cert, validator.getValidityOptsForCert(cert))
	if err != nil {
		return nil, fmt.Errorf("Failed getting validation chain %s", err)
	}

	// we expect a chain of length at least 2
	if len(validationChain) < 2 {
		return nil, fmt.Errorf("Expected a chain of length at least 2, got %d", len(validationChain))
	}

	// check that the parent is a leaf of the certification tree
	// if validating an intermediate chain, the first certificate will the parent
	parentPosition := 1
	if isIntermediateChain {
		parentPosition = 0
	}
	if validator.certificationTreeInternalNodesMap[string(validationChain[parentPosition].Raw)] {
		return nil, fmt.Errorf("Invalid validation chain. Parent certificate should be a leaf of the certification tree [%v].", cert.Raw)
	}
	return validationChain, nil
}

// getCertificationChainIdentifier returns the certification chain identifier of the passed identity within this msp.
// The identifier is computes as the SHA256 of the concatenation of the certificates in the chain.
func (validator *ecdsaValidator) getCertificationChainIdentifier(cert *x509.Certificate) ([]byte, error) {
	chain, err := validator.getCertificationChain(cert)
	if err != nil {
		return nil, fmt.Errorf("Failed getting certification chain: [%s]", err)
	}

	hf := sha256.New()
	for i := 0; i < len(chain); i++ {
		hf.Write(chain[i].Raw)
	}
	return hf.Sum(nil), nil
}

func (validator *ecdsaValidator) setupCAs(conf *AuthConfig) error {
	// make and fill the set of CA certs - we expect them to be there
	if len(conf.RootCerts) == 0 {
		return errors.New("Expected at least one CA certificate")
	}

	// pre-create the verify options with roots and intermediates.
	// This is needed to make certificate sanitation working.
	// Recall that sanitization is applied also to root CA and intermediate
	// CA certificates. After their sanitization is done, the opts
	// will be recreated using the sanitized certs.
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

		validator.rootCerts[i] = cert
	}

	validator.intermediateCerts = make([]*x509.Certificate, len(conf.IntermediateCerts))
	for i, trustedCert := range conf.IntermediateCerts {
		cert, err := validator.getCertFromPem(trustedCert)
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

func (validator *ecdsaValidator) setupCRLs(conf *AuthConfig) error {
	// setup the CRL (if present)
	validator.CRL = make([]*pkix.CertificateList, len(conf.RevocationList))
	for i, crlbytes := range conf.RevocationList {
		crl, err := x509.ParseCRL(crlbytes)
		if err != nil {
			return fmt.Errorf("Could not parse RevocationList, err %s", err)
		}

		// TODO: pre-verify the signature on the CRL and create a map
		//       of CA certs to respective CRLs so that later upon
		//       validation we can already look up the CRL given the
		//       chain of the certificate to be validated

		validator.CRL[i] = crl
	}

	return nil
}

func (validator *ecdsaValidator) finalizeSetupCAs(config *AuthConfig) error {
	// ensure that our CAs are properly formed and that they are valid
	for _, cert := range append(append([]*x509.Certificate{}, validator.rootCerts...), validator.intermediateCerts...) {
		if !isCACert(cert) {
			return fmt.Errorf("CA Certificate did not have the Subject Key Identifier extension, (SN: %s)", cert.SerialNumber)
		}

		if err := validator.validateCAIdentity(cert); err != nil {
			return fmt.Errorf("CA Certificate is not valid, (SN: %s) [%s]", cert.SerialNumber, err)
		}
	}

	// populate certificationTreeInternalNodesMap to mark the internal nodes of the
	// certification tree
	validator.certificationTreeInternalNodesMap = make(map[string]bool)
	for _, cert := range append([]*x509.Certificate{}, validator.intermediateCerts...) {
		chain, err := validator.getUniqueValidationChain(cert, validator.getValidityOptsForCert(cert))
		if err != nil {
			return fmt.Errorf("Failed getting validation chain, (SN: %s)", cert.SerialNumber)
		}

		// Recall chain[0] is id.(*identity).id so it does not count as a parent
		for i := 1; i < len(chain); i++ {
			validator.certificationTreeInternalNodesMap[string(chain[i].Raw)] = true
		}
	}

	return nil
}

// sanitizeCert ensures that x509 certificates signed using ECDSA
// do have signatures in Low-S. If this is not the case, the certificate
// is regenerated to have a Low-S signature.
func (validator *ecdsaValidator) sanitizeCert(cert *x509.Certificate) (*x509.Certificate, error) {
	if isECDSASignedCert(cert) {
		// Lookup for a parent certificate to perform the sanitization
		var parentCert *x509.Certificate
		if cert.IsCA {
			// at this point, cert might be a root CA certificate
			// or an intermediate CA certificate
			chain, err := validator.getUniqueValidationChain(cert, validator.getValidityOptsForCert(cert))
			if err != nil {
				return nil, err
			}
			if len(chain) == 1 {
				// cert is a root CA certificate
				parentCert = cert
			} else {
				// cert is an intermediate CA certificate
				parentCert = chain[1]
			}
		} else {
			chain, err := validator.getUniqueValidationChain(cert, validator.getValidityOptsForCert(cert))
			if err != nil {
				return nil, err
			}
			parentCert = chain[1]
		}

		// Sanitize
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
		// validationChain[0] is the root CA certificate
		return nil
	}

	return validator.validateCertAgainstChain(cert, validationChain)
}

func (validator *ecdsaValidator) validateCertAgainstChain(cert *x509.Certificate, validationChain []*x509.Certificate) error {
	// here we know that the identity is valid; now we have to check whether it has been revoked

	// identify the SKI of the CA that signed this cert
	SKI, err := getSubjectKeyIdentifierFromCert(validationChain[1])
	if err != nil {
		return fmt.Errorf("Could not obtain Subject Key Identifier for signer cert, err %s", err)
	}

	// check whether one of the CRLs we have has this cert's
	// SKI as its AuthorityKeyIdentifier
	for _, crl := range validator.CRL {
		aki, err := getAuthorityKeyIdentifierFromCrl(crl)
		if err != nil {
			return fmt.Errorf("Could not obtain Authority Key Identifier for crl, err %s", err)
		}

		// check if the SKI of the cert that signed us matches the AKI of any of the CRLs
		if bytes.Equal(aki, SKI) {
			// we have a CRL, check whether the serial number is revoked
			for _, rc := range crl.TBSCertList.RevokedCertificates {
				if rc.SerialNumber.Cmp(cert.SerialNumber) == 0 {
					// We have found a CRL whose AKI matches the SKI of
					// the CA (root or intermediate) that signed the
					// certificate that is under validation. As a
					// precaution, we verify that said CA is also the
					// signer of this CRL.
					err = validationChain[1].CheckCRLSignature(crl)
					if err != nil {
						// the CA cert that signed the certificate
						// that is under validation did not sign the
						// candidate CRL - skip
						authLogger.Warn("Invalid signature over the identified CRL, error %s", err)
						continue
					}

					// A CRL also includes a time of revocation so that
					// the CA can say "this cert is to be revoked starting
					// from this time"; however here we just assume that
					// revocation applies instantaneously from the time
					// the MSP config is committed and used so we will not
					// make use of that field
					return errors.New("The certificate has been revoked")
				}
			}
		}
	}

	return nil
}

func (validator *ecdsaValidator) getValidityOptsForCert(cert *x509.Certificate) x509.VerifyOptions {
	// First copy the opts to override the CurrentTime field
	// in order to make the certificate passing the expiration test
	// independently from the real local current time.
	// This is a temporary workaround for FAB-3678

	var tempOpts x509.VerifyOptions
	tempOpts.Roots = validator.opts.Roots
	tempOpts.DNSName = validator.opts.DNSName
	tempOpts.Intermediates = validator.opts.Intermediates
	tempOpts.KeyUsages = validator.opts.KeyUsages
	tempOpts.CurrentTime = cert.NotBefore.Add(time.Second)

	return tempOpts
}
