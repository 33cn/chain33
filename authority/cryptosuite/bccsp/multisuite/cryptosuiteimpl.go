/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisuite

import (
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite/bccsp/pkcs11"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite/bccsp/sw"
	"github.com/pkg/errors"
)

//GetSuiteByConfig returns cryptosuite adaptor for bccsp loaded according to given config
func GetSuiteByConfig(config core.CryptoSuiteConfig) (core.CryptoSuite, error) {
	switch config.SecurityProvider() {
	case "SW":
		return sw.GetSuiteByConfig(config)
	case "PKCS11":
		return pkcs11.GetSuiteByConfig(config)
	}

	return nil, errors.Errorf("Unsupported security provider requested: %s", config.SecurityProvider())
}
