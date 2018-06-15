package authority

import (
	"fmt"
	"runtime"

	log "github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/common/core"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
	"gitlab.33.cn/chain33/chain33/authority/identitymgr"
	"gitlab.33.cn/chain33/chain33/authority/mspmgr"
	"gitlab.33.cn/chain33/chain33/authority/signingmgr"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
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
	signType    int32
}

func New(conf *types.Authority) *Authority {
	auth := &Authority{}
	if conf != nil && conf.Enable {
		err := auth.initConfig(conf)
		if err != nil {
			alog.Error("Initialize authority module failed", "Error", err.Error())
			panic("")
		}

		auth.signType = conf.SignType
		auth.cryptoPath = conf.CryptoPath
		auth.cfg = conf
		OrgName = conf.GetOrgName()

	}

	return auth
}

func (auth *Authority) initConfig(conf *types.Authority) error {
	config := &cryptosuite.CryptoConfig{conf}

	types.IsAuthEnable = true

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
	if types.IsAuthEnable {
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
					go auth.procCheckTxs(msg)
				} else if msg.Ty == types.EventAuthoritySignTxs {
					go auth.procSignTxs(msg)
				}
			}
		}()
	}
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
	tx.Signature = &types.Signature{auth.signType, nil, signature}

	alog.Debug("Process authority tx signing end")

	txHex := types.Encode(&tx)
	msg.Reply(auth.client.NewMessage("", types.EventReplyAuthSignTx, &types.ReplyAuthSignTx{txHex}))
}

//to be merged with Tx
func (auth *Authority) procSignTxs(msg queue.Message) {
	data, ok := msg.GetData().(*types.ReqAuthSignTxs)
	if !ok {
		alog.Error("Get Auth data err")
		panic("")
	}

	var reply types.ReplyAuthSignTxs

	alog.Debug("Process authority tx signing begin")
	var tx types.Transaction
	for i := range data.Txs {
		tx = *data.Txs[i]

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
		tx.Signature = &types.Signature{auth.signType, nil, signature}
		reply.Txs = append(reply.Txs, &tx)
	}
	alog.Debug("Process authority tx signing end")
	msg.Reply(auth.client.NewMessage("", types.EventReplyAuthSignTxs, &reply))
}

func (auth *Authority) checkTx(tx *types.Transaction) bool {
	alog.Debug(fmt.Sprintf("transaction user name %s", tx.GetCert().GetUsername()))

	localMSP := mspmgr.GetLocalMSP()
	identity, err := localMSP.DeserializeIdentity(tx.GetCert().GetCertbytes())
	if err != nil {
		alog.Error("Deserialize identity from cert bytes failed", "Error", err.Error())
		return false
	}

	err = identity.Validate()
	if err != nil {
		alog.Error("Validate certificates failed", "Error", err.Error())
		return false
	}
	alog.Debug("Certificates is valid")

	copytx := *tx
	copytx.Signature = nil
	bytes := types.Encode(&copytx)
	err = identity.Verify(bytes, tx.GetSignature().GetSignature())
	if err != nil {
		alog.Error("Verify signature failed", "Error", err.Error())
		return false
	}
	alog.Debug("Verify signature success")

	return true
}

func (auth *Authority) procCheckTx(msg queue.Message) {
	data := msg.GetData().(*types.ReqAuthSignCheck)
	tx := data.GetTx()
	if tx.GetSignature().Ty != auth.signType {
		alog.Error("Signature type in transaction is %d, but in authority is %d", tx.GetSignature().Ty, auth.signType)
		msg.ReplyErr("EventReplyAuthCheckTx", types.ErrInvalidParam)
		return
	}

	result := auth.checkTx(tx)
	if !result {
		msg.Reply(auth.client.NewMessage("", types.EventReplyAuthCheckTx, &types.RespAuthSignCheck{false}))
	} else {
		msg.Reply(auth.client.NewMessage("", types.EventReplyAuthCheckTx, &types.RespAuthSignCheck{true}))
	}
}

func (auth *Authority) procCheckTxs(msg queue.Message) {
	data := msg.GetData().(*types.ReqAuthSignCheckTxs)
	txs := data.GetTxs()
	if len(txs) == 0 {
		alog.Error("Empty transactions")
		msg.ReplyErr("EventReplyAuthCheckTxs", types.ErrInvalidParam)
		return
	}

	cpu := runtime.NumCPU()
	ok := types.CheckAll(txs, cpu, auth.checkTx)
	msg.Reply(auth.client.NewMessage("", types.EventReplyAuthCheckTxs, &types.RespAuthSignCheckTxs{ok}))
}

func (auth *Authority) Close() {
	alog.Info("authority module closed")
}
