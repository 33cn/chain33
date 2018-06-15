/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cryptosuite

import (
	"gitlab.33.cn/chain33/chain33/types"
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
