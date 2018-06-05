/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package signingmgr

import (
	"gitlab.33.cn/chain33/chain33/authority/common/core"
	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
	"github.com/pkg/errors"
	"fmt"
	"encoding/hex"
	log "github.com/inconshreveable/log15"
)

var signLogger = log.New("module", "signingmgr")
// SigningManager is used for signing objects with private key
type SigningManager struct {
	cryptoProvider core.CryptoSuite
	hashOpts       bccsp.HashOpts
	signerOpts     bccsp.SignerOpts
}

// New Constructor for a signing manager.
// @param {BCCSP} cryptoProvider - crypto provider
// @param {Config} config - configuration provider
// @returns {SigningManager} new signing manager
func New(cryptoProvider core.CryptoSuite) (*SigningManager, error) {
	return &SigningManager{cryptoProvider: cryptoProvider, hashOpts: cryptosuite.GetSHAOpts()}, nil
}

// Sign will sign the given object using provided key
func (mgr *SigningManager) Sign(object []byte, key bccsp.Key) ([]byte, error) {
	if len(object) == 0 {
		return nil, errors.New("object (to sign) required")
	}

	if key == nil {
		return nil, errors.New("key (for signing) required")
	}

	digest, err := mgr.cryptoProvider.Hash(object, mgr.hashOpts)
	if err != nil {
		return nil, err
	}
	signLogger.Debug(fmt.Sprintf("Verify: digest = %s", hex.Dump(digest)))
	signature, err := mgr.cryptoProvider.Sign(key, digest, mgr.signerOpts)
	if err != nil {
		return nil, err
	}
	return signature, nil
}
