// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package common 实现钱包基础功能包
package common

import (
	"math/rand"
	"reflect"
	"sync"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

var (
	// QueryData 查询接口数据全局对象
	QueryData = types.NewQueryData("On_")
	// PolicyContainer 钱包业务容器
	PolicyContainer = make(map[string]WalletBizPolicy)
)

// Init 初始化所有已经注册的钱包业务
func Init(wallet WalletOperate, sub map[string][]byte) {
	for k, v := range PolicyContainer {
		v.Init(wallet, sub[k])
	}
}

// RegisterPolicy 注册钱包业务策略接口
func RegisterPolicy(key string, policy WalletBizPolicy) {
	if _, existed := PolicyContainer[key]; existed {
		panic("RegisterPolicy dup")
	}
	PolicyContainer[key] = policy
	QueryData.Register(key, policy)
	QueryData.SetThis(key, reflect.ValueOf(policy))
}

// WalletOperate 钱包对业务插件提供服务的操作接口
type WalletOperate interface {
	RegisterMineStatusReporter(reporter MineStatusReport) error

	GetAPI() client.QueueProtocolAPI
	GetMutex() *sync.Mutex
	GetDBStore() db.DB
	GetSignType() int
	GetPassword() string
	GetBlockHeight() int64
	GetRandom() *rand.Rand
	GetWalletDone() chan struct{}
	GetLastHeader() *types.Header
	GetTxDetailByHashs(ReqHashes *types.ReqHashes)
	GetWaitGroup() *sync.WaitGroup
	GetAllPrivKeys() ([]crypto.PrivKey, error)
	GetWalletAccounts() ([]*types.WalletAccountStore, error)
	GetPrivKeyByAddr(addr string) (crypto.PrivKey, error)
	GetConfig() *types.Wallet
	GetBalance(addr string, execer string) (*types.Account, error)

	IsWalletLocked() bool
	IsClose() bool
	IsCaughtUp() bool
	AddrInWallet(addr string) bool

	CheckWalletStatus() (bool, error)
	Nonce() int64

	WaitTx(hash []byte) *types.TransactionDetail
	WaitTxs(hashes [][]byte) (ret []*types.TransactionDetail)
	SendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error)
	SendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error)
}
