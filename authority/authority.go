package authority

import (
	log "github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite/bccsp/sw"
	"gitlab.33.cn/chain33/chain33/authority/identitymgr"
	"gitlab.33.cn/chain33/chain33/authority/mspmgr"
	"gitlab.33.cn/chain33/chain33/authority/signingmgr"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var alog = log.New("module", "autority")
var orgName = "org1"
var userName = "User"

type Authority struct {
	cryptoPath  string
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

	var cryconf cryptosuite.CryptoConfig
	cryconf.Config = auth.cfg
	mspmgr.LoadLocalMsp(auth.cryptoPath, &cryconf)

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

//签名与CERT独立发送，验证签名时，如有异常，可以先把这两个数据拷贝置空再验证
//或者在此处重新构造TX数据
func (auth *Authority) procSignTx(msg queue.Message) {
	//var key core.Key
	//var tx types.transaction
	data, ok := msg.GetData().(*types.ReqAuthSignTx)
	if !ok {
		panic("")
	}
	//var action types.HashlockAction
	//err := types.Decode(data.Tx, &tx)

	user, err := auth.idmgr.GetUser(userName)
	if err != nil {
		panic(err)
	}

	//tx.Cert.Certbytes = user.enrollmentCertificate
	//tx.Cert.Username = user.id

	signature, err := auth.signer.Sign(data.Tx, user.PrivateKey())
	if err != nil {
		panic(err)
	}

	msg.Reply(auth.client.NewMessage("", types.EventReplyAuthSignTx, &types.ReplyAuthSignTx{signature, user.EnrollmentCertificate(), userName}))
}

func (auth *Authority) procCheckTx(msg queue.Message) {

}

func (auth *Authority) Close() {
	alog.Info("authority module closed")
}
