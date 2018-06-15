/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package filestore

import (
	"encoding/hex"
	"path"
	"strings"

	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/common/core"
)

// NewFileKeyStore ...
func NewFileKeyStore(cryptoConfigMSPPath string) (core.KVStore, error) {
	opts := &FileKeyValueStoreOptions{
		Path: cryptoConfigMSPPath,
		KeySerializer: func(key interface{}) (string, error) {
			pkk, ok := key.(*core.PrivKeyKey)
			if !ok {
				return "", errors.New("converting key to PrivKeyKey failed")
			}
			if pkk == nil || pkk.ID == "" || pkk.SKI == nil {
				return "", errors.New("invalid key")
			}

			// TODO: refactor to case insensitive or remove eventually.
			r := strings.NewReplacer("{userName}", pkk.ID, "{username}", pkk.ID)
			keyDir := path.Join(r.Replace(cryptoConfigMSPPath), "keystore")

			return path.Join(keyDir, hex.EncodeToString(pkk.SKI)+"_sk"), nil
		},
	}
	return New(opts)
}
