// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package client 实现系统消息队列事件处理，新增client包避免循环引用
package client

import (
	"context"
	"fmt"
	"github.com/33cn/chain33/common/log/log15"
	"sync"
	"time"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

type msgHandler func(message *queue.Message)
type subInitFunc func(ctx CryptoContext) error

var (
	cryptoCtx    = &CryptoContext{}
	lock         = sync.RWMutex{}
	handlers     map[int64]msgHandler
	subInitFuncs map[string]subInitFunc
	log          = log15.New("module", "crypto.client")
)

// CryptoContext system context for crypto
type CryptoContext struct {
	// CurrBlockHeight current height
	CurrBlockHeight int64
	// CurrBlockTime current block time
	CurrBlockTime int64
	// Client queue client
	Client queue.Client
	// API queue api
	API client.QueueProtocolAPI
	// Ctx context
	Ctx context.Context
}

type module struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// New new module
func New() queue.Module {
	ctx, cancel := context.WithCancel(context.Background())
	return module{
		ctx:    ctx,
		cancel: cancel,
	}
}

// RegisterSubInitFunc register sub module init func
func RegisterSubInitFunc(name string, initFunc subInitFunc) {
	lock.Lock()
	defer lock.Unlock()
	_, exist := subInitFuncs[name]
	if exist {
		panic("duplicate sub module init func, name=" + name)
	}
	subInitFuncs[name] = initFunc
}

// RegisterCryptoHandler register handler
func RegisterCryptoHandler(id int64, handler msgHandler) {
	lock.Lock()
	defer lock.Unlock()
	_, exist := handlers[id]
	if exist {
		panic(fmt.Sprintf("duplicate handler id %d", id))
	}
	handlers[id] = handler
}

func getHandler(id int64) msgHandler {
	lock.RLock()
	defer lock.RUnlock()
	return handlers[id]
}

// SetQueueClient set queue client
func (m module) SetQueueClient(cli queue.Client) {

	crypto.Init(cli.GetConfig().GetModuleConfig().Crypto, cli.GetConfig().GetSubConfig().Crypto)
	api, err := client.New(cli, nil)
	if err != nil {
		panic("crypto SetQueueClient, new client api err:" + err.Error())
	}
	initCryptoCtx(cli, api, m.ctx)
	initSubModule()
	cli.Sub("crypto")
	ticker := time.NewTicker(time.Second)
	go func() {

		for {

			select {
			case msg := <-cli.Recv():
				if msg.Ty == types.EventAddBlock {
					if header, ok := msg.Data.(*types.Header); ok {
						SetCurrentBlock(header.GetHeight(), header.GetBlockTime())
					}
				} else if handler := getHandler(msg.Ty); handler != nil {
					handler(msg)
				}
			case <-ticker.C:
				// 首次启动, 需要主动获取依次区块头信息
				h, err := api.GetLastHeader()
				if err == nil {
					SetCurrentBlock(h.GetHeight(), h.GetBlockTime())
					ticker.Stop()
				}
			case <-m.ctx.Done():
				return
			}

		}
	}()

}

// Close exit routine
func (m module) Close() {
	m.cancel()
}

// Wait wait
func (m module) Wait() {

}

// GetCryptoContext get crypto ctx
func GetCryptoContext() CryptoContext {
	lock.RLock()
	defer lock.RUnlock()
	return *cryptoCtx
}

// init sub module
func initSubModule() {
	lock.RLock()
	defer lock.RUnlock()
	ctx := *cryptoCtx
	for name, initFunc := range subInitFuncs {
		err := initFunc(ctx)
		if err != nil {
			log.Error("init sub module failed", "name", name, "err", err)
		}
	}
}

// init crypto context
func initCryptoCtx(cli queue.Client, api client.QueueProtocolAPI, ctx context.Context) {
	lock.Lock()
	defer lock.Unlock()
	cryptoCtx.Client = cli
	cryptoCtx.API = api
	cryptoCtx.Ctx = ctx
}

// SetQueueAPI set api
func SetQueueAPI(api client.QueueProtocolAPI) {
	lock.Lock()
	defer lock.Unlock()
	cryptoCtx.API = api
}

// SetCurrentBlock set block
func SetCurrentBlock(blockHeight, blockTime int64) {
	lock.Lock()
	defer lock.Unlock()
	cryptoCtx.CurrBlockHeight = blockHeight
	cryptoCtx.CurrBlockTime = blockTime
}
