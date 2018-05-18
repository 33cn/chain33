package authority

import (
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
    "gitlab.33.cn/chain33/chain33/authority/cryptosuite/bccsp/sw"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/signingmgr"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/authority/identitymgr"
	"gitlab.33.cn/chain33/chain33/authority/mspmgr"
)

var alog = log.New("module", "autority")

type Authority struct {
	cryptoPath string
	client      queue.Client
	cfg         *types.Authority
	cryptoSuite core.CryptoSuite
	signer      *signingmgr.SigningManager
	identity    *identitymgr.IdentityManager
}

func New(conf *types.Authority) *Authority{
	auth := &Authority{}
	auth.initConfig(conf)
	auth.cryptoPath = conf.CryptoPath
	auth.cfg = conf

	return auth
}

func (auth *Authority) initConfig(conf *types.Authority) error {
	config := &cryptosuite.CryptoConfig{conf}

	cryptoSuite,err := sw.GetSuiteByConfig(config)
	if err != nil {
		return errors.WithMessage(err, "Failed to initialize crypto suite")
	}
	auth.cryptoSuite = cryptoSuite

	signer,err := signingmgr.New(cryptoSuite)
	if err != nil {
		return errors.WithMessage(err, "Failed to initialize signing manage")
	}
	auth.signer = signer

	identity,err := identitymgr.NewIdentityManager(conf.OrgName, cryptoSuite, conf.CryptoPath)
	if err != nil {
		return errors.WithMessage(err, "Failed to initialize identity manage")
	}
	auth.identity = identity

	mspmgr.LoadLocalMsp(conf.CryptoPath, config)
	return nil
}

func (auth *Authority) GetSigningMgr() *signingmgr.SigningManager {
	return auth.signer
}

func (auth *Authority) GetIdentityMgr() *identitymgr.IdentityManager {
	return auth.identity
}

func (auth *Authority) SetQueueClient(client queue.Client) {

}

func (auth *Authority) Close() {
	alog.Info("authority module closed")
}