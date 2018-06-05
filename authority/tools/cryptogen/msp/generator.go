/*
Copyright IBM Corp. 2017 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package msp

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"

	"encoding/hex"

	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	"gitlab.33.cn/chain33/chain33/authority/bccsp/factory"
	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/ca"
	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/csp"
)

func GenerateLocalMSP(baseDir, name string, signCA *ca.CA) error {

	// create folder structure

	err := createFolderStructure(baseDir, true)
	if err != nil {
		return err
	}

	/*
		Create the MSP identity artifacts
	*/
	// get keystore path
	keystore := filepath.Join(baseDir, "keystore")

	// generate private key
	priv, _, err := csp.GeneratePrivateKey(keystore)
	if err != nil {
		return err
	}

	// get public key
	ecPubKey, err := csp.GetECPublicKey(priv)
	if err != nil {
		return err
	}
	// generate X509 certificate using signing CA
	cert, err := signCA.SignCertificate(filepath.Join(baseDir, "signcerts"),
		name, []string{}, ecPubKey, x509.KeyUsageDigitalSignature, []x509.ExtKeyUsage{})
	if err != nil || cert == nil {
		return err
	}

	// write artifacts to MSP folders

	// the signing CA certificate goes into cacerts
	err = x509Export(filepath.Join(baseDir, "cacerts", x509Filename(signCA.Name)), signCA.SignCert)
	if err != nil {
		return err
	}

	return nil
}

func GenerateVerifyingMSP(baseDir string, signCA *ca.CA) error {

	// create folder structure and write artifacts to proper locations
	err := createFolderStructure(baseDir, false)
	if err == nil {
		// the signing CA certificate goes into cacerts
		err = x509Export(filepath.Join(baseDir, "cacerts", x509Filename(signCA.Name)), signCA.SignCert)
		if err != nil {
			return err
		}
	}

	// create a throwaway cert to act as an admin cert
	// NOTE: the admincerts folder is going to be
	// cleared up anyway by copyAdminCert, but
	// we leave a valid admin for now for the sake
	// of unit tests
	factory.InitFactories(nil)
	bcsp := factory.GetDefault()
	priv, err := bcsp.KeyGen(&bccsp.ECDSAP256KeyGenOpts{})
	ecPubKey, err := csp.GetECPublicKey(priv)
	if err != nil {
		return err
	}
	_, err = signCA.SignCertificate(filepath.Join(baseDir, "admincerts"), signCA.Name,
		[]string{""}, ecPubKey, x509.KeyUsageDigitalSignature, []x509.ExtKeyUsage{})
	if err != nil {
		return err
	}

	return nil
}

func createFolderStructure(rootDir string, local bool) error {

	var folders []string
	// create admincerts, cacerts, keystore and signcerts folders
	folders = []string{
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

func keyExport(keystore, output string, key bccsp.Key) error {
	id := hex.EncodeToString(key.SKI())

	return os.Rename(filepath.Join(keystore, id+"_sk"), output)
}

func pemExport(path, pemType string, bytes []byte) error {
	//write pem out to file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, &pem.Block{Type: pemType, Bytes: bytes})
}
