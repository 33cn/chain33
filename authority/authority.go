package authority

import (
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
    "gitlab.33.cn/chain33/chain33/authority/cryptosuite/bccsp/sw"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/signingmgr"
)

type Authority struct {
	cryptoPath string
	client     queue.Client
	cfg        *types.Authority
	cryptoSuite core.CryptoSuite
	signer   *signingmgr.SigningManager
}

func New(conf *types.Authority) *Authority{
	auth := &Authority{}
	auth.initConfig(conf)
	auth.cryptoPath = conf.CryptoPath
	auth.cfg = conf

	return auth
}

func (c *Authority)initConfig(conf *types.Authority) error {
	config := &cryptosuite.CryptoConfig{conf}

	cryptoSuite,err := sw.GetSuiteByConfig(config)
	if err != nil {
		return errors.WithMessage(err, "Failed to initialize crypto suite")
	}
	c.cryptoSuite = cryptoSuite


	signer,err := signingmgr.New(cryptoSuite)
	if err != nil {
		return errors.WithMessage(err, "Failed to initialize signing manage")
	}
	c.signer = signer



	return nil
}