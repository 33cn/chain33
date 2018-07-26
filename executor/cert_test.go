package executor

import (
	"encoding/asn1"
	"fmt"
	"testing"

	"gitlab.33.cn/chain33/chain33/authority"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	USERNAME = "User"

	to        = drivers.ExecAddress("cert")
	transfer1 = &types.CertAction{Value: nil, Ty: types.CertActionNew}
	tx1       = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer1), Fee: 1000000, Expire: 0, To: to}

	transfer2 = &types.CertAction{Value: nil, Ty: types.CertActionUpdate}
	tx2       = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer2), Fee: 2000000, Expire: 0, To: to}

	transfer3 = &types.CertAction{Value: nil, Ty: types.CertActionNormal}
	tx3       = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer3), Fee: 3000000, Expire: 0, To: to}
	tx4       = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer3), Fee: 4000000, Expire: 0, To: to}
)

var SIGNTYPE = types.SignNameAuthSM2

func signtx(tx *types.Transaction, priv crypto.PrivKey, cert []byte) {
	tx.Sign(int32(types.MapSignName2Type[SIGNTYPE]), priv)
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
}

func initCertEnv() (queue.Queue, error) {
	q, _ := initUnitEnv()

	cfgAuth := types.Authority{true, "../authority/test/authdir/crypto", SIGNTYPE}
	authority.Author.Init(&cfgAuth)

	userLoader := &authority.UserLoader{}
	err := userLoader.Init(cfgAuth.CryptoPath, cfgAuth.SignType)
	if err != nil {
		fmt.Printf("Init user loader falied")
		return nil, err
	}

	user, err := userLoader.Get(USERNAME)
	if err != nil {
		fmt.Printf("Get user failed")
		return nil, err
	}

	signtxs(user.Key, user.Cert)

	return q, nil
}

func createBlockCert(txs []*types.Transaction) *types.Block {
	newblock := &types.Block{}
	newblock.Height = 2
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = append(newblock.Txs, txs...)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

func genEventAddBlockMsgCert(client queue.Client, block *types.Block) queue.Message {
	blockDetail := constructionBlockDetail(block, 1, 2)
	msg := client.NewMessage("execs", types.EventAddBlock, blockDetail)
	return msg
}

/**
Testcase01 证书管理new，update，normal
 */
func TestCertMgr(t *testing.T) {
	q, _ := initCertEnv()
	storeProcess(q)
	blockchainProcess(q)

	prev := types.MinFee
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	var msgs []queue.Message
	var msg queue.Message

	txs := []*types.Transaction{tx1, tx3}
	block := createBlockCert(txs)
	msg = genEventAddBlockMsgCert(q.Client(), block)
	msgs = append(msgs, msg)

	txs = []*types.Transaction{tx2, tx3}
	block = createBlockCert(txs)
	msg = genEventAddBlockMsgCert(q.Client(), block)
	msgs = append(msgs, msg)

	go func() {
		for _, msga := range msgs {
			q.Client().Send(msga, true)
			_, err := q.Client().Wait(msga)
			if err == nil || err == types.ErrNotFound || err == types.ErrEmpty {
				t.Logf("%v,%v", msga, err)
			} else {
				t.Error(err)
			}
		}
		q.Close()
	}()
	q.Start()
}


/**
TestCase02 交易校验
 */
func TestCertTxCheck(t *testing.T) {
	q, _ := initCertEnv()
	storeProcess(q)
	blockchainProcess(q)

	prev := types.MinFee
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	var msgs []queue.Message
	var msg queue.Message

	txs := []*types.Transaction{tx3, tx4}
	block := createBlockCert(txs)
	msg = genExecCheckTxMsg(q.Client(), block)
	msgs = append(msgs, msg)

	go func() {
		for _, msga := range msgs {
			q.Client().Send(msga, true)
			_, err := q.Client().Wait(msga)
			if err == nil || err == types.ErrNotFound || err == types.ErrEmpty {
				t.Logf("%v,%v", msga, err)
			} else {
				t.Error(err)
			}
		}
		q.Close()
	}()
	q.Start()
}

/**
TestCase03 回滚验证
 */
func TestCertTxCheckRollback(t *testing.T) {
	q, _ := initCertEnv()
	storeProcess(q)
	blockchainProcess(q)

	prev := types.MinFee
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	var msgs []queue.Message
	var msg queue.Message

	txs := []*types.Transaction{tx3, tx4}
	block := createBlockCert(txs)
	msg = genExecCheckTxMsg(q.Client(), block)
	msgs = append(msgs, msg)

	authority.Author.HistoryCertCache.CurHeight = 6
	go func() {
		for _, msga := range msgs {
			q.Client().Send(msga, true)
			_, err := q.Client().Wait(msga)
			if err == nil || err == types.ErrNotFound || err == types.ErrEmpty {
				t.Logf("%v,%v", msga, err)
			} else {
				t.Error(err)
			}
		}
		q.Close()
	}()
	q.Start()
}
