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
/*
Notice: This file has been modified for Hyperledger Fabric SDK Go usage.
Please review third_party pinning scripts and patches for more details.
*/

package cryptoutils

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"

	"github.com/pkg/errors"

	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	log "github.com/inconshreveable/log15"
)

var alog = log.New("module", "autority_cryptoutils")

// ImportBCCSPKeyFromPEMBytes attempts to create a private BCCSP key from a pem byte slice
func ImportBCCSPKeyFromPEMBytes(keyBuff []byte, myCSP core.CryptoSuite, temporary bool) (core.Key, error) {
	keyFile := "pem bytes"

	key, err := PEMtoPrivateKey(keyBuff, nil)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("Failed parsing private key from %s", keyFile))
	}
	switch key.(type) {
	case *ecdsa.PrivateKey:
		priv, err := PrivateKeyToDER(key.(*ecdsa.PrivateKey))
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("Failed to convert ECDSA private key for '%s'", keyFile))
		}
		sk, err := myCSP.KeyImport(priv, GetECDSAPrivateKeyImportOpts(temporary))
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("Failed to import ECDSA private key for '%s'", keyFile))
		}
		return sk, nil
	case *rsa.PrivateKey:
		return nil, errors.Errorf("Failed to import RSA key from %s; RSA private key import is not supported", keyFile)
	default:
		return nil, errors.Errorf("Failed to import key from %s: invalid secret key type", keyFile)
	}
}
