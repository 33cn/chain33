/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitymgr

import (
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/authority/identitymgr/filestore"
)

var logger = log.New("module", "autority_identitymgr")
// IdentityManager implements fab/IdentityManager
type IdentityManager struct {
	orgName         string
	cryptoSuite     core.CryptoSuite
	mspPrivKeyStore core.KVStore
	mspCertStore    core.KVStore
	userStore       map[string]User
}

// NewIdentityManager creates a new instance of IdentityManager
func NewIdentityManager(orgName string, cryptoSuite core.CryptoSuite, CryptoPath string) (*IdentityManager, error) {
	var mspPrivKeyStore core.KVStore
	var mspCertStore core.KVStore
	var err error

	mspPrivKeyStore, err = filestore.NewFileKeyStore(CryptoPath)
	if err != nil {
		return nil, errors.Wrapf(err, "creating a private key store failed")
	}

	mspCertStore, err = filestore.NewFileCertStore(orgName, CryptoPath)
	if err != nil {
		return nil, errors.Wrapf(err, "creating a cert store failed")
	}
	userStore := make(map[string]User)

	mgr := &IdentityManager{
		orgName:         orgName,
		cryptoSuite:     cryptoSuite,
		mspPrivKeyStore: mspPrivKeyStore,
		mspCertStore:    mspCertStore,
		userStore:		 userStore,
	}
	return mgr, nil
}
