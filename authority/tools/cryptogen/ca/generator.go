package ca

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"

	"path/filepath"

	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/csp"
)

type CA struct {
	Name string
	Signer   crypto.Signer
	SignCert *x509.Certificate
}

func NewCA(baseDir, name string) (*CA, error) {
	var ca *CA

	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		return nil, err
	}

	priv, signer, err := csp.GeneratePrivateKey(baseDir)
	if err != nil {
		return nil, err
	}

	// get public signing certificate
	ecPubKey, err := csp.GetECPublicKey(priv)
	if err != nil {
		return nil, err
	}

	template := x509Template()
	//this is a CA
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageDigitalSignature |
		x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign |
		x509.KeyUsageCRLSign
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageAny}

	//set the organization for the subject
	subject := subjectTemplate()
	subject.CommonName = name

	template.Subject = subject
	template.SubjectKeyId = priv.SKI()

	x509Cert, err := genCertificateECDSA(baseDir, name, &template, &template,
		ecPubKey, signer)
	if err != nil {
		return nil, err
	}
	ca = &CA{
		Name:     name,
		Signer:   signer,
		SignCert: x509Cert,
	}

	return ca, nil
}

func (ca *CA) SignCertificate(baseDir, name string, sans []string, pub *ecdsa.PublicKey,
	ku x509.KeyUsage, eku []x509.ExtKeyUsage) (*x509.Certificate, error) {

	template := x509Template()
	template.KeyUsage = ku
	template.ExtKeyUsage = eku

	//set the organization for the subject
	subject := subjectTemplate()
	subject.CommonName = name

	template.Subject = subject
	template.DNSNames = sans

	cert, err := genCertificateECDSA(baseDir, name, &template, ca.SignCert,
		pub, ca.Signer)

	if err != nil {
		return nil, err
	}

	return cert, nil
}

// default template for X509 subject
func subjectTemplate() pkix.Name {
	return pkix.Name{
		Country:  []string{"US"},
		Locality: []string{"San Francisco"},
		Province: []string{"California"},
	}
}

// default template for X509 certificates
func x509Template() x509.Certificate {

	// generate a serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	// set expiry to around 10 years
	expiry := 3650 * 24 * time.Hour
	// backdate 5 min
	notBefore := time.Now().Add(-5 * time.Minute).UTC()

	//basic template to use
	x509 := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(expiry).UTC(),
		BasicConstraintsValid: true,
	}
	return x509

}

// generate a signed X509 certficate using ECDSA
func genCertificateECDSA(baseDir, name string, template, parent *x509.Certificate, pub *ecdsa.PublicKey,
	priv interface{}) (*x509.Certificate, error) {

	//create the x509 public cert
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent, pub, priv)
	if err != nil {
		return nil, err
	}

	//write cert out to file
	fileName := filepath.Join(baseDir, name+"-cert.pem")
	certFile, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}
	//pem encode the cert
	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certFile.Close()
	if err != nil {
		return nil, err
	}

	x509Cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, err
	}
	return x509Cert, nil
}
