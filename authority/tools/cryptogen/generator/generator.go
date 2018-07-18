package generator

import "crypto/x509"

type CAGenerator interface {
	SignCertificate(baseDir, name string, sans []string, pub interface{}) (*x509.Certificate, error)

	GenerateLocalUser(baseDir, name string) error
}
