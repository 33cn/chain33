/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pkcs11

import (
	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	bccspPkcs11 "gitlab.33.cn/chain33/chain33/authority/bccsp/factory/pkcs11"
	"gitlab.33.cn/chain33/chain33/authority/bccsp/pkcs11"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite/bccsp/wrapper"
	"github.com/pkg/errors"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var logger = log.New("auth", "cryptosuite_p11")

//GetSuiteByConfig returns cryptosuite adaptor for bccsp loaded according to given config
func GetSuiteByConfig(config *types.Authority) (core.CryptoSuite, error) {
	// TODO: delete this check?
	if config.DefaultProvider != "PKCS11" {
		return nil, errors.Errorf("Unsupported BCCSP Provider: %s", config.SecurityProvider())
	}

	opts := getOptsByConfig(config)
	bccsp, err := getBCCSPFromOpts(opts)

	if err != nil {
		return nil, err
	}
	return &wrapper.CryptoSuite{BCCSP: bccsp}, nil
}

func getBCCSPFromOpts(config *pkcs11.PKCS11Opts) (bccsp.BCCSP, error) {
	f := &bccspPkcs11.PKCS11Factory{}

	csp, err := f.Get(config)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not initialize BCCSP %s", f.Name())
	}
	return csp, nil
}

//getOptsByConfig Returns Factory opts for given SDK config
func getOptsByConfig(c core.CryptoSuiteConfig) *pkcs11.PKCS11Opts {
	pkks := pkcs11.FileKeystoreOpts{KeyStorePath: c.KeyStorePath()}
	opts := &pkcs11.PKCS11Opts{
		SecLevel:     c.SecurityLevel(),
		HashFamily:   c.SecurityAlgorithm(),
		FileKeystore: &pkks,
		Library:      c.SecurityProviderLibPath(),
		Pin:          c.SecurityProviderPin(),
		Label:        c.SecurityProviderLabel(),
		SoftVerify:   c.SoftVerify(),
	}
	logger.Debug("Initialized PKCS11 cryptosuite")

	return opts
}
