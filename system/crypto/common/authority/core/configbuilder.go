// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/crypto/common/authority/utils"

	"os"
)

var authLogger = log.New("module", "crypto")

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

// GetAuthConfig 获取证书文件配置
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

	authconf := &AuthConfig{
		RootCerts:         cacerts,
		IntermediateCerts: intermediatecerts,
		RevocationList:    crls,
	}

	return authconf, nil
}
