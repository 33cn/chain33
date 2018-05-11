/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitymgr

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/msp"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/common/util/cryptoutils"
)

func newUser(userData *msp.UserData, cryptoSuite core.CryptoSuite) (*User, error) {
	pubKey, err := cryptoutils.GetPublicKeyFromCert(userData.EnrollmentCertificate, cryptoSuite)
	if err != nil {
		return nil, errors.WithMessage(err, "fetching public key from cert failed")
	}
	pk, err := cryptoSuite.GetKey(pubKey.SKI())
	if err != nil {
		return nil, errors.WithMessage(err, "cryptoSuite GetKey failed")
	}
	u := &User{
		id:    userData.ID,
		enrollmentCertificate: userData.EnrollmentCertificate,
		privateKey:            pk,
	}
	return u, nil
}

// NewUser creates a User instance
func (mgr *IdentityManager) NewUser(userData *msp.UserData) (*User, error) {
	return newUser(userData, mgr.cryptoSuite)
}

func (mgr *IdentityManager) loadUserFromStore(username string) (*User, error) {
	//TODO
	return nil,nil
}

// GetSigningIdentity returns a signing identity for the given id
func (mgr *IdentityManager) GetSigningIdentity(id string) (msp.SigningIdentity, error) {
	user, err := mgr.GetUser(id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUser returns a user for the given user name
func (mgr *IdentityManager) GetUser(username string) (*User, error) { //nolint

	u, err := mgr.loadUserFromStore(username)
	if err != nil {
		if err != msp.ErrUserNotFound {
			return nil, errors.WithMessage(err, "loading user from store failed")
		}
		// Not found, continue
	}

	if u == nil {
		certBytes, err := mgr.getCertBytesFromCertStore(username)
		if err != nil && err != msp.ErrUserNotFound {
			return nil, errors.WithMessage(err, "fetching cert from store failed")
		}
		if certBytes == nil {
			return nil, msp.ErrUserNotFound
		}
		privateKey, err := mgr.getPrivateKeyFromCert(username, certBytes)
		if err != nil {
			return nil, errors.WithMessage(err, "getting private key from cert failed")
		}
		if privateKey == nil {
			return nil, fmt.Errorf("unable to find private key for user [%s]", username)
		}
		u = &User{
			id:    username,
			enrollmentCertificate: certBytes,
			privateKey:            privateKey,
		}
	}
	return u, nil
}

func (mgr *IdentityManager) getPrivateKeyPemFromKeyStore(username string, ski []byte) ([]byte, error) {
	if mgr.mspPrivKeyStore == nil {
		return nil, nil
	}
	key, err := mgr.mspPrivKeyStore.Load(
		&msp.PrivKeyKey{
			ID:    username,
			SKI:   ski,
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
		return nil, msp.ErrUserNotFound
	}
	cert, err := mgr.mspCertStore.Load(&msp.IdentityIdentifier{
		ID:    username,
	})
	if err != nil {
		if err == core.ErrKeyValueNotFound {
			return nil, msp.ErrUserNotFound
		}
		return nil, err
	}
	certBytes, ok := cert.([]byte)
	if !ok {
		return nil, errors.New("cert from store is not []byte")
	}
	return certBytes, nil
}

func (mgr *IdentityManager) getPrivateKeyFromCert(username string, cert []byte) (core.Key, error) {
	if cert == nil {
		return nil, errors.New("cert is nil")
	}
	pubKey, err := cryptoutils.GetPublicKeyFromCert(cert, mgr.cryptoSuite)
	if err != nil {
		return nil, errors.WithMessage(err, "fetching public key from cert failed")
	}
	privKey, err := mgr.getPrivateKeyFromKeyStore(username, pubKey.SKI())
	if err == nil {
		return privKey, nil
	}
	if err != core.ErrKeyValueNotFound {
		return nil, errors.WithMessage(err, "fetching private key from key store failed")
	}
	return mgr.cryptoSuite.GetKey(pubKey.SKI())
}

func (mgr *IdentityManager) getPrivateKeyFromKeyStore(username string, ski []byte) (core.Key, error) {
	pemBytes, err := mgr.getPrivateKeyPemFromKeyStore(username, ski)
	if err != nil {
		return nil, err
	}
	if pemBytes != nil {
		return cryptoutils.ImportBCCSPKeyFromPEMBytes(pemBytes, mgr.cryptoSuite, true)
	}
	return nil, core.ErrKeyValueNotFound
}
