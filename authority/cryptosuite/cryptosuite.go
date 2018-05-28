/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cryptosuite

import (
	"sync/atomic"

	"errors"

	"sync"

	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite/bccsp/sw"
	log "github.com/inconshreveable/log15"
)

var logger = log.New("auth", "cryptosuite")

var initOnce sync.Once
var defaultCryptoSuite core.CryptoSuite
var initialized int32

func initSuite(defaultSuite core.CryptoSuite) error {
	if defaultSuite == nil {
		return errors.New("attempting to set invalid default suite")
	}
	initOnce.Do(func() {
		defaultCryptoSuite = defaultSuite
		atomic.StoreInt32(&initialized, 1)
	})
	return nil
}

//GetDefault returns default core
func GetDefault() core.CryptoSuite {
	if atomic.LoadInt32(&initialized) > 0 {
		return defaultCryptoSuite
	}
	//Set default suite
	logger.Info("No default cryptosuite found, using default SW implementation")

	// Use SW as the default cryptosuite when not initialized properly - should be for testing only
	s, err := sw.GetSuiteWithDefaultEphemeral()
	if err != nil {
		//TODO logger.Panicf->logger.Crit
		logger.Crit("Could not initialize default cryptosuite: %v", err)
	}
	err = initSuite(s)
	if err != nil {
		//TODO logger.Panicf->logger.Crit
		logger.Crit("Could not set default cryptosuite: %v", err)
	}

	return defaultCryptoSuite
}

//SetDefault sets default suite if one is not already set or created
//Make sure you set default suite before very first call to GetDefault(),
//otherwise this function will return an error
func SetDefault(newDefaultSuite core.CryptoSuite) error {
	if atomic.LoadInt32(&initialized) > 0 {
		return errors.New("default crypto suite is already set")
	}
	return initSuite(newDefaultSuite)
}

//GetSHAOpts returns options for computing SHA.
func GetSHAOpts() core.HashOpts {
	return &bccsp.SHAOpts{}
}
