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

package mspmgr

import (
	"fmt"
	"io/ioutil"

	"encoding/pem"
	"path/filepath"

	"os"

	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	"gitlab.33.cn/chain33/chain33/authority/bccsp/factory"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
)

func readFile(file string) ([]byte, error) {
	fileCont, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Could not read file %s, err %s", file, err)
	}

	return fileCont, nil
}

func readPemFile(file string) ([]byte, error) {
	bytes, err := readFile(file)
	if err != nil {
		return nil, err
	}

	b, _ := pem.Decode(bytes)
	if b == nil { // TODO: also check that the type is what we expect (cert vs key..)
		return nil, fmt.Errorf("No pem content for file %s", file)
	}

	return bytes, nil
}

func getPemMaterialFromDir(dir string) ([][]byte, error) {
	mspLogger.Debug(fmt.Sprintf("Reading directory %s", dir))

	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, err
	}

	content := make([][]byte, 0)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("Could not read directory %s, err %s", err, dir)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		fullName := filepath.Join(dir, string(filepath.Separator), f.Name())
		mspLogger.Debug(fmt.Sprintf("Inspecting file %s", fullName))

		item, err := readPemFile(fullName)
		if err != nil {
			mspLogger.Warn(fmt.Sprintf("Failed readgin file %s: %s", fullName, err))
			continue
		}

		content = append(content, item)
	}

	return content, nil
}

const (
	cacerts              = "cacerts"
	admincerts           = "admincerts"
	intermediatecerts    = "intermediatecerts"
	crlsfolder           = "crls"
)

func SetupBCCSPKeystoreConfig(bccspConfig *factory.FactoryOpts, conf *cryptosuite.CryptoConfig) *factory.FactoryOpts {
	swOpts := &factory.SwOpts{}
	swOpts.HashFamily = conf.SecurityAlgorithm()
	swOpts.SecLevel = conf.SecurityLevel()
	bccspConfig.SwOpts = swOpts

	return bccspConfig
}

func GetLocalMspConfig(dir string, conf *cryptosuite.CryptoConfig) (*MSPConfig, error) {
	bccspConfig := &factory.FactoryOpts{}
	bccspConfig = SetupBCCSPKeystoreConfig(bccspConfig, conf)

	err := factory.InitFactories(bccspConfig)
	if err != nil {
		return nil, fmt.Errorf("Could not initialize BCCSP Factories [%s]", err)
	}

	return getMspConfig(dir, conf)
}

func getMspConfig(dir string, conf *cryptosuite.CryptoConfig) (*MSPConfig, error) {
	cacertDir := filepath.Join(dir, cacerts)
	admincertDir := filepath.Join(dir, admincerts)
	intermediatecertsDir := filepath.Join(dir, intermediatecerts)
	crlsDir := filepath.Join(dir, crlsfolder)

	cacerts, err := getPemMaterialFromDir(cacertDir)
	if err != nil || len(cacerts) == 0 {
		return nil, fmt.Errorf("Could not load a valid ca certificate from directory %s, err %s", cacertDir, err)
	}

	admincert, err := getPemMaterialFromDir(admincertDir)
	//if err != nil || len(admincert) == 0 {
	//	return nil, fmt.Errorf("Could not load a valid admin certificate from directory %s, err %s", admincertDir, err)
	//}

	intermediatecerts, err := getPemMaterialFromDir(intermediatecertsDir)
	if os.IsNotExist(err) {
		mspLogger.Debug(fmt.Sprintf("Intermediate certs folder not found at [%s]. Skipping. [%s]", intermediatecertsDir, err))
	} else if err != nil {
		return nil, fmt.Errorf("Failed loading intermediate ca certs at [%s]: [%s]", intermediatecertsDir, err)
	}

	crls, err := getPemMaterialFromDir(crlsDir)
	if os.IsNotExist(err) {
		mspLogger.Debug(fmt.Sprintf("crls folder not found at [%s]. Skipping. [%s]", crlsDir, err))
	} else if err != nil {
		return nil, fmt.Errorf("Failed loading crls at [%s]: [%s]", crlsDir, err)
	}

	// Set MspCryptoConfig
	var hashFunction string
	tmpValue := conf.SecurityLevel()
	switch tmpValue {
	case 256:
		hashFunction = bccsp.SHA256
	case 384:
		hashFunction = bccsp.SHA384
	default:
		mspLogger.Error(fmt.Sprintf("Invalid security level value:%d", tmpValue))
	}

	crypto := &MSPCryptoConfig{
		SignatureHashFamily:            conf.SecurityAlgorithm(),
		IdentityIdentifierHashFunction: hashFunction,
	}

	// Compose FabricMSPConfig
	mspconf := &MSPConfig{
		Admins:            admincert,
		RootCerts:         cacerts,
		IntermediateCerts: intermediatecerts,
		RevocationList:    crls,
		CryptoConfig:      crypto,
	}

	return mspconf, nil
}
