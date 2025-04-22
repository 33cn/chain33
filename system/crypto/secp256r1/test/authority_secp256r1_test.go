// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/33cn/chain33/system/crypto/secp256r1"

	"github.com/33cn/chain33/system/crypto/common/authority"
	"github.com/33cn/chain33/system/crypto/common/authority/utils"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"

	_ "github.com/33cn/chain33/system"
)

var (
	transfer = &cty.CoinsAction{Value: nil, Ty: cty.CoinsActionTransfer}
	to       = address.PubKeyToAddr(address.DefaultID, privKey.PubKey().Bytes())
	tx1      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 2, To: to}
	tx2      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: to}
	tx3      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 200000000, Expire: 0, To: to}
	tx4      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 300000000, Expire: 0, To: to}
	tx5      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 400000000, Expire: 0, To: to}
	tx6      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 500000000, Expire: 0, To: to}
	tx7      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 600000000, Expire: 0, To: to}
	tx8      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 700000000, Expire: 0, To: to}
	tx9      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 800000000, Expire: 0, To: to}
	tx10     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 900000000, Expire: 0, To: to}
	tx11     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 450000000, Expire: 0, To: to}
	tx12     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0, To: to}
	tx13     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100, Expire: 0, To: to}
	txs      = []*types.Transaction{tx1, tx2, tx3, tx4, tx5, tx6, tx7, tx8, tx9, tx10, tx11, tx12}

	privRaw, _  = common.FromHex("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	tr          = &cty.CoinsAction_Transfer{Transfer: &types.AssetsTransfer{Amount: int64(1e8)}}
	secpp256, _ = crypto.Load(types.GetSignName("", types.SECP256K1), -1)
	privKey, _  = secpp256.PrivKeyFromBytes(privRaw)
	tx14        = &types.Transaction{
		Execer:  []byte("coins"),
		Payload: types.Encode(&cty.CoinsAction{Value: tr, Ty: cty.CoinsActionTransfer}),
		Fee:     1000000,
		Expire:  2,
		To:      address.PubKeyToAddr(address.DefaultID, privKey.PubKey().Bytes()),
	}
)

var USERNAME = "user1"
var ORGNAME = "org1"

func signtx(tx *types.Transaction, priv crypto.PrivKey, cert []byte) {
	tx.Sign(int32(secp256r1.ID), priv)
	tx.Signature.Signature = utils.EncodeCertToSignature(tx.Signature.Signature, cert, nil)
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

// User 用户关联的证书私钥信息
type User struct {
	ID   string
	Cert []byte
	Key  crypto.PrivKey
}

// UserLoader SKD加载user使用
type UserLoader struct {
	configPath string
	userMap    map[string]*User
	signType   int
}

// Init userloader初始化
func (loader *UserLoader) Init(configPath string, signType string) error {
	loader.configPath = configPath
	loader.userMap = make(map[string]*User)

	sign := types.GetSignType("cert", signType)
	if sign == types.Invalid {
		return types.ErrInvalidParam
	}
	loader.signType = sign

	return loader.loadUsers()
}

func (loader *UserLoader) loadUsers() error {
	var priv crypto.PrivKey
	keyDir := path.Join(loader.configPath, "keystore")
	dir, err := ioutil.ReadDir(keyDir)
	if err != nil {
		return err
	}
	for _, file := range dir {
		filePath := path.Join(keyDir, file.Name())
		keyBytes, err := utils.ReadFile(filePath)
		if err != nil {
			continue
		}
		privHex, _ := common.FromHex(string(keyBytes))
		priv, err = loader.genCryptoPriv(privHex)
		if err != nil {
			continue
		}
	}

	certDir := path.Join(loader.configPath, "signcerts")
	dir, err = ioutil.ReadDir(certDir)
	if err != nil {
		return err
	}
	for _, file := range dir {
		filePath := path.Join(certDir, file.Name())
		certBytes, err := utils.ReadFile(filePath)
		if err != nil {
			continue
		}
		loader.userMap[file.Name()] = &User{file.Name(), certBytes, priv}
	}

	return nil
}

func (loader *UserLoader) genCryptoPriv(keyBytes []byte) (crypto.PrivKey, error) {
	cr, err := crypto.Load(types.GetSignName("cert", loader.signType), -1)
	if err != nil {
		return nil, fmt.Errorf("create crypto %s failed, error:%s", types.GetSignName("cert", loader.signType), err)
	}

	priv, err := cr.PrivKeyFromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("get private key failed, error:%s", err)
	}

	return priv, nil
}

