package authority

import (
	"encoding/asn1"
	"fmt"
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	transfer = &types.CertAction{Value: nil, Ty: types.CertActionNormal}
	to       = drivers.ExecAddress("cert")
	tx1      = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 2, To: to}
	tx2      = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: to}
	tx3      = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 200000000, Expire: 0, To: to}
	tx4      = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 300000000, Expire: 0, To: to}
	tx5      = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 400000000, Expire: 0, To: to}
	tx6      = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 500000000, Expire: 0, To: to}
	tx7      = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 600000000, Expire: 0, To: to}
	tx8      = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 700000000, Expire: 0, To: to}
	tx9      = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 800000000, Expire: 0, To: to}
	tx10     = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 900000000, Expire: 0, To: to}
	tx11     = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 450000000, Expire: 0, To: to}
	tx12     = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0, To: to}
	tx13     = &types.Transaction{Execer: []byte("cert"), Payload: types.Encode(transfer), Fee: 100, Expire: 0, To: to}
	txs      = []*types.Transaction{tx1, tx2, tx3, tx4, tx5, tx6, tx7, tx8, tx9, tx10, tx11, tx12, tx13}
)

var USERNAME = "User"
var SIGNTYPE = types.AUTH_SM2

func signtx(tx *types.Transaction, priv crypto.PrivKey, cert []byte) {
	tx.Sign(int32(SIGNTYPE), priv)
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

func initEnv() error {
	cfg := config.InitCfg("./test/chain33.auth.test.toml")

	Author.Init(cfg.Auth)

	userLoader := &UserLoader{}
	err := userLoader.Init(cfg.Auth.CryptoPath)
	if err != nil {
		fmt.Printf("Init user loader falied")
		return err
	}

	user, err := userLoader.GetUser(USERNAME)
	if err != nil {
		fmt.Printf("Get user failed")
		return err
	}

	cr, err := crypto.New(types.GetSignatureTypeName(SIGNTYPE))
	if err != nil {
		return fmt.Errorf("create crypto %s failed, error:%s", types.GetSignatureTypeName(SIGNTYPE), err)
	}

	priv, err := cr.PrivKeyFromBytes(user.Key)
	if err != nil {
		return fmt.Errorf("get private key failed, error:%s", err)
	}
	signtxs(priv, user.Cert)

	if err != nil {
		fmt.Printf("Init authority failed")
		return err
	}

	return nil
}

func TestChckSign(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	prev := types.MinFee
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	if !tx1.CheckSign() {
		t.Error("check signature failed")
		return
	}
}

func TestChckSigns(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	prev := types.MinFee
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	block := types.Block{}
	block.Txs = txs
	if !block.CheckSign() {
		t.Error("error check txs")
		return
	}
}

func TestValidateCert(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}

	prev := types.MinFee
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	for _,tx := range txs {
		err = Author.Validate(tx.Signature)
		if err != nil {
			t.Error("error cert validate", err.Error())
			return
		}
	}
}

//FIXME 有并发校验的场景需要考虑竞争，暂时没有并发校验的场景
/*
func TestValidateCerts(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
	}

	prev := types.MinFee
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	signatures := []*types.Signature{tx1.Signature, tx2.Signature, tx3.Signature, tx4.Signature, tx5.Signature,
		tx6.Signature, tx7.Signature, tx8.Signature, tx9.Signature, tx10.Signature, tx11.Signature,
		tx12.Signature, tx13.Signature}

	result := Author.ValidateCerts(signatures)
	if !result {
		t.Error("error process txs signature validate")
	}
}
*/

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
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}

	prev := types.MinFee
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	block := createBlock(10)

	txlist := &types.ExecTxList{}
	txlist.Txs = txs
	txlist.BlockTime = block.BlockTime
	txlist.Height = block.Height
	txlist.StateHash = block.StateHash
	// 增加这个属性，在执行器中会使用到
	txlist.Difficulty = uint64(block.Difficulty)
}

func TestReloadCert(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}

	prev := types.MinFee
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	store := &types.HistoryCertStore{}

	Author.ReloadCert(store)

	err = Author.Validate(tx1.Signature)
	if err != nil {
		t.Error(err.Error())
	}
}

func TestReloadByHeight(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}

	prev := types.MinFee
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	Author.ReloadCertByHeght(30)

	if Author.HistoryCertCache.CurHeight != 30 {
		t.Error("reload by height failed")
	}
}
