// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dht

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	clientMocks "github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/queue"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// 本文件钉住 dht 节点身份的取舍:
// addrbook 里已经持久化身份(peer id 已对外发布)时, 不得在运行期用钱包空投地址替换它;
// 只有节点还没有身份时, 才用钱包派生身份。

func newAirDropIdentityP2P(t *testing.T, book *AddrBook, walletPrivHex string, waitPid bool) (*P2P, *clientMocks.QueueProtocolAPI) {
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
	return p, api
}

func testP2PCfg(t *testing.T) *types.P2P {
	return &types.P2P{Driver: "leveldb", DbCache: 4, DbPath: t.TempDir()}
}

func testWalletPrivHex(t *testing.T) string {
	priv, _, err := GenPrivPubkey()
	require.Nil(t, err)
	return hex.EncodeToString(priv)
}

// TestGenAirDropKeyKeepsEstablishedIdentity 身份已建立时不得替换, 也不触发 host 重启:
// 运行期更换 peer id 会让所有按该 id 配置了本节点的对端(peering 列表、TSS peers、白名单)失效。
// 同时断言 “把已有身份导入钱包做备份” 仍然发生 —— 守卫必须放在导入之后。
func TestGenAirDropKeyKeepsEstablishedIdentity(t *testing.T) {
	book := testAddrbook(t, testP2PCfg(t))
	savedPriv, _ := book.GetPrivPubKey()
	require.NotEmpty(t, savedPriv)

	walletPrivHex := testWalletPrivHex(t)
	require.NotEqual(t, savedPriv, walletPrivHex)

	p, api := newAirDropIdentityP2P(t, book, walletPrivHex, false)
	require.False(t, p.doGenAirDropKey(), "身份已建立时不应请求重启 p2p host")

	nowPriv, _ := book.GetPrivPubKey()
	assert.Equal(t, savedPriv, nowPriv, "addrbook 中的节点身份必须保持不变")
	api.AssertCalled(t, "ExecWalletFunc", "wallet", "WalletImportPrivkey", mock.Anything)
}

// TestGenAirDropKeyKeepsEstablishedIdentityWithWaitPid
// waitPid=true 且 addrbook 里已有另一把私钥时同样不得替换:
// 守卫不能收窄成 `savePrivkey != "" && !WaitPid`(那会让运行期换身份又回来)。
// 该路径下不会调用钱包导入(导入块由 !WaitPid 门控)。
func TestGenAirDropKeyKeepsEstablishedIdentityWithWaitPid(t *testing.T) {
	book := testAddrbook(t, testP2PCfg(t))
	savedPriv, _ := book.GetPrivPubKey()
	require.NotEmpty(t, savedPriv)

	walletPrivHex := testWalletPrivHex(t)
	require.NotEqual(t, savedPriv, walletPrivHex)

	p, api := newAirDropIdentityP2P(t, book, walletPrivHex, true)
	require.False(t, p.doGenAirDropKey(), "已有身份时不应请求重启 p2p host")

	nowPriv, _ := book.GetPrivPubKey()
	assert.Equal(t, savedPriv, nowPriv, "addrbook 中的节点身份必须保持不变")
	api.AssertNotCalled(t, "ExecWalletFunc", "wallet", "WalletImportPrivkey", mock.Anything)
}

// TestGenAirDropKeyAdoptsWalletKeyWithoutIdentity 节点还没有身份时仍采用钱包派生的空投地址;
// waitPid=true 会在 DHT host 启动之前确定身份, 因此不需要运行期替换。
func TestGenAirDropKeyAdoptsWalletKeyWithoutIdentity(t *testing.T) {
	book := NewAddrBook(testP2PCfg(t))
	priv, pub := book.GetPrivPubKey()
	require.Empty(t, priv)
	require.Empty(t, pub)

	walletPrivHex := testWalletPrivHex(t)
	p, api := newAirDropIdentityP2P(t, book, walletPrivHex, true)
	require.True(t, p.doGenAirDropKey(), "没有身份时应采用钱包身份")

	nowPriv, _ := book.GetPrivPubKey()
	assert.Equal(t, walletPrivHex, nowPriv)
	api.AssertNotCalled(t, "ExecWalletFunc", "wallet", "WalletImportPrivkey", mock.Anything)
}

