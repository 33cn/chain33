/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sw

import (
	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	bccspSw "gitlab.33.cn/chain33/chain33/authority/bccsp/factory"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite/bccsp/wrapper"
	"github.com/pkg/errors"
	log "github.com/inconshreveable/log15"
)

var logger = log.New("auth", "cryptosuite_sw")

//GetSuiteByConfig returns cryptosuite adaptor for bccsp loaded according to given config
func GetSuiteByConfig(config core.CryptoSuiteConfig) (core.CryptoSuite, error) {
	opts := getOptsByConfig(config)
	bccsp, err := getBCCSPFromOpts(opts)
	if err != nil {
		return nil, err
	}
	return wrapper.NewCryptoSuite(bccsp), nil
}

//GetSuiteWithDefaultEphemeral returns cryptosuite adaptor for bccsp with default ephemeral options (intended to aid testing)
func GetSuiteWithDefaultEphemeral() (core.CryptoSuite, error) {
	opts := getEphemeralOpts()

	bccsp, err := getBCCSPFromOpts(opts)
	if err != nil {
		return nil, err
	}
	return wrapper.NewCryptoSuite(bccsp), nil
}

func getBCCSPFromOpts(config *bccspSw.SwOpts) (bccsp.BCCSP, error) {
	f := &bccspSw.SWFactory{}

	csp, err := f.Get(&bccspSw.FactoryOpts{"",config})
	if err != nil {
		return nil, errors.Wrapf(err, "Could not initialize BCCSP %s", f.Name())
	}
	return csp, nil
}

//GetOptsByConfig Returns Factory opts for given SDK config
func getOptsByConfig(c core.CryptoSuiteConfig) *bccspSw.SwOpts {
	opts := &bccspSw.SwOpts{
		HashFamily: c.SecurityAlgorithm(),
		SecLevel:   c.SecurityLevel(),
		FileKeystore: &bccspSw.FileKeystoreOpts{
			KeyStorePath: c.KeyStorePath(),
		},
	}
	logger.Debug("Initialized cryptosuite")

	return opts
}

func getEphemeralOpts() *bccspSw.SwOpts {
	opts := &bccspSw.SwOpts{
		HashFamily: "SHA2",
		SecLevel:   256,
		Ephemeral:  false,
	}
	logger.Debug("Initialized ephemeral SW cryptosuite with default opts")

	return opts
}
