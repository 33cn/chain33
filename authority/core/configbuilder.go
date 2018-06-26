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

package core

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"os"
	"gitlab.33.cn/chain33/chain33/authority/utils"
)


func getPemMaterialFromDir(dir string) ([][]byte, error) {
	authLogger.Debug(fmt.Sprintf("Reading directory %s", dir))

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
		authLogger.Debug(fmt.Sprintf("Inspecting file %s", fullName))

		item, err := utils.ReadPemFile(fullName)
		if err != nil {
			authLogger.Warn(fmt.Sprintf("Failed readgin file %s: %s", fullName, err))
			continue
		}

		content = append(content, item)
	}

	return content, nil
}

const (
	cacerts           = "cacerts"
	intermediatecerts = "intermediatecerts"
	crlsfolder        = "crls"
)

func GetAuthConfig(dir string) (*AuthConfig, error) {
	cacertDir := filepath.Join(dir, cacerts)
	intermediatecertsDir := filepath.Join(dir, intermediatecerts)
	crlsDir := filepath.Join(dir, crlsfolder)

	cacerts, err := getPemMaterialFromDir(cacertDir)
	if err != nil || len(cacerts) == 0 {
		return nil, fmt.Errorf("Could not load a valid ca certificate from directory %s, err %s", cacertDir, err)
	}

	intermediatecerts, err := getPemMaterialFromDir(intermediatecertsDir)
	if os.IsNotExist(err) {
		authLogger.Debug(fmt.Sprintf("Intermediate certs folder not found at [%s]. Skipping. [%s]", intermediatecertsDir, err))
	} else if err != nil {
		return nil, fmt.Errorf("Failed loading intermediate ca certs at [%s]: [%s]", intermediatecertsDir, err)
	}

	crls, err := getPemMaterialFromDir(crlsDir)
	if os.IsNotExist(err) {
		authLogger.Debug(fmt.Sprintf("crls folder not found at [%s]. Skipping. [%s]", crlsDir, err))
	} else if err != nil {
		return nil, fmt.Errorf("Failed loading crls at [%s]: [%s]", crlsDir, err)
	}

	// Compose FabricMSPConfig
	authconf := &AuthConfig{
		RootCerts:         cacerts,
		IntermediateCerts: intermediatecerts,
		RevocationList:    crls,
	}

	return authconf, nil
}
