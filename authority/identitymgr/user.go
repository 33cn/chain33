/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitymgr

import (
	"gitlab.33.cn/chain33/chain33/authority/bccsp"
)

// User is a representation of a Fabric user
type User struct {
	id                    string
	enrollmentCertificate []byte
	privateKey            bccsp.Key
}

// EnrollmentCertificate Returns the underlying ECert representing this user’s identity.
func (u *User) EnrollmentCertificate() []byte {
	return u.enrollmentCertificate
}

// PrivateKey returns the crypto suite representation of the private key
func (u *User) PrivateKey() bccsp.Key {
	return u.privateKey
}

// PublicVersion returns the public parts of this identity
func (u *User) PublicVersion() *User {
	return u
}
