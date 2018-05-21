package authority

import (
	log "github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite/bccsp/sw"
	"gitlab.33.cn/chain33/chain33/authority/identitymgr"
	"gitlab.33.cn/chain33/chain33/authority/signingmgr"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/authority/mspmgr"
)

var alog = log.New("module", "autority")
var orgName = "org1"
var userName = "User"

type Authority struct {
	cryptoPath string
	client      queue.Client
	cfg         *types.Authority
	cryptoSuite core.CryptoSuite
	signer      *signingmgr.SigningManager
	idmgr       *identitymgr.IdentityManager
}

func New(conf *types.Authority) *Authority {
	auth := &Authority{}
	auth.initConfig(conf)
	auth.cryptoPath = conf.CryptoPath
	auth.cfg = conf

	return auth
}

func (auth *Authority) initConfig(conf *types.Authority) error {
	config := &cryptosuite.CryptoConfig{conf}

	cryptoSuite, err := sw.GetSuiteByConfig(config)
	if err != nil {
		return errors.WithMessage(err, "Failed to initialize crypto suite")
	}
	auth.cryptoSuite = cryptoSuite

	signer, err := signingmgr.New(cryptoSuite)
	if err != nil {
		return errors.WithMessage(err, "Failed to initialize signing manage")
	}
	auth.signer = signer

	idmgr, err := identitymgr.NewIdentityManager(orgName, cryptoSuite, conf.CryptoPath)
	if err != nil {
		return errors.WithMessage(err, "Failed to initialize identity manage")
	}
	auth.idmgr = idmgr

	mspmgr.LoadLocalMsp(conf.CryptoPath, config)

	return nil
}

func (auth *Authority) GetSigningMgr() *signingmgr.SigningManager {
	return auth.signer
}

func (auth *Authority) GetIdentityMgr() *identitymgr.IdentityManager {
	return auth.idmgr
}

func (auth *Authority) SetQueueClient(client queue.Client) {
	auth.client = client
	auth.client.Sub("authority")

	//recv 消息的处理
	go func() {
		for msg := range client.Recv() {
			alog.Debug("authority recv", "msg", msg)
			if msg.Ty == types.EventAuthoritySignTx {
				go auth.procSignTx(msg)
			} else if msg.Ty == types.EventAuthorityCheckTx {
				go auth.procCheckTx(msg)
			}
		}
	}()
}

func (auth *Authority) procSignTx(msg queue.Message) {
	//var key core.Key
	data, ok := msg.GetData().(*types.ReqAuthSignTx)
	if !ok {
		panic("")
	}

	user, err := auth.idmgr.GetUser(userName)
	if err != nil {
		panic(err)
	}

	signature, err := auth.signer.Sign(data.Tx, user.PrivateKey())
	if err != nil {
		panic(err)
	}

	msg.Reply(auth.client.NewMessage("", types.EventReplyAuthSignTx, &types.ReplyAuthSignTx{signature}))
}

func (auth *Authority) procCheckTx(msg queue.Message) {

}

func (auth *Authority) Close() {
	alog.Info("authority module closed")
}
