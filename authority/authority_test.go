package authority

import (
	"fmt"
	"testing"

	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	amount   = int64(1e8)
	v        = &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer = &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx1      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 2, Cert: &types.AuthCert{nil, "User1"}}
	tx2      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
	tx3      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 200000000, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
	tx4      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 300000000, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
	tx5      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 400000000, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
	tx6      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 500000000, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
	tx7      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 600000000, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
	tx8      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 700000000, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
	tx9      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 800000000, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
	tx10     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 900000000, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
	tx11     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 450000000, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
	tx12     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
	tx13     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100, Expire: 0, Cert: &types.AuthCert{nil, "User1"}}
)

var USERNAME = "User1"

func sign(auth *Authority, tx *types.Transaction) {
	user, err := auth.GetIdentityMgr().GetUser(USERNAME)
	if err != nil {
		alog.Error(fmt.Sprintf("Wrong username:%s", USERNAME))
		return
	}

	if tx.Signature != nil {
		alog.Error("Signature already exist")
		return
	}

	tx.Cert.Certbytes = append(tx.Cert.Certbytes, user.EnrollmentCertificate()...)
	signature, err := auth.signer.Sign(types.Encode(tx), user.PrivateKey())
	if err != nil {
		panic(err)
	}
	tx.Signature = &types.Signature{1, nil, signature}
}

func initEnv() (queue.Queue, *Authority) {
	var q = queue.New("channel")
	cfg := config.InitCfg("../cmd/chain33/chain33.test.toml")

	types.SetMinFee(0)

	auth := New(cfg.Auth)
	auth.SetQueueClient(q.Client())

	sign(auth, tx1)
	sign(auth, tx2)
	sign(auth, tx3)
	sign(auth, tx4)
	sign(auth, tx5)
	sign(auth, tx6)
	sign(auth, tx7)
	sign(auth, tx8)
	sign(auth, tx9)
	sign(auth, tx10)
	sign(auth, tx11)
	sign(auth, tx12)
	sign(auth, tx13)

	return q, auth
}

func TestCheckTxs(t *testing.T) {
	q, auth := initEnv()
	defer q.Close()
	defer auth.Close()

	txs := []*types.Transaction{tx1, tx2, tx3, tx4, tx5, tx6, tx7, tx8, tx9, tx10, tx11, tx12, tx13}
	txsReq := &types.ReqAuthSignCheckTxs{txs}
	msg := auth.client.NewMessage("authority", types.EventAuthorityCheckTxs, txsReq)
	auth.client.Send(msg, true)
	resp, err := auth.client.Wait(msg)
	if err != nil {
		t.Error(err)
		return
	}

	respData := resp.GetData().(*types.RespAuthSignCheckTxs).GetResult()
	if respData != true {
		t.Error("error process txs signature validate")
	}
}
