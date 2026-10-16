// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dht

import (
	"context"
	"encoding/hex"
	"sync"
	"testing"

	clientMocks "github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// 本文件钉住 dht 节点身份的取舍:
// addrbook 里已经持久化身份(peer id 已对外发布)时, 不得在运行期用钱包空投地址替换它;
// 只有节点还没有身份时, 才用钱包派生身份。

func newAirDropIdentityP2P(t *testing.T, book *AddrBook, walletPrivHex string, waitPid bool) *P2P {
	api := &clientMocks.QueueProtocolAPI{}
	api.On("ExecWalletFunc", "wallet", "GetWalletStatus", mock.Anything).
		Return(&types.WalletStatus{IsHasSeed: true, IsWalletLock: false}, nil)
	api.On("ExecWalletFunc", "wallet", "NewAccountByIndex", mock.Anything).
		Return(&types.ReplyString{Data: walletPrivHex}, nil)
	api.On("ExecWalletFunc", "wallet", "WalletImportPrivkey", mock.Anything).
		Return(&types.Reply{IsOk: true}, nil)

	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	cfg.GetModuleConfig().Address.DefaultDriver = "BTC"

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	p := &P2P{
		chainCfg:  cfg,
		api:       api,
		addrBook:  book,
		taskGroup: &sync.WaitGroup{},
		ctx:       ctx,
		p2pCfg:    &types.P2P{WaitPid: waitPid},
	}
	p.taskGroup.Add(1) // doGenAirDropKey 结束前会 Done 一次
	return p
}

func testP2PCfg(t *testing.T) *types.P2P {
	return &types.P2P{Driver: "leveldb", DbCache: 4, DbPath: t.TempDir()}
}

// TestGenAirDropKeyKeepsEstablishedIdentity 身份已建立时不得替换, 也不触发 host 重启:
// 运行期更换 peer id 会让所有按该 id 配置了本节点的对端(peering 列表、TSS peers、白名单)失效。
func TestGenAirDropKeyKeepsEstablishedIdentity(t *testing.T) {
	book := testAddrbook(t, testP2PCfg(t))
	savedPriv, _ := book.GetPrivPubKey()
	require.NotEmpty(t, savedPriv)

	walletPriv, _, err := GenPrivPubkey()
	require.Nil(t, err)
	require.NotEqual(t, savedPriv, hex.EncodeToString(walletPriv))

	p := newAirDropIdentityP2P(t, book, hex.EncodeToString(walletPriv), false)
	require.False(t, p.doGenAirDropKey(), "身份已建立时不应请求重启 p2p host")

	nowPriv, _ := book.GetPrivPubKey()
	assert.Equal(t, savedPriv, nowPriv, "addrbook 中的节点身份必须保持不变")
}

// TestGenAirDropKeyAdoptsWalletKeyWithoutIdentity 节点还没有身份时仍采用钱包派生的空投地址;
// waitPid=true 会在 DHT host 启动之前确定身份, 因此不需要运行期替换。
func TestGenAirDropKeyAdoptsWalletKeyWithoutIdentity(t *testing.T) {
	book := NewAddrBook(testP2PCfg(t))
	priv, pub := book.GetPrivPubKey()
	require.Empty(t, priv)
	require.Empty(t, pub)

	walletPriv, _, err := GenPrivPubkey()
	require.Nil(t, err)
	walletPrivHex := hex.EncodeToString(walletPriv)

	p := newAirDropIdentityP2P(t, book, walletPrivHex, true)
	require.True(t, p.doGenAirDropKey(), "没有身份时应采用钱包身份")

	nowPriv, _ := book.GetPrivPubKey()
	assert.Equal(t, walletPrivHex, nowPriv)
}

// TestGenAirDropKeySameIdentityNoRestart 钱包身份与 addrbook 一致时保持原状。
func TestGenAirDropKeySameIdentityNoRestart(t *testing.T) {
	book := testAddrbook(t, testP2PCfg(t))
	savedPriv, _ := book.GetPrivPubKey()
	require.NotEmpty(t, savedPriv)

	p := newAirDropIdentityP2P(t, book, savedPriv, false)
	require.False(t, p.doGenAirDropKey())

	nowPriv, _ := book.GetPrivPubKey()
	assert.Equal(t, savedPriv, nowPriv)
}
