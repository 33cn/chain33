package authority_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/cert/authority"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/cert/authority/utils"
	ct "gitlab.33.cn/chain33/chain33/plugin/dapp/cert/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"

	_ "gitlab.33.cn/chain33/chain33/plugin"
	_ "gitlab.33.cn/chain33/chain33/system"
)

var (
	transfer = &ct.CertAction{Value: nil, Ty: ct.CertActionNormal}
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
	txs      = []*types.Transaction{tx1, tx2, tx3, tx4, tx5, tx6, tx7, tx8, tx9, tx10, tx11, tx12}

	privRaw, _  = common.FromHex("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	tr          = &cty.CoinsAction_Transfer{&types.AssetsTransfer{Amount: int64(1e8)}}
	secpp256, _ = crypto.New(types.GetSignName("", types.SECP256K1))
	privKey, _  = secpp256.PrivKeyFromBytes(privRaw)
	tx14        = &types.Transaction{
		Execer:  []byte("coins"),
		Payload: types.Encode(&cty.CoinsAction{Value: tr, Ty: cty.CoinsActionTransfer}),
		Fee:     1000000,
		Expire:  2,
		To:      address.PubKeyToAddress(privKey.PubKey().Bytes()).String(),
	}
)

var USERNAME = "User"
var SIGNTYPE = ct.AUTH_SM2

func signtx(tx *types.Transaction, priv crypto.PrivKey, cert []byte) {
	tx.Sign(int32(SIGNTYPE), priv)
	tx.Signature.Signature, _ = utils.EncodeCertToSignature(tx.Signature.Signature, cert)
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

/**
初始化Author实例和userloader
*/
func initEnv() error {
	_, sub := types.InitCfg("./test/chain33.auth.test.toml")
	var subcfg ct.Authority
	if sub.Exec["cert"] != nil {
		types.MustDecode(sub.Exec["cert"], &subcfg)
	}
	authority.Author.Init(&subcfg)
	SIGNTYPE = types.GetSignType("cert", subcfg.SignType)

	userLoader := &authority.UserLoader{}
	err := userLoader.Init(subcfg.CryptoPath, subcfg.SignType)
	if err != nil {
		fmt.Printf("Init user loader falied -> %v", err)
		return err
	}

	user, err := userLoader.Get(USERNAME)
	if err != nil {
		fmt.Printf("Get user failed")
		return err
	}

	signtxs(user.Key, user.Cert)
	if err != nil {
		fmt.Printf("Init authority failed")
		return err
	}

	return nil
}

/**
TestCase01 带证书的交易验签
*/
func TestChckSign(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	prev := types.GInt("MinFee")
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	if !tx1.CheckSign() {
		t.Error("check signature failed")
		return
	}
}

/**
TestCase10 带证书的多交易验签
*/
func TestChckSigns(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	prev := types.GInt("MinFee")
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	for i, tx := range txs {
		if !tx.CheckSign() {
			t.Error(fmt.Sprintf("error check tx[%d]", i+1))
			return
		}
	}
}

/**
TestCase02 带证书的交易并行验签
*/
func TestChckSignsPara(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	prev := types.GInt("MinFee")
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	block := types.Block{}
	block.Txs = txs
	if !block.CheckSign() {
		t.Error("error check txs")
		return
	}
}

/**
TestCase03 不带证书，公链签名算法验证
*/
func TestChckSignWithNoneAuth(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	prev := types.GInt("MinFee")
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	tx14.Sign(types.SECP256K1, privKey)
	if !tx14.CheckSign() {
		t.Error("check signature failed")
		return
	}
}

/**
TestCase04 不带证书，SM2签名验证
*/
func TestChckSignWithSm2(t *testing.T) {
	sm2, err := crypto.New(types.GetSignName("cert", ct.AUTH_SM2))
	assert.Nil(t, err)
	privKeysm2, _ := sm2.PrivKeyFromBytes(privRaw)
	tx15 := &types.Transaction{Execer: []byte("coins"),
		Payload: types.Encode(&cty.CoinsAction{Value: tr, Ty: cty.CoinsActionTransfer}),
		Fee:     1000000, Expire: 2, To: address.PubKeyToAddress(privKeysm2.PubKey().Bytes()).String()}

	err = initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	prev := types.GInt("MinFee")
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	tx15.Sign(ct.AUTH_SM2, privKeysm2)
	if !tx15.CheckSign() {
		t.Error("check signature failed")
		return
	}
}

/**
TestCase05 不带证书，secp256r1签名验证
*/
func TestChckSignWithEcdsa(t *testing.T) {
	ecdsacrypto, _ := crypto.New(types.GetSignName("cert", ct.AUTH_ECDSA))
	privKeyecdsa, _ := ecdsacrypto.PrivKeyFromBytes(privRaw)
	tx16 := &types.Transaction{Execer: []byte("coins"),
		Payload: types.Encode(&cty.CoinsAction{Value: tr, Ty: cty.CoinsActionTransfer}),
		Fee:     1000000, Expire: 2, To: address.PubKeyToAddress(privKeyecdsa.PubKey().Bytes()).String()}

	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	prev := types.GInt("MinFee")
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	tx16.Sign(ct.AUTH_ECDSA, privKeyecdsa)
	if !tx16.CheckSign() {
		t.Error("check signature failed")
		return
	}
}

/**
TestCase 06 证书检验
*/
func TestValidateCert(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}

	prev := types.GInt("MinFee")
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	for _, tx := range txs {
		err = authority.Author.Validate(tx.Signature)
		if err != nil {
			t.Error("error cert validate", err.Error())
			return
		}
	}
}

/**
Testcase07 noneimpl校验器验证（回滚到未开启证书验证的区块使用）
*/
func TestValidateTxWithNoneAuth(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	noneCertdata := &types.HistoryCertStore{}
	noneCertdata.CurHeigth = 0
	authority.Author.ReloadCert(noneCertdata)

	prev := types.GInt("MinFee")
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	err = authority.Author.Validate(tx14.Signature)
	if err != nil {
		t.Error("error cert validate", err.Error())
		return
	}
}

/**
Testcase08 重载历史证书
*/
func TestReloadCert(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}

	prev := types.GInt("MinFee")
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	store := &types.HistoryCertStore{}

	authority.Author.ReloadCert(store)

	err = authority.Author.Validate(tx1.Signature)
	if err != nil {
		t.Error(err.Error())
	}
}

/**
Testcase09 根据高度重载历史证书
*/
func TestReloadByHeight(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	prev := types.GInt("MinFee")
	types.SetMinFee(0)
	defer types.SetMinFee(prev)

	authority.Author.ReloadCertByHeght(30)
	if authority.Author.HistoryCertCache.CurHeight != 30 {
		t.Error("reload by height failed")
	}
}

//FIXME 有并发校验的场景需要考虑竞争，暂时没有并发校验的场景
/*
func TestValidateCerts(t *testing.T) {
	err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
	}

	prev := types.GInt("MinFee")
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
