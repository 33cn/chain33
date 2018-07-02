package authority

import (
	"fmt"
	"path"
	"runtime"

	log "github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/core"
	"gitlab.33.cn/chain33/chain33/authority/utils"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"io/ioutil"
	"sync"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"encoding/asn1"
)

var alog = log.New("module", "autority")
var OrgName = "Chain33"
var cpuNum  = runtime.NumCPU()

type Authority struct {
	cryptoPath string
	client     queue.Client
	cfg        *types.Authority
	signType   int32
	userMap    map[string]*User
	validator   core.Validator
}

type User struct {
	id                    string
	enrollmentCertificate []byte
	privateKey            []byte
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

func (auth *Authority) loadUsers(configPath string) error {
	auth.userMap = make(map[string]*User)

	certDir := path.Join(configPath, "signcerts")
	dir,err := ioutil.ReadDir(certDir)
	if err != nil {
		return err
	}

	keyDir := path.Join(configPath, "keystore")
	for _,file := range dir {
		filePath := path.Join(certDir,file.Name())
		certBytes, err := utils.ReadFile(filePath)
		if err != nil {
			continue
		}

		ski,err := utils.GetPublicKeySKIFromCert(certBytes)
		if err != nil {
			alog.Error(fmt.Sprintf("Value in certificate file:%s not found", filePath))
			continue
		}
		filePath = path.Join(keyDir, ski+"_sk")
		KeyBytes, err := utils.ReadFile(filePath)
		if err != nil {
			continue
		}

		auth.userMap[file.Name()] = &User{file.Name(), certBytes, KeyBytes}
	}

	return nil
}

func (auth *Authority) initConfig(conf *types.Authority) error {
	types.IsAuthEnable = true

	if len(conf.CryptoPath) == 0 {
		return errors.New("Config path cannot be null")
	}

	err := auth.loadUsers(conf.CryptoPath)
	if err != nil {
		return errors.WithMessage(err, "Failed to load users' file")
	}

	vldt, err := core.GetLocalValidator(conf.CryptoPath)
	if err != nil {
		return err
	}
	auth.validator = vldt

	return nil
}

func (auth *Authority) SetQueueClient(client queue.Client) {
	if types.IsAuthEnable {
		auth.client = client
		auth.client.Sub("authority")

		//recv 消息的处理
		go func() {
			for msg := range client.Recv() {
				alog.Debug("authority recv", "msg", msg)
				if msg.Ty == types.EventAuthorityGetUser {
					go auth.procGetUser(msg)
				} else if msg.Ty == types.EventAuthorityCheckCert {
					go auth.procCheckCert(msg)
				} else if msg.Ty == types.EventAuthorityCheckCerts {
					go auth.procCheckCerts(msg)
				}
			}
		}()
	}
}

func (auth *Authority) procCheckCerts(msg queue.Message) {
	data,_ := msg.GetData().(*types.ReqAuthCheckCerts)

	done := make(chan struct{})
	defer close(done)

	taskes := gen(done, data.GetSig())

	c := make(chan result)
	var wg sync.WaitGroup
	wg.Add(cpuNum)
	for i := 0; i < cpuNum; i++ {
		go func() {
			auth.checksign(done, taskes, c)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c)
	}()

	for r := range c {
		if r.err != nil {
			msg.Reply(auth.client.NewMessage("", types.EventReplyAuthCheckCerts, &types.ReplyAuthCheckCerts{false}))
			return
		}
	}

	msg.Reply(auth.client.NewMessage("", types.EventReplyAuthCheckCerts, &types.ReplyAuthCheckCerts{true}))
}

func gen(done <-chan struct{}, task []*types.Signature) <-chan *types.Signature {
	ch := make(chan *types.Signature)
	go func() {
		defer func() {
			close(ch)
		}()
		for i := 0; i < len(task); i++ {
			select {
			case ch <- task[i]:
			case <-done:
				return
			}
		}
	}()
	return ch
}

type result struct {
	err error
}

func (auth *Authority) checksign(done <-chan struct{}, taskes <-chan *types.Signature, c chan<- result) {
	for task := range taskes {
		select {
		case c <- result{auth.checkCert(task)}:
		case <-done:
			return
		}
	}
}

func (auth *Authority) checkCert(signature *types.Signature) error {
	var certSignature crypto.CertSignature
	_, err := asn1.Unmarshal(signature.Signature, &certSignature)
	if err != nil {
		alog.Error(fmt.Sprintf("unmashal certificate from signature failed. %s", err.Error()))
		return err
	}

	if len(certSignature.Cert) == 0 {
		alog.Error("cert can not be null")
		return types.ErrInvalidParam
	}

	err = auth.validator.Validate(certSignature.Cert, signature.GetPubkey())
	if err != nil {
		alog.Error(fmt.Sprintf("validate cert failed. %s", err.Error()))
		return fmt.Errorf("validate cert failed. error:%s", err.Error())
	}

	return nil
}

func (auth *Authority) procCheckCert(msg queue.Message) {
	data,_ := msg.GetData().(*types.ReqAuthCheckCert)

	err := auth.checkCert(data.GetSig())
	if err != nil {
		msg.Reply(auth.client.NewMessage("", types.EventReplyAuthCheckCert, &types.ReplyAuthCheckCert{false}))
		return
	}

	msg.Reply(auth.client.NewMessage("", types.EventReplyAuthCheckCert, &types.ReplyAuthCheckCert{true}))
}

func (auth *Authority) procGetUser(msg queue.Message) {
	data, _ := msg.GetData().(*types.ReqAuthGetUser)

	userName := data.GetName()
	keyvalue := fmt.Sprintf("%s@%s-cert.pem", userName, OrgName)
	user,ok := auth.userMap[keyvalue]
	if !ok {
		msg.ReplyErr("EventReplyAuthGetUser", types.ErrInvalidParam)
		return
	}

	resp := &types.ReplyAuthGetUser{}
	resp.Cert = append(resp.Cert, user.enrollmentCertificate...)
	resp.Key = append(resp.Key, user.privateKey...)

	msg.Reply(auth.client.NewMessage("", types.EventReplyAuthGetUser,resp))
}

func (auth *Authority) Close() {
	alog.Info("authority module closed")
}
