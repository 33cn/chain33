package msp

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"

	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/ca"
	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/csp"
)

func GenerateLocalMSP(baseDir, name string, signCA *ca.CA) error {
	err := createFolderStructure(baseDir, true)
	if err != nil {
		return err
	}

	keystore := filepath.Join(baseDir, "keystore")

	priv, _, err := csp.GeneratePrivateKey(keystore)
	if err != nil {
		return err
	}

	ecPubKey, err := csp.GetECPublicKey(priv)
	if err != nil {
		return err
	}
	cert, err := signCA.SignCertificate(filepath.Join(baseDir, "signcerts"),
		name, []string{}, ecPubKey, x509.KeyUsageDigitalSignature, []x509.ExtKeyUsage{})
	if err != nil || cert == nil {
		return err
	}

	err = x509Export(filepath.Join(baseDir, "cacerts", x509Filename(signCA.Name)), signCA.SignCert)
	return err
}

func createFolderStructure(rootDir string, local bool) error {
	var folders = []string{
		filepath.Join(rootDir, "cacerts"),
	}
	if local {
		folders = append(folders, filepath.Join(rootDir, "keystore"),
			filepath.Join(rootDir, "signcerts"))
	}

	for _, folder := range folders {
		err := os.MkdirAll(folder, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func x509Filename(name string) string {
	return name + "-cert.pem"
}

func x509Export(path string, cert *x509.Certificate) error {
	return pemExport(path, "CERTIFICATE", cert.Raw)
}

func pemExport(path, pemType string, bytes []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, &pem.Block{Type: pemType, Bytes: bytes})
}
