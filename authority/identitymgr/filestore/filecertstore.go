/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package filestore

import (
	"fmt"
	"path"
	"strings"

	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/common/core"
)

// NewFileCertStore ...
func NewFileCertStore(orgname string, cryptoConfigMSPPath string) (core.KVStore, error) {
	opts := &FileKeyValueStoreOptions{
		Path: cryptoConfigMSPPath,
		KeySerializer: func(key interface{}) (string, error) {
			ck, ok := key.(string)
			if !ok {
				return "", errors.New("converting key to CertKey failed")
			}
			if ck == "" {
				return "", errors.New("invalid key")
			}

			// TODO: refactor to case insensitive or remove eventually.
			r := strings.NewReplacer("{userName}", ck, "{username}", ck)
			certDir := path.Join(r.Replace(cryptoConfigMSPPath), "signcerts")
			return path.Join(certDir, fmt.Sprintf("%s@%s-cert.pem", ck, orgname)), nil
		},
	}
	return New(opts)
}
