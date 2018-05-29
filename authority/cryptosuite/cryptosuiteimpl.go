/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cryptosuite

import (
	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	"gitlab.33.cn/chain33/chain33/authority/bccsp/factory"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	"github.com/pkg/errors"
	log "github.com/inconshreveable/log15"
)

var logger = log.New("auth", "cryptosuite")

//GetSuiteByConfig returns cryptosuite adaptor for bccsp loaded according to given config
func GetSuiteByConfig(config core.CryptoSuiteConfig) (core.CryptoSuite, error) {
	opts := getOptsByConfig(config)
	bccsp, err := getBCCSPFromOpts(opts)
	if err != nil {
		return nil, err
	}
	return NewCryptoSuite(bccsp), nil
}

func getBCCSPFromOpts(config *factory.SwOpts) (bccsp.BCCSP, error) {
	f := &factory.SWFactory{}

	csp, err := f.Get(&factory.FactoryOpts{"",config})
	if err != nil {
		return nil, errors.Wrapf(err, "Could not initialize BCCSP %s", f.Name())
	}
	return csp, nil
}

//GetOptsByConfig Returns Factory opts for given SDK config
func getOptsByConfig(c core.CryptoSuiteConfig) *factory.SwOpts {
	opts := &factory.SwOpts{
		HashFamily: c.SecurityAlgorithm(),
		SecLevel:   c.SecurityLevel(),
		FileKeystore: &factory.FileKeystoreOpts{
			KeyStorePath: c.KeyStorePath(),
		},
	}
	logger.Debug("Initialized cryptosuite")

	return opts
}

//GetSHAOpts returns options for computing SHA.
func GetSHAOpts() bccsp.HashOpts {
	return &bccsp.SHAOpts{}
}

//NewCryptoSuite returns cryptosuite adaptor for given bccsp.BCCSP implementation
func NewCryptoSuite(bccsp bccsp.BCCSP) core.CryptoSuite {
	return &CryptoSuite{
		BCCSP: bccsp,
	}
}

//GetKey returns implementation of of cryptosuite.Key
func GetKey(newkey bccsp.Key) bccsp.Key {
	return &key{newkey}
}