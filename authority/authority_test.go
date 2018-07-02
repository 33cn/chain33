package authority

import (
	"encoding/asn1"
	"fmt"
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	amount   = int64(1e8)
	v        = &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer = &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	to       = drivers.ExecAddress("norm")
	tx1      = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 2, To: to}
	tx2      = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: to}
	tx3      = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 200000000, Expire: 0, To: to}
	tx4      = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 300000000, Expire: 0, To: to}
	tx5      = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 400000000, Expire: 0, To: to}
	tx6      = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 500000000, Expire: 0, To: to}
	tx7      = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 600000000, Expire: 0, To: to}
	tx8      = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 700000000, Expire: 0, To: to}
	tx9      = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 800000000, Expire: 0, To: to}
	tx10     = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 900000000, Expire: 0, To: to}
	tx11     = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 450000000, Expire: 0, To: to}
	tx12     = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0, To: to}
	tx13     = &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: 100, Expire: 0, To: to}
	txs      = []*types.Transaction{tx1, tx2, tx3, tx4, tx5, tx6, tx7, tx8, tx9, tx10, tx11, tx12, tx13}
)

var USERNAME = "User"

func signtx(tx *types.Transaction, priv crypto.PrivKey, cert []byte) {
	tx.Sign(types.AUTH_ECDSA, priv)
	certSign := crypto.CertSignature{}
	certSign.Signature = append(certSign.Signature, tx.Signature.Signature...)
	certSign.Cert = append(certSign.Cert, cert...)
	tx.Signature.Signature, _ = asn1.Marshal(certSign)
}

func signtxs(priv crypto.PrivKey, cert []byte) {
	signtx(tx1, priv, cert)
	signtx(tx2, priv, cert)
	signtx(tx3, priv, cert)
	signtx(tx4, priv, cert)
	signtx(tx5, priv, cert)
	signtx(tx6, priv, cert)
	signtx(tx7, priv, cert)
	signtx(tx8, priv, cert)
	signtx(tx9, priv, cert)
	signtx(tx10, priv, cert)
	signtx(tx11, priv, cert)
	signtx(tx12, priv, cert)
	signtx(tx13, priv, cert)

}

func initEnv() (queue.Queue, *Authority, error) {
	var q = queue.New("channel")
	cfg := config.InitCfg("./chain33.auth.test.toml")

	types.SetMinFee(0)

	auth := New(cfg.Auth)
	auth.SetQueueClient(q.Client())

	exec := executor.New(cfg.Exec)
	exec.SetQueueClient(q.Client())

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
	signtxs(priv, cert)

	return q, auth, nil
}

func TestChckSign(t *testing.T) {
	q, auth, err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
	}
	defer q.Close()
	defer auth.Close()

	if !tx1.CheckSign() {
		t.Error("check signature failed")
		return
	}
}

func TestChckSigns(t *testing.T) {
	q, auth, err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
	}
	defer q.Close()
	defer auth.Close()

	block := types.Block{}
	block.Txs = txs
	if !block.CheckSign() {
		t.Error("error check txs")
		return
	}
}

func TestValidateCert(t *testing.T) {
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

func TestValidateCerts(t *testing.T) {
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

func createBlock(n int64) *types.Block {
	var zeroHash [32]byte
	newblock := &types.Block{}
	newblock.Height = -1
	newblock.BlockTime = time.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = txs
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

func TestCheckTx(t *testing.T) {
	q, auth, err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
	}

	defer q.Close()
	defer auth.Close()

	block := createBlock(10)

	txlist := &types.ExecTxList{}
	txlist.Txs = txs
	txlist.BlockTime = block.BlockTime
	txlist.Height = block.Height
	txlist.StateHash = block.StateHash
	// 增加这个属性，在执行器中会使用到
	txlist.Difficulty = uint64(block.Difficulty)

	msg := auth.client.NewMessage("execs", types.EventCheckTx, txlist)
	err = auth.client.Send(msg, true)
	if err != nil {
		t.Error(err)
		return
	}
	msg, err = auth.client.Wait(msg)
	if err != nil {
		t.Error(err)
		return
	}

	result := msg.GetData().(*types.ReceiptCheckTxList)
	for i := 0; i < len(result.Errs); i++ {
		if result.Errs[i] != "" {
			t.Error(result.Errs[i])
			return
		}
	}
}
