package factory

import (
	"fmt"
	"errors"
	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/factory/csp"
)

type FactoryOpts struct {
	KeyStorePath string
}

func GetCSPFromOpts(config *FactoryOpts) (csp.CSP, error) {
	if config == nil || config.KeyStorePath == "" {
		return nil, errors.New("Invalid config. It must not be nil.")
	}

	fks, err := csp.NewFileBasedKeyStore(nil, config.KeyStorePath, false)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize software key store: %s", err)
	}

	return csp.New(fks)
}