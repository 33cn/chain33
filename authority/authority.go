package authority

import (
	log "github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/common/core"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
	"gitlab.33.cn/chain33/chain33/authority/identitymgr"
	"gitlab.33.cn/chain33/chain33/authority/mspmgr"
	"gitlab.33.cn/chain33/chain33/authority/signingmgr"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"fmt"
)

var alog = log.New("module", "autority")
var OrgName = "chain33"

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

	err := auth.initConfig(conf)
	if err != nil {
		panic("Initialize authority module failed")
	}

	auth.cryptoPath = conf.CryptoPath
	auth.cfg = conf
	OrgName = conf.GetOrgName()

	return auth
}

func (auth *Authority) initConfig(conf *types.Authority) error {
	config := &cryptosuite.CryptoConfig{conf}

	cryptoSuite, err := cryptosuite.GetSuiteByConfig(config)
	if err != nil {
		return errors.WithMessage(err, "Failed to initialize crypto suite")
	}
	auth.cryptoSuite = cryptoSuite

	signer, err := signingmgr.New(cryptoSuite)
	if err != nil {
		return errors.WithMessage(err, "Failed to initialize signing manage")
	}
	auth.signer = signer

	idmgr, err := identitymgr.NewIdentityManager(OrgName, cryptoSuite, conf.CryptoPath)
	if err != nil {
		return errors.WithMessage(err, "Failed to initialize identity manage")
	}
	auth.idmgr = idmgr

	err = mspmgr.LoadLocalMsp(conf.CryptoPath, config)
	if err != nil {
		return err
	}

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
			} else if msg.Ty == types.EventAuthorityCheckTxs {

			}
		}
	}()
}

//签名与CERT独立发送，验证签名时，如有异常，可以先把这两个数据拷贝置空再验证
//或者在此处重新构造TX数据
func (auth *Authority) procSignTx(msg queue.Message) {
	data, _ := msg.GetData().(*types.ReqAuthSignTx)

	alog.Debug("Process authority tx signing begin")
	var tx types.Transaction
	err := types.Decode(data.Tx, &tx)
	if err != nil {
		alog.Error("ReqAuthSignTx decode error, wrong data format")
		msg.ReplyErr("EventReplyAuthSignTx", types.ErrInvalidParam)
		return
	}

	userName := tx.GetCert().Username
	user, err := auth.idmgr.GetUser(userName)
	if err != nil {
		alog.Error(fmt.Sprintf("Wrong username:%s", userName))
		msg.ReplyErr("EventReplyAuthSignTx", types.ErrInvalidParam)
		return
	}

	if tx.Signature != nil {
		alog.Error("Signature already exist")
		msg.ReplyErr("EventReplyAuthSignTx", types.ErrSignatureExist)
		return
	}

	tx.Cert.Certbytes = append(tx.Cert.Certbytes, user.EnrollmentCertificate()...)
	signature, err := auth.signer.Sign(types.Encode(&tx), user.PrivateKey())
	if err != nil {
		panic(err)
	}
	tx.Signature = &types.Signature{types.SIG_TYPE_AUTHORITY, nil, signature}

	alog.Debug("Process authority tx signing end")

	txHex := types.Encode(&tx)
	msg.Reply(auth.client.NewMessage("", types.EventReplyAuthSignTx, &types.ReplyAuthSignTx{txHex}))
}

func (auth *Authority) procCheckTx(msg queue.Message) {
	data := msg.GetData().(*types.ReqAuthSignCheck)
	tx := data.GetTx()
	if tx.GetSignature().Ty != types.SIG_TYPE_AUTHORITY {
		alog.Error("Error signature type, should be AUTHORITY")
		msg.ReplyErr("EventReplyAuthCheckTx", types.ErrInvalidParam)
		return
	}
	falseResultMsg := auth.client.NewMessage("", types.EventReplyAuthCheckTx, &types.RespAuthSignCheck{false})

	alog.Debug(fmt.Sprintf("transaction user name %s", tx.GetCert().GetUsername()))

	localMSP := mspmgr.GetLocalMSP()
	identity, err := localMSP.DeserializeIdentity(tx.GetCert().GetCertbytes())
	if err != nil {
		alog.Error("Deserialize identity from cert bytes failed", err.Error())
		msg.Reply(falseResultMsg)
		return
	}

	err = identity.Validate()
	if err != nil {
		alog.Error("Validate certificates failed", err.Error())
		msg.Reply(falseResultMsg)
		return
	}
	alog.Debug("Certificates is valid")

	copytx := *tx
	copytx.Signature = nil
	bytes := types.Encode(&copytx)
	err = identity.Verify(bytes, tx.GetSignature().GetSignature())
	if err != nil {
		alog.Error("Verify signature failed", err.Error())
		msg.Reply(falseResultMsg)
		return
	}
	alog.Debug("Verify signature success")

	msg.Reply(auth.client.NewMessage("", types.EventReplyAuthCheckTx, &types.RespAuthSignCheck{true}))
}

func (auth *Authority) procCheckTxs(msg queue.Message) {
	data := msg.GetData().(*types.ReqAuthSignCheckTxs)
	txs := data.GetTxs()
	if len(txs) == 0 {
		alog.Error("Empty transactions")
		msg.ReplyErr("EventReplyAuthCheckTx", types.ErrInvalidParam)
		return
	}

	//TODO
	msg.Reply(auth.client.NewMessage("", types.EventReplyAuthCheckTxs, &types.RespAuthSignCheckTxs{true}))
}

func (auth *Authority) Close() {
	alog.Info("authority module closed")
}
