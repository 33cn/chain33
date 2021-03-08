// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

type noneValidator struct {
}

// NewNoneValidator 创建none校验器
func NewNoneValidator() (Validator, error) {
	return &noneValidator{}, nil
}

func (validator *noneValidator) Setup(conf *AuthConfig) error {
	return nil
}

func (validator *noneValidator) Validate(certByte []byte, pubKey []byte) error {
	return nil
}

func (validator *noneValidator) GetCertFromSignature(signature []byte) ([]byte, error) {
	return []byte(""), nil
}
