package authority

import (
	"fmt"
	"testing"

	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common/crypto"
)

var (
	amount   = int64(1e8)
	v        = &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer = &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx1      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 2}
	tx2      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0}
	tx3      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 200000000, Expire: 0}
	tx4      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 300000000, Expire: 0}
	tx5      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 400000000, Expire: 0}
	tx6      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 500000000, Expire: 0}
	tx7      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 600000000, Expire: 0}
	tx8      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 700000000, Expire: 0}
	tx9      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 800000000, Expire: 0}
	tx10     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 900000000, Expire: 0}
	tx11     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 450000000, Expire: 0}
	tx12     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0}
	tx13     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100, Expire: 0}
)

var USERNAME = "User"

func sign(priv crypto.PrivKey, cert []byte) {
	tx1.Sign(types.AUTH_ECDSA, priv)
	tx1.Signature.Cert = append(tx1.Signature.Cert, cert...)

	tx2.Sign(types.AUTH_ECDSA, priv)
	tx2.Signature.Cert = append(tx2.Signature.Cert, cert...)

	tx3.Sign(types.AUTH_ECDSA, priv)
	tx3.Signature.Cert = append(tx3.Signature.Cert, cert...)

	tx4.Sign(types.AUTH_ECDSA, priv)
	tx4.Signature.Cert = append(tx4.Signature.Cert, cert...)

	tx5.Sign(types.AUTH_ECDSA, priv)
	tx5.Signature.Cert = append(tx5.Signature.Cert, cert...)

	tx6.Sign(types.AUTH_ECDSA, priv)
	tx6.Signature.Cert = append(tx6.Signature.Cert, cert...)

	tx7.Sign(types.AUTH_ECDSA, priv)
	tx7.Signature.Cert = append(tx7.Signature.Cert, cert...)

	tx8.Sign(types.AUTH_ECDSA, priv)
	tx8.Signature.Cert = append(tx8.Signature.Cert, cert...)

	tx9.Sign(types.AUTH_ECDSA, priv)
	tx9.Signature.Cert = append(tx9.Signature.Cert, cert...)

	tx10.Sign(types.AUTH_ECDSA, priv)
	tx10.Signature.Cert = append(tx10.Signature.Cert, cert...)

	tx11.Sign(types.AUTH_ECDSA, priv)
	tx11.Signature.Cert = append(tx11.Signature.Cert, cert...)

	tx12.Sign(types.AUTH_ECDSA, priv)
	tx12.Signature.Cert = append(tx12.Signature.Cert, cert...)

	tx13.Sign(types.AUTH_ECDSA, priv)
	tx13.Signature.Cert = append(tx13.Signature.Cert, cert...)
}

func initEnv() (queue.Queue, *Authority, error) {
	var q = queue.New("channel")
	cfg := config.InitCfg("../cmd/chain33/chain33.test.toml")

	types.SetMinFee(0)

	auth := New(cfg.Auth)
	auth.SetQueueClient(q.Client())

	msg := auth.client.NewMessage("authority", types.EventAuthorityGetUser, &types.ReqAuthGetUser{USERNAME})
	auth.client.Send(msg, true)
	resp, err := auth.client.Wait(msg)
	if err != nil {
		return nil, nil, fmt.Errorf("get user %s failed, error:%s", USERNAME, err)
	}
	cert := resp.Data.(*types.ReplyAuthGetUser).Cert
	key := resp.Data.(*types.ReplyAuthGetUser).Key

	cr, err := crypto.New(types.GetSignatureTypeName(types.AUTH_ECDSA))
	if err != nil {
		return nil, nil, fmt.Errorf("create crypto %s failed, error:%s", types.GetSignatureTypeName(types.AUTH_ECDSA), err)
	}

	priv, err := cr.PrivKeyFromBytes(key)
	if err != nil {
		return nil, nil, fmt.Errorf("get private key failed, error:%s", err)
	}
	sign(priv, cert)

	return q, auth, nil
}

func TestCheckTx(t *testing.T) {
	q, auth, err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
	}
	defer q.Close()
	defer auth.Close()

	txReq := &types.ReqAuthCheckCert{tx1.Signature}
	msg := auth.client.NewMessage("authority", types.EventAuthorityCheckCert, txReq)
	auth.client.Send(msg, true)
	resp, err := auth.client.Wait(msg)
	if err != nil {
		t.Error(err)
		return
	}

	respData := resp.GetData().(*types.ReplyAuthCheckCert).GetResult()
	if !respData {
		t.Error("error process txs signature validate")
	}
}

func TestCheckTxs(t *testing.T) {
	q, auth, err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
	}
	defer q.Close()
	defer auth.Close()

	signatures := []*types.Signature{tx1.Signature, tx2.Signature, tx3.Signature, tx4.Signature, tx5.Signature,
					tx6.Signature, tx7.Signature, tx8.Signature, tx9.Signature, tx10.Signature, tx11.Signature,
					tx12.Signature, tx13.Signature}
	txsReq := &types.ReqAuthCheckCerts{signatures}
	msg := auth.client.NewMessage("authority", types.EventAuthorityCheckCerts, txsReq)
	auth.client.Send(msg, true)
	resp, err := auth.client.Wait(msg)
	if err != nil {
		t.Error(err)
		return
	}

	respData := resp.GetData().(*types.ReplyAuthCheckCerts).GetResult()
	if !respData {
		t.Error("error process txs signature validate")
	}
}

