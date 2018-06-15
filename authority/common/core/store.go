/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

// UserData is the representation of User in UserStore
// PrivateKey is stored separately, in the crypto store
type UserData struct {
	ID                    string
	EnrollmentCertificate []byte
}

// PrivKeyKey is a composite key for accessing a private key in the key store
type PrivKeyKey struct {
	ID  string
	SKI []byte
}
