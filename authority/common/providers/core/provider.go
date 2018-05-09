/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

//CryptoSuiteConfig contains sdk configuration items for cryptosuite.
type CryptoSuiteConfig interface {
	SecurityAlgorithm() string
	SecurityLevel() int
	SecurityProvider() string
	KeyStorePath() string
}

// Providers represents the SDK configured core providers context.
type Providers interface {
	CryptoSuite() CryptoSuite
	SigningManager() SigningManager
}

//ConfigProvider provides config backend for SDK
type ConfigProvider func() (ConfigBackend, error)

//ConfigBackend backend for all config types in SDK
type ConfigBackend interface {
	Lookup(key string) (interface{}, bool)
}