// Get 根据用户名获取user结构
func (loader *UserLoader) Get(userName, orgName string) (*User, error) {
	keyvalue := fmt.Sprintf("%s@%s-cert.pem", userName, orgName)
	user, ok := loader.userMap[keyvalue]
	if !ok {
		return nil, types.ErrInvalidParam
	}

	resp := &User{}
	resp.Cert = append(resp.Cert, user.Cert...)
	resp.Key = user.Key

	return resp, nil
}

/*
*
初始化Author实例和userloader
*/
func initEnv() (*types.Chain33Config, error) {
	cfg := types.NewChain33Config(types.ReadFile("./chain33.auth.test.toml"))
	sub := cfg.GetSubConfig().Crypto[secp256r1.Name]
	var subcfg authority.SubConfig
	if sub != nil {
		utils.MustDecode(sub, &subcfg)
	}
	secp256r1.EcdsaAuthor.Init(&subcfg, secp256r1.ID, secp256r1.NewEcdsaValidator())

	userLoader := &UserLoader{}
	err := userLoader.Init(subcfg.CertPath, secp256r1.Name)
	if err != nil {
		fmt.Printf("Init user loader falied -> %v", err)
		return nil, err
	}

	user, err := userLoader.Get(USERNAME, ORGNAME)
	if err != nil {
		fmt.Printf("Get user failed")
		return nil, err
	}

	signtxs(user.Key, user.Cert)
	if err != nil {
		fmt.Printf("Init authority failed")
		return nil, err
	}

	return cfg, nil
}

/*
*
TestCase01 带证书的交易验签
*/
func TestChckSign(t *testing.T) {
	cfg, err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	cfg.SetMinFee(0)

	assert.Equal(t, true, tx1.CheckSign(0))
}

/*
*
TestCase02 带证书的多交易验签
*/
func TestChckSigns(t *testing.T) {
	cfg, err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	cfg.SetMinFee(0)

	for i, tx := range txs {
		if !tx.CheckSign(0) {
			t.Error(fmt.Sprintf("error check tx[%d]", i+1))
			return
		}
	}
}

/*
*
TestCase03 带证书的交易并行验签
*/
func TestChckSignsPara(t *testing.T) {
	cfg, err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	cfg.SetMinFee(0)

	block := types.Block{}
	block.Txs = txs
	if !block.CheckSign(cfg) {
		t.Error("error check txs")
		return
	}
}

/*
*
TestCase04 不带证书，公链签名算法验证
*/
func TestChckSignWithNoneAuth(t *testing.T) {
	cfg, err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	cfg.SetMinFee(0)

	tx14.Sign(types.SECP256K1, privKey)
	if !tx14.CheckSign(0) {
		t.Error("check signature failed")
		return
	}
}

/*
*
TestCase05 不带证书，secp256r1签名验证
*/
func TestChckSignWithEcdsa(t *testing.T) {
	ecdsacrypto, _ := crypto.Load(types.GetSignName("cert", secp256r1.ID), -1)
	privKeyecdsa, _ := ecdsacrypto.PrivKeyFromBytes(privRaw)
	tx16 := &types.Transaction{Execer: []byte("coins"),
		Payload: types.Encode(&cty.CoinsAction{Value: tr, Ty: cty.CoinsActionTransfer}),
		Fee:     1000000, Expire: 2, To: address.PubKeyToAddr(address.DefaultID, privKey.PubKey().Bytes())}

	cfg, err := initEnv()
	if err != nil {
		t.Errorf("init env failed, error:%s", err)
		return
	}
	cfg.SetMinFee(0)

	tx16.Sign(secp256r1.ID, privKeyecdsa)
	assert.Equal(t, false, tx16.CheckSign(0))
}
