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
	config *types.Authority
}

// SecurityAlgorithm returns cryptoSuite config hash algorithm
func (c *CryptoConfig) SecurityAlgorithm() string {
	return c.config.HashAlgorithm
}

// SecurityLevel returns cryptSuite config security level
func (c *CryptoConfig) SecurityLevel() int {
	return int(c.config.SecurityLevel)
}

//SecurityProvider provider SW or PKCS11
func (c *CryptoConfig) SecurityProvider() string {
	return c.config.DefaultProvider
}

// KeyStorePath returns the keystore path used by BCCSP
func (c *CryptoConfig) KeyStorePath() string {
	keystorePath := pathvar.Subst(c.config.KeyStorePath)
	return path.Join(keystorePath, "keystore")
}
