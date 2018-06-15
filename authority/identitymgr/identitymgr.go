/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitymgr

import (
	"fmt"

	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	"gitlab.33.cn/chain33/chain33/authority/common/core"
	"gitlab.33.cn/chain33/chain33/authority/identitymgr/filestore"
)

var (
	// ErrUserNotFound indicates the user was not found
	ErrUserNotFound = errors.New("user not found")
)

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
		userStore:       userStore,
	}
	return mgr, nil
}

func (mgr *IdentityManager) loadUserFromStore(username string) *User {
	user, ok := mgr.userStore[username]
	if ok {
		return &user
	} else {
		return nil
	}
}

func (mgr *IdentityManager) storeUserFromStore(username string, user *User) {
	tmpUser := &User{}
	tmpUser.id = user.id
	tmpUser.enrollmentCertificate = append(tmpUser.enrollmentCertificate, user.EnrollmentCertificate()...)
	tmpUser.privateKey = user.PrivateKey()
	mgr.userStore[username] = *tmpUser
}

// GetSigningIdentity returns a signing identity for the given id
func (mgr *IdentityManager) GetSigningIdentity(id string) (*User, error) {
	user, err := mgr.GetUser(id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUser returns a user for the given user name
func (mgr *IdentityManager) GetUser(username string) (*User, error) { //nolint
	u := mgr.loadUserFromStore(username)
	if u == nil {
		certBytes, err := mgr.getCertBytesFromCertStore(username)
		if err != nil && err != ErrUserNotFound {
			return nil, errors.WithMessage(err, "fetching cert from store failed")
		}
		if certBytes == nil {
			return nil, ErrUserNotFound
		}
		privateKey, err := mgr.getPrivateKeyFromCert(username, certBytes)
		if err != nil {
			return nil, errors.WithMessage(err, "getting private key from cert failed")
		}
		if privateKey == nil {
			return nil, fmt.Errorf("unable to find private key for user [%s]", username)
		}
		u = &User{
			id: username,
			enrollmentCertificate: certBytes,
			privateKey:            privateKey,
		}
		mgr.storeUserFromStore(username, u)
	}
	return u, nil
}

func (mgr *IdentityManager) getPrivateKeyPemFromKeyStore(username string, ski []byte) ([]byte, error) {
	if mgr.mspPrivKeyStore == nil {
		return nil, nil
	}
	key, err := mgr.mspPrivKeyStore.Load(
		&core.PrivKeyKey{
			ID:  username,
			SKI: ski,
		})
	if err != nil {
		return nil, err
	}
	keyBytes, ok := key.([]byte)
	if !ok {
		return nil, errors.New("key from store is not []byte")
	}
	return keyBytes, nil
}

func (mgr *IdentityManager) getCertBytesFromCertStore(username string) ([]byte, error) {
	if mgr.mspCertStore == nil {
		return nil, ErrUserNotFound
	}
	cert, err := mgr.mspCertStore.Load(username)
	if err != nil {
		if err == core.ErrKeyValueNotFound {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	certBytes, ok := cert.([]byte)
	if !ok {
		return nil, errors.New("cert from store is not []byte")
	}
	return certBytes, nil
}

func (mgr *IdentityManager) getPrivateKeyFromCert(username string, cert []byte) (bccsp.Key, error) {
	if cert == nil {
		return nil, errors.New("cert is nil")
	}
	pubKey, err := GetPublicKeyFromCert(cert, mgr.cryptoSuite)
	if err != nil {
		return nil, errors.WithMessage(err, "fetching public key from cert failed")
	}
	privKey, err := mgr.getPrivateKeyFromKeyStore(username, pubKey.SKI())
	if err == nil {
		return privKey, nil
	}

	return nil, errors.WithMessage(err, "fetching private key from key store failed")
}

func (mgr *IdentityManager) getPrivateKeyFromKeyStore(username string, ski []byte) (bccsp.Key, error) {
	pemBytes, err := mgr.getPrivateKeyPemFromKeyStore(username, ski)
	if err != nil {
		return nil, err
	}
	if pemBytes != nil {
		return ImportBCCSPKeyFromPEMBytes(pemBytes, mgr.cryptoSuite)
	}
	return nil, core.ErrKeyValueNotFound
}
