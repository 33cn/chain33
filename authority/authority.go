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
	data := msg.GetData().(*types.ReqAuthSignCheck)
	tx := data.GetTx()
	if tx.GetSignature().Ty != types.SIG_TYPE_AUTHORITY {
		msg.ReplyErr("EventReplyAuthSignTx", types.ErrInvalidParam)
		return
	}

	if tx.GetCert() == nil {
		msg.ReplyErr("EventReplyAuthSignTx", types.ErrInvalidParam)
		return
	}

	falseResultMsg := auth.client.NewMessage("mempool", types.EventReplyAuthCheckTx, &types.RespAuthSignCheck{false})
	localMSP := mspmgr.GetLocalMSP()

	identity,err := localMSP.DeserializeIdentity(tx.GetCert().Certbytes)
	if err != nil {
		alog.Error("Deserialize identity from cert bytes failed", err)
		msg.Reply(falseResultMsg)
		return
	}

	alog.Debug("transaction user name %s", tx.GetCert().Username)

	err = identity.Validate()
	if err != nil {
		alog.Error("Validate certificates failed", err)
		msg.Reply(falseResultMsg)
		return
	}

	alog.Debug("Certificates is valid")

	copytx := *tx
	copytx.Signature = nil
	bytes := types.Encode(&copytx)
	err = identity.Verify(bytes,types.Encode(tx.GetSignature()))
	if err != nil {
		alog.Error("Verify signature failed", err)
		msg.Reply(falseResultMsg)
		return
	}

	alog.Debug("Verify signature success")

	msg.Reply(auth.client.NewMessage("mempool", types.EventReplyAuthCheckTx, &types.RespAuthSignCheck{true}))
}

func (auth *Authority) Close() {
	alog.Info("authority module closed")
}
