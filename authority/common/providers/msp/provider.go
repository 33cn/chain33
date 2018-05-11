/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
)

// Context is the context required by MSP services
type Context interface {
	core.Providers
	Providers
}

// IdentityManagerProvider provides identity management services
type IdentityManagerProvider interface {
	IdentityManager(orgName string) (IdentityManager, bool)
}

//IdentityConfig contains identity configurations
type IdentityConfig interface {
	Client() (*ClientConfig, error)
	CAConfig(org string) (*CAConfig, error)
	CAServerCerts(org string) ([][]byte, error)
	CAClientKey(org string) ([]byte, error)
	CAClientCert(org string) ([]byte, error)
	CAKeyStorePath() string
	CredentialStorePath() string
}

// ClientConfig provides the definition of the client configuration
type ClientConfig struct {
	Organization    string
	CryptoConfig    CCType
	CredentialStore CredentialStoreType
}

// CCType defines the path to crypto keys and certs
type CCType struct {
	Path string
}

// CredentialStoreType defines pluggable KV store properties
type CredentialStoreType struct {
	Path        string
	CryptoStore struct {
		Path string
	}
}

// EnrollCredentials holds credentials used for enrollment
type EnrollCredentials struct {
	EnrollID     string
	EnrollSecret string
}

// CAConfig defines a CA configuration
type CAConfig struct {
	URL        string
	Registrar  EnrollCredentials
	CAName     string
}

// Providers represents a provider of MSP service.
type Providers interface {
	UserStore() UserStore
	IdentityManagerProvider
	IdentityConfig() IdentityConfig
}
