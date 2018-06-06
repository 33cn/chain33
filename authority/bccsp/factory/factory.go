/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package factory

import (
	"fmt"
	"sync"

	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	log "github.com/inconshreveable/log15"
)

var (
	// Default BCCSP
	defaultBCCSP bccsp.BCCSP

	// when InitFactories has not been called yet (should only happen
	// in test cases), use this BCCSP temporarily
	bootBCCSP bccsp.BCCSP

	// factories' Sync on Initialization
	factoriesInitOnce sync.Once
	bootBCCSPInitOnce sync.Once

	// Factories' Initialization Error
	factoriesInitError error

	logger = log.New("auth", "bccsp")
)

// BCCSPFactory is used to get instances of the BCCSP interface.
// A Factory has name used to address it.
type BCCSPFactory interface {

	// Get returns an instance of BCCSP using opts.
	Get(opts *FactoryOpts) (bccsp.BCCSP, error)
}

// GetDefault returns a non-ephemeral (long-term) BCCSP
func GetDefault() bccsp.BCCSP {
	if defaultBCCSP == nil {
		logger.Warn("Before using BCCSP, please call InitFactories(). Falling back to bootBCCSP.")
		bootBCCSPInitOnce.Do(func() {
			var err error
			f := &SWFactory{}
			bootBCCSP, err = f.Get(GetDefaultOpts())
			if err != nil {
				panic("BCCSP Internal error, failed initialization with GetDefaultOpts!")
			}
		})
		return bootBCCSP
	}
	return defaultBCCSP
}

func initBCCSP(f BCCSPFactory, config *FactoryOpts) error {
	csp, err := f.Get(config)
	if err != nil {
		return fmt.Errorf("Could not initialize", err)
	}

	logger.Debug("Initialize BCCSP")
	defaultBCCSP = csp
	return nil
}

// InitFactories must be called before using factory interfaces
// It is acceptable to call with config = nil, in which case
// some defaults will get used
// Error is returned only if defaultBCCSP cannot be found
func InitFactories(config *FactoryOpts) error {
	factoriesInitOnce.Do(func() {
		setFactories(config)
	})

	return factoriesInitError
}

func setFactories(config *FactoryOpts) error {
	// Take some precautions on default opts
	if config == nil {
		config = GetDefaultOpts()
	}

	f := &SWFactory{}
	err := initBCCSP(f, config)
	if err != nil {
		factoriesInitError = fmt.Errorf("Failed initializing BCCSP", err)
	}

	return factoriesInitError
}

// GetBCCSPFromOpts returns a BCCSP created according to the options passed in input.
func GetBCCSPFromOpts(config *FactoryOpts) (bccsp.BCCSP, error) {
	f := &SWFactory{}
	csp, err := f.Get(config)
	if err != nil {
		return nil, fmt.Errorf("Could not initialize BCCSP", err)
	}
	return csp, nil
}
