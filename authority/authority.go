package authority

import (
	"crypto/ecdsa"
	"io/ioutil"

	log "github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	"gitlab.33.cn/chain33/chain33/authority/bccsp/utils"
	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	//"gitlab.33.cn/chain33/chain33/authority/common/util/cryptoutils"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite/bccsp/sw"
	"gitlab.33.cn/chain33/chain33/authority/signingmgr"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var alog = log.New("module", "autority")

type Authority struct {
	cryptoPath  string
	client      queue.Client
	cfg         *types.Authority
	cryptoSuite core.CryptoSuite
	signer      *signingmgr.SigningManager
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

	return nil
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
	var key core.Key
	data, ok := msg.GetData().(*types.ReqAuthSignTx)
	if !ok {
		panic("")
	}

	keyBuff, err := ioutil.ReadFile("authdir/keystore/keystore/user.pem")
	if err != nil {
		panic(err)
	}

	key, err = importKeyFromPEMBytes(keyBuff, auth.cryptoSuite, true)

	signature, err := auth.signer.Sign(data.Tx, key)
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

//remove to other package later
func importKeyFromPEMBytes(keyBuff []byte, myCSP core.CryptoSuite, temporary bool) (core.Key, error) {

	key, err := utils.PEMtoPrivateKey(keyBuff, nil)
	if err != nil {
		return nil, err
	}
	switch key.(type) {
	case *ecdsa.PrivateKey:
		priv, err := utils.PrivateKeyToDER(key.(*ecdsa.PrivateKey))
		if err != nil {
			return nil, err
		}
		sk, err := myCSP.KeyImport(priv, &bccsp.ECDSAPrivateKeyImportOpts{Temporary: temporary})
		if err != nil {
			return nil, err
		}
		return sk, nil
	//case *rsa.PrivateKey:
	//return nil, errors.Errorf("Failed to import RSA key from %s; RSA private key import is not supported", keyFile)
	default:
		return nil, nil
	}
}
