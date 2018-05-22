/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cryptosuite

import (
	"path"

	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/authority/common/util/pathvar"
)


// Config represents the crypto suite configuration for the client
type CryptoConfig struct {
	Config *types.Authority
}

// SecurityAlgorithm returns cryptoSuite config hash algorithm
func (c *CryptoConfig) SecurityAlgorithm() string {
	return c.Config.HashAlgorithm
}

// SecurityLevel returns cryptSuite config security level
func (c *CryptoConfig) SecurityLevel() int {
	return int(c.Config.SecurityLevel)
}

//SoftVerify flag TODO
func (c *CryptoConfig) SoftVerify() bool {
	return true
}
// KeyStorePath returns the keystore path used by BCCSP
func (c *CryptoConfig) KeyStorePath() string {
	keystorePath := pathvar.Subst(c.Config.KeyStorePath)
	return path.Join(keystorePath, "keystore")
}
