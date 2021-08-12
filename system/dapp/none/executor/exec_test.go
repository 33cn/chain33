// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package executor

import (
	"testing"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	nty "github.com/33cn/chain33/system/dapp/none/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/require"
)

var (
	testCfg = types.NewChain33Config(types.GetDefaultCfgstring())
)

func init() {
	Init(driverName, testCfg, nil)
}

func initTestNone() (queue.Queue, string, *None, *types.Chain33Config) {
	q := queue.New("testnone")
	q.SetConfig(testCfg)
	api, _ := client.New(q.Client(), nil)
	dbDir, stateDB, _ := util.CreateTestDB()
	none := newNone()
	none.SetAPI(api)
	none.SetStateDB(stateDB)
	return q, dbDir, none.(*None), testCfg
}

func TestNone_CheckTx(t *testing.T) {

	_, dbDir, n, cfg := initTestNone()
	defer util.CloseTestDB(dbDir, n.GetStateDB().(db.DB))
	addr, priv := util.Genaddress()
	tx := util.CreateNoneTx(cfg, priv)
	tx.To = addr

	err := n.CheckTx(tx, 0)
	require.Equal(t, types.ErrToAddrNotSameToExecAddr, err)
	tx.To = address.ExecAddress(driverName)
	err = n.CheckTx(tx, 0)
	require.Nil(t, err)

	action := &nty.NoneAction{Ty: nty.TyCommitDelayTxAction}
	tx.Payload = types.Encode(action)
	err = n.CheckTx(tx, 0)
	require.Equal(t, errNilDelayTx, err)
	delayTx := util.CreateNoneTx(cfg, priv)
	commit := &nty.CommitDelayTx{RelativeDelayHeight: -1, DelayTx: common.ToHex(types.Encode(delayTx))}
	action.Value = &nty.NoneAction_CommitDelayTx{
		CommitDelayTx: commit,
	}
	tx.Payload = types.Encode(action)
	err = n.CheckTx(tx, 0)
	require.Equal(t, errNegativeDelayTime, err)

	commit.RelativeDelayHeight = 1
	tx.Payload = types.Encode(action)
	err = n.CheckTx(tx, 0)
	require.Nil(t, err)
	_ = n.GetStateDB().Set(formatDelayTxKey(delayTx.Hash()), []byte("testval"))
	err = n.CheckTx(tx, 0)
	require.Equal(t, errDuplicateDelayTx, err)
}

func TestNone_Exec_CommitDelayTx(t *testing.T) {

	_, dbDir, n, cfg := initTestNone()
	n.SetEnv(1, types.Now().Unix(), 10)
	defer util.CloseTestDB(dbDir, n.GetStateDB().(db.DB))
	addr, priv := util.Genaddress()
	delayTx := util.CreateNoneTx(cfg, priv)
	commit := &nty.CommitDelayTx{DelayTx: common.ToHex(types.Encode(delayTx)), RelativeDelayHeight: 10}
	noneType := types.LoadExecutorType(driverName)
	tx, err := noneType.CreateTransaction(nty.NameCommitDelayTxAction, commit)
	require.Nil(t, err)
	tx, err = types.FormatTx(cfg, driverName, tx)
	require.Nil(t, err)
	tx.Sign(int32(types.SECP256K1), priv)

	recp, err := n.Exec(tx, 0)
	require.Nil(t, err)
	require.True(t, types.ExecOk == recp.Ty)
	require.True(t, 1 == len(recp.Logs))
	require.True(t, nty.TyCommitDelayTxLog == recp.Logs[0].Ty)
	info := &nty.CommitDelayTxLog{}
	err = types.Decode(recp.Logs[0].Log, info)
	require.Nil(t, err)
	require.Equal(t, common.ToHex(delayTx.Hash()), info.DelayTxHash)
	require.Equal(t, addr, info.Submitter)
	require.True(t, 1 == info.DelayBeginHeight)
}

func TestNone_ExecLocal_CommitDelayTx(t *testing.T) {
	n := newNone().(*None)
	dbSet, err := n.ExecLocal_CommitDelayTx(nil, nil, nil, 0)
	require.Nil(t, dbSet)
	require.Nil(t, err)
}

func TestNone_ExecDelLocal(t *testing.T) {
	n := newNone().(*None)
	dbSet, err := n.ExecDelLocal(nil, nil, 0)
	require.Nil(t, dbSet)
	require.Nil(t, err)
}

func TestNone_Query_GetDelayBeginTime(t *testing.T) {

	_, dbDir, n, cfg := initTestNone()
	n.SetEnv(1, types.Now().Unix(), 10)
	defer util.CloseTestDB(dbDir, n.GetStateDB().(db.DB))
	_, priv := util.Genaddress()
	delayTx := util.CreateNoneTx(cfg, priv)
	commit := &nty.CommitDelayTx{DelayTx: common.ToHex(types.Encode(delayTx)), RelativeDelayHeight: 10}
	noneType := types.LoadExecutorType(driverName)
	tx, err := noneType.CreateTransaction(nty.NameCommitDelayTxAction, commit)
	require.Nil(t, err)
	tx, err = types.FormatTx(cfg, driverName, tx)
	require.Nil(t, err)
	recp, err := n.Exec(tx, 0)
	require.Nil(t, err)
	util.SaveKVList(n.GetStateDB().(db.DB), recp.KV)
	msg, err := n.Query(nty.QueryGetDelayBegin, types.Encode(&types.ReqBytes{Data: delayTx.Hash()}))
	require.Nil(t, err)
	reply, ok := msg.(*types.Int64)
	require.True(t, ok)
	require.True(t, 1 == reply.Data)
}
