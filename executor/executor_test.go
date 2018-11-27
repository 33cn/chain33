// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"testing"
	"time"

	"encoding/hex"

	_ "github.com/33cn/chain33/system"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

func init() {
	types.Init("local", nil)
}

func TestExecutorGetTxGroup(t *testing.T) {
	exec := &Executor{}
	execInit(nil)
	var txs []*types.Transaction
	addr2, priv2 := util.Genaddress()
	addr3, priv3 := util.Genaddress()
	addr4, _ := util.Genaddress()
	genkey := util.TestPrivkeyList[0]
	txs = append(txs, util.CreateCoinsTx(genkey, addr2, types.Coin))
	txs = append(txs, util.CreateCoinsTx(priv2, addr3, types.Coin))
	txs = append(txs, util.CreateCoinsTx(priv3, addr4, types.Coin))
	//执行三笔交易: 全部正确
	txgroup, err := types.CreateTxGroup(txs)
	if err != nil {
		t.Error(err)
		return
	}
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, priv2)
	txgroup.SignN(2, types.SECP256K1, priv3)
	txs = txgroup.GetTxs()
	execute := newExecutor(nil, exec, 1, time.Now().Unix(), 1, txs, nil)
	e := execute.loadDriver(txs[0], 0)
	execute.setEnv(e)
	txs2 := e.GetTxs()
	assert.Equal(t, txs2, txgroup.GetTxs())
	for i := 0; i < len(txs); i++ {
		txg, err := e.GetTxGroup(i)
		assert.Nil(t, err)
		assert.Equal(t, txg, txgroup.GetTxs())
	}
	_, err = e.GetTxGroup(len(txs))
	assert.Equal(t, err, types.ErrTxGroupIndex)

	//err tx group list
	txs[0].Header = nil
	execute = newExecutor(nil, exec, 1, time.Now().Unix(), 1, txs, nil)
	e = execute.loadDriver(txs[0], 0)
	execute.setEnv(e)
	_, err = e.GetTxGroup(len(txs) - 1)
	assert.Equal(t, err, types.ErrTxGroupFormat)
}

//gen 1万币需要 2s，主要是签名的花费
func BenchmarkGenRandBlock(b *testing.B) {
	_, key := util.Genaddress()
	for i := 0; i < b.N; i++ {
		util.CreateNoneBlock(key, 10000)
	}
}

func TestLoadDriver(t *testing.T) {
	d, err := drivers.LoadDriver("none", 0)
	if err != nil {
		t.Error(err)
	}

	if d.GetName() != "none" {
		t.Error(d.GetName())
	}
}

func TestKeyAllow(t *testing.T) {
	execInit(nil)
	key := []byte("mavl-coins-bty-exec-1wvmD6RNHzwhY4eN75WnM6JcaAvNQ4nHx:19xXg1WHzti5hzBRTUphkM8YmuX6jJkoAA")
	exec := []byte("retrieve")
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = exec
	if !isAllowKeyWrite(key, exec, &tx12, int64(1)) {
		t.Error("retrieve can modify exec")
	}
}

func TestKeyAllow_evm(t *testing.T) {
	execInit(nil)
	key := []byte("mavl-coins-bty-exec-1GacM93StrZveMrPjXDoz5TxajKa9LM5HG:19EJVYexvSn1kZ6MWiKcW14daXsPpdVDuF")
	exec := []byte("user.evm.0xc79c9113a71c0a4244e20f0780e7c13552f40ee30b05998a38edb08fe617aaa5")
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = exec
	if !isAllowKeyWrite(key, exec, &tx12, int64(1)) {
		t.Error("user.evm.hash can modify exec")
	}
	//assert.Nil(t, t)
}

/*
func TestKeyAllow_evmallow(t *testing.T) {
	execInit(nil)
	key := []byte("mavl-evm-xxx")
	exec := []byte("user.evm.0xc79c9113a71c0a4244e20f0780e7c13552f40ee30b05998a38edb08fe617aaa5")
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = exec
	if !isAllowKeyWrite(key, exec, &tx12, int64(1)) {
		t.Error("user.evm.hash can modify exec")
	}
	//assert.Nil(t, t)
}

func TestKeyAllow_paraallow(t *testing.T) {
	execInit(nil)
	key := []byte("mavl-noexec-xxx")
	exec := []byte("user.p.user.noexec.0xc79c9113a71c0a4244e20f0780e7c13552f40ee30b05998a38edb08fe617aaa5")
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = exec
	if isAllowKeyWrite(key, exec, &tx12, int64(1)) {
		t.Error("user.noexec.hash can not modify noexec")
	}
	//assert.Nil(t, t)
}

func TestKeyAllow_ticket(t *testing.T) {
	execInit(nil)
	key := []byte("mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp")
	exec := []byte("ticket")
	tx1 := "0a067469636b657412c701501022c20108dcaed4f1011080a4a7da061a70314556474572784d52343565577532386d6f4151616e6b34413864516635623639383a3078356161303431643363623561356230396131333336626536373539356638366461336233616564386531653733373139346561353135313562653336363933333a3030303030303030303022423078336461373533326364373839613330623037633538343564336537383433613731356630393961616566386533646161376134383765613135383135336331631a6e08011221025a317f60e6962b7ce9836a83b775373b614b290bee595f8aecee5499791831c21a473045022100850bb15cdcdaf465af7ad1ffcbc1fd6a86942a1ddec1dc112164f37297e06d2d02204aca9686fd169462be955cef1914a225726280739770ab1c0d29eb953e54c6b620a08d0630e3faecf8ead9f9e1483a22313668747663424e53454137665a6841644c4a706844775152514a61487079485470"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = exec
	if !isAllowKeyWrite(key, exec, &tx12, int64(1)) {
		t.Error("ticket can modify exec")
	}
}

func TestKeyAllow_paracross(t *testing.T) {
	execInit(nil)
	key := []byte("mavl-coins-bty-exec-1HPkPopVe3ERfvaAgedDtJQ792taZFEHCe:19xXg1WHzti5hzBRTUphkM8YmuX6jJkoAA")
	exec := []byte("paracross")
	tx1 := "0a15757365722e702e746573742e7061726163726f7373124310904e223e1080c2d72f1a1374657374206173736574207472616e736665722222314a524e6a64457170344c4a356671796355426d396179434b536565736b674d4b5220a08d0630f7cba7ec9e8f9bac163a2231367a734d68376d764e444b50473645394e5672506877367a4c3933675773547052"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = []byte("user.p.para.paracross")
	if !isAllowKeyWrite(key, exec, &tx12, int64(1)) {
		t.Error("paracross can modify exec")
	}
}
*/

func TestKeyLocalAllow(t *testing.T) {
	err := isAllowLocalKey([]byte("token"), []byte("LODB-token-"))
	assert.Equal(t, err, types.ErrLocalKeyLen)
	err = isAllowLocalKey([]byte("token"), []byte("LODB-token-a"))
	assert.Nil(t, err)
	err = isAllowLocalKey([]byte(""), []byte("LODB--a"))
	assert.Nil(t, err)
	err = isAllowLocalKey([]byte("exec"), []byte("LODB-execaa"))
	assert.Equal(t, err, types.ErrLocalPrefix)
	err = isAllowLocalKey([]byte("exec"), []byte("-exec------aa"))
	assert.Equal(t, err, types.ErrLocalPrefix)
	err = isAllowLocalKey([]byte("paracross"), []byte("LODB-user.p.para.paracross-xxxx"))
	assert.Equal(t, err, types.ErrLocalPrefix)
}
