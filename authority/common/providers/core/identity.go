/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"github.com/pkg/errors"
)

var (
	// ErrUserNotFound indicates the user was not found
	ErrUserNotFound = errors.New("user not found")
)

// IdentityManager provides management of identities in Fabric network
type IdentityManager interface {
	GetSigningIdentity(name string) (SigningIdentity, error)
}

// Identity represents a Fabric client identity
type Identity interface {

	// Verify a signature over some message using this identity as reference
	Verify(msg []byte, sig []byte) error

	// EnrollmentCertificate Returns the underlying ECert representing this user’s identity.
	EnrollmentCertificate() []byte
}

// SigningIdentity is an extension of Identity to cover signing capabilities.
type SigningIdentity interface {

	// Extends Identity
	Identity

	// Sign the message
	Sign(msg []byte) ([]byte, error)

	// GetPublicVersion returns the public parts of this identity
	PublicVersion() Identity

	// PrivateKey returns the crypto suite representation of the private key
	PrivateKey() Key
}


