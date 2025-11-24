// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"math/big"
	"testing"
	"time"

	"github.com/33cn/chain33/common/crypto"
	erpctypes "github.com/33cn/chain33/rpc/ethrpc/types"
	drivers "github.com/33cn/chain33/system/dapp"
	ecommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"strings"

	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

func TestLoadDriverFork(t *testing.T) {

	str := types.GetDefaultCfgstring()
	new := strings.Replace(str, "Title=\"local\"", "Title=\"chain33\"", 1)
	exec, _ := initEnv(types.MergeCfg(types.ReadFile("../cmd/chain33/chain33.fork.toml"), new))
	cfg := exec.client.GetConfig()
	execInit(cfg)
	drivers.Register(cfg, "notAllow", newAllowApp, 0)
	var txs []*types.Transaction
	addr, _ := util.Genaddress()
	genkey := util.TestPrivkeyList[0]
	tx := util.CreateCoinsTx(cfg, genkey, addr, types.DefaultCoinPrecision)
	txs = append(txs, tx)
	tx.Execer = []byte("notAllow")
	tx1 := types.CloneTx(tx)
	tx1.Execer = []byte("user.p.para.notAllow")
	tx2 := types.CloneTx(tx)
	tx2.Execer = []byte("user.p.test.notAllow")
	// local fork值 为0, 测试不出fork前的情况
	//types.SetTitleOnlyForTest("chain33")
	t.Log("get fork value", cfg.GetFork("ForkCacheDriver"), cfg.GetTitle())
	cases := []struct {
		height     int64
		driverName string
		tx         *types.Transaction
		index      int
	}{
		{cfg.GetFork("ForkCacheDriver") - 1, "notAllow", tx, 0},
		//相当于平行链交易 使用none执行器
		{cfg.GetFork("ForkCacheDriver") - 1, "none", tx1, 0},
		// index > 0的allow, 但由于是fork之前，所以不会检查allow, 直接采用上一个缓存的，即none执行器
		{cfg.GetFork("ForkCacheDriver") - 1, "none", tx1, 1},
		// 不能通过allow判定 加载none执行器
		{cfg.GetFork("ForkCacheDriver"), "none", tx2, 0},
		// fork之后需要重新判定, index>0 通过allow判定
		{cfg.GetFork("ForkCacheDriver"), "notAllow", tx2, 1},
	}
	ctx := &executorCtx{
		stateHash:  nil,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
		mainHash:   nil,
		parentHash: nil,
	}
	execute := newExecutor(ctx, exec, nil, txs, nil)
	for idx, c := range cases {
		if execute.height != c.height {
			execute.driverCache = make(map[string]drivers.Driver)
			execute.height = c.height
		}
		driver := execute.loadDriver(c.tx, c.index)
		assert.Equal(t, c.driverName, driver.GetDriverName(), "case index=%d", idx)
	}
}

type notAllowApp struct {
	*drivers.DriverBase
}

func newAllowApp() drivers.Driver {
	app := &notAllowApp{DriverBase: &drivers.DriverBase{}}
	app.SetChild(app)
	return app
}

func (app *notAllowApp) GetDriverName() string {
	return "notAllow"
}

func (app *notAllowApp) Allow(tx *types.Transaction, index int) error {

	if app.GetHeight() == 0 {
		return types.ErrActionNotSupport
	}
	// 这里简单模拟 同比交易不同action 的allow限定，大于0 只是配合上述测试用例
	if index > 0 || string(tx.Execer) == "notAllow" {
		return nil
	}
	return types.ErrActionNotSupport
}

func TestProxyExec(t *testing.T) {
	exec, _ := initEnv(types.GetDefaultCfgstring())
	cfg := exec.client.GetConfig()
	ctx := &executorCtx{
		stateHash:  nil,
		height:     0,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
		mainHash:   nil,
		parentHash: nil,
	}

	execute := newExecutor(ctx, exec, nil, nil, nil)
	//测试解析代理地址
	proxyAddr := ecommon.HexToAddress("0x0000000000000000000000000000000000200005")
	var hexKey = "7939624566468cfa3cb2c9f39d5ad83bdc7cf4356bfd1a7b8094abda6b0699d1"
	sk, err := ethcrypto.ToECDSA(ecommon.FromHex(hexKey))
	assert.Nil(t, err)

	signer := ethtypes.NewEIP155Signer(big.NewInt(3999))
	//构建eth交易
	to, _ := util.Genaddress()
	//coins 转账，没有签名的裸交易
	coinsTx := util.CreateCoinsTx(cfg, nil, to, 1000)
	etx := ethtypes.NewTransaction(uint64(0), proxyAddr, big.NewInt(0), 3000000, big.NewInt(10e9), types.Encode(coinsTx))
	signtx, err := ethtypes.SignTx(etx, signer, sk)
	assert.Nil(t, err)
	v, r, s := signtx.RawSignatureValues()
	cv, err := erpctypes.CaculateRealV(v, signtx.ChainId().Uint64(), signtx.Type())
	assert.Nil(t, err)
	sig := make([]byte, 65)
	copy(sig[32-len(r.Bytes()):32], r.Bytes())
	copy(sig[64-len(s.Bytes()):64], s.Bytes())
	sig[64] = cv
	txSha3 := signer.Hash(signtx)
	pubkey, err := ethcrypto.Ecrecover(txSha3.Bytes(), sig)
	assert.Nil(t, err)
	assembleTx := erpctypes.AssembleChain33Tx(signtx, sig, pubkey, cfg)
	//checkSign
	err = execute.checkTx(assembleTx, 0)
	assert.Errorf(t, err, "ErrExecNameNotAllow")
	types.AllowUserExec = append(types.AllowUserExec, []byte("evm"))
	err = execute.checkTx(assembleTx, 0)
	assert.Nil(t, err)

	crypto.Init(cfg.GetModuleConfig().Crypto, cfg.GetSubConfig().Crypto)
	assert.True(t, assembleTx.CheckSign(0))
	//解析交易
	realTx, err := execute.proxyExecTx(assembleTx)
	assert.Nil(t, err)
	assert.Equal(t, "0xa42431da868c58877a627cc71dc95f01bf40c196", realTx.From())
	testTx := realTx.Clone()
	testTx.Signature = nil

	//与原始交易对比
	assert.Equal(t, coinsTx.Hash(), testTx.Hash())
}