// TestGenAirDropKeySameKeyReturnsEarly 钱包身份与 addrbook 一致时走同密钥提前返回分支
// (注意: 这条用例钉的是该提前返回, 不是本次新增的守卫)。
func TestGenAirDropKeySameKeyReturnsEarly(t *testing.T) {
	book := testAddrbook(t, testP2PCfg(t))
	savedPriv, _ := book.GetPrivPubKey()
	require.NotEmpty(t, savedPriv)

	p, _ := newAirDropIdentityP2P(t, book, savedPriv, false)
	require.False(t, p.doGenAirDropKey())

	nowPriv, _ := book.GetPrivPubKey()
	assert.Equal(t, savedPriv, nowPriv)
}

// TestP2PHostIdentityStableAfterWalletReady 真 host 回归: 钱包就绪后 genAirDropKey 会异步执行,
// 已经对外发布的 peer id 与 addrbook 身份都不得变化。去掉守卫时本用例 FAIL
// (peer id 会在钱包就绪约 1 秒后被钱包空投地址替换), 而既有 Test_p2p 仍会通过。
func TestP2PHostIdentityStableAfterWalletReady(t *testing.T) {
	cfg := types.NewChain33Config(types.ReadFile("../../../cmd/chain33/chain33.test.toml"))
	q := queue.New("channel")
	datadir := util.ResetDatadir(cfg.GetModuleConfig(), "$TEMP/")
	setLibp2pLog("", "")
	cfg.GetModuleConfig().Log.LogFile = ""
	cfg.GetModuleConfig().Address.DefaultDriver = "BTC"
	q.SetConfig(cfg)
	processMsg(q)
	defer func() {
		_ = os.RemoveAll(datadir)
	}()

	// 钱包返回一把与 addrbook 随机身份不同的密钥: 修复前这会把已发布的 peer id 换掉
	qapi.On("ExecWalletFunc", "wallet", "NewAccountByIndex", mock.Anything).
		Return(&types.ReplyString{Data: testWalletPrivHex(t)}, nil)
	qapi.On("ExecWalletFunc", "wallet", "GetWalletStatus", &types.ReqNil{}).
		Return(&types.WalletStatus{IsWalletLock: false, IsHasSeed: true}, nil)
	qapi.On("ExecWalletFunc", "wallet", "WalletImportPrivkey", mock.Anything).
		Return(&types.Reply{IsOk: true}, nil)
	// peer 协议的后台 refreshSelf 会调用它, 与身份无关, 给个宽松返回值即可
	qapi.On("GetFinalizedBlock").Return(&types.SnowChoice{}, nil).Maybe()

	var mcfg p2pty.P2PSubConfig
	types.MustDecode(cfg.GetSubConfig().P2P[p2pty.DHTTypeName], &mcfg)
	jcfg, err := json.Marshal(mcfg)
	require.Nil(t, err)
	cfg.GetSubConfig().P2P[p2pty.DHTTypeName] = jcfg

	p2pNode := NewP2p(cfg, q.Client())
	dhtP2P, ok := p2pNode.(*P2P)
	require.True(t, ok)
	defer dhtP2P.CloseP2P()

	hostID := dhtP2P.host.ID()
	savedPriv, _ := dhtP2P.addrBook.GetPrivPubKey()
	require.NotEmpty(t, savedPriv)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		require.Equal(t, hostID, dhtP2P.host.ID(), "钱包就绪后不得更换已经发布的 peer id")
		nowPriv, _ := dhtP2P.addrBook.GetPrivPubKey()
		require.Equal(t, savedPriv, nowPriv, "addrbook 中的节点身份不得被替换")
	}
}
