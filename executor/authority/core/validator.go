// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

// Validator 证书校验器
type Validator interface {
	Setup(config *AuthConfig) error

	Validate(cert []byte, pubKey []byte) error

	GetCertFromSignature(signature []byte) ([]byte, error)
}

// AuthConfig 校验器配置
type AuthConfig struct {
	RootCerts         [][]byte
	IntermediateCerts [][]byte
	RevocationList    [][]byte
}
