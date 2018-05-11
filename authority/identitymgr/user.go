/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitymgr

import (
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/msp"
	"github.com/pkg/errors"
)

// User is a representation of a Fabric user
type User struct {
	id                    string
	enrollmentCertificate []byte
	privateKey            core.Key
}

// Identifier returns user identifier
func (u *User) Identifier() *msp.IdentityIdentifier {
	return &msp.IdentityIdentifier{ID: u.id}
}

// Verify a signature over some message using this identity as reference
func (u *User) Verify(msg []byte, sig []byte) error {
	return errors.New("not implemented")
}

// EnrollmentCertificate Returns the underlying ECert representing this user’s identity.
func (u *User) EnrollmentCertificate() []byte {
	return u.enrollmentCertificate
}

// PrivateKey returns the crypto suite representation of the private key
func (u *User) PrivateKey() core.Key {
	return u.privateKey
}

// PublicVersion returns the public parts of this identity
func (u *User) PublicVersion() msp.Identity {
	return u
}

// Sign the message
func (u *User) Sign(msg []byte) ([]byte, error) {
	return nil, errors.New("not implemented")
}
