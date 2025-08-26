// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package client 实现系统消息队列事件处理，新增client包避免循环引用
package client

import (
	"fmt"
	"sync"
	"time"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

type msgHandler func(message *queue.Message)

var (
	ctx      = &CryptoContext{}
	lock     = sync.RWMutex{}
	handlers map[int64]msgHandler
)

// CryptoContext system context for crypto
type CryptoContext struct {
	// API queue api
	API             client.QueueProtocolAPI
	CurrBlockHeight int64
	CurrBlockTime   int64
}

type module struct {
	done chan struct{}
}

// New new module
func New() queue.Module {
	return module{
		done: make(chan struct{}),
	}
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
	SetQueueAPI(api)
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
			case <-m.done:
				return
			}

		}
	}()

}

// Close close
func (m module) Close() {
	close(m.done)
}

// Wait wait
func (m module) Wait() {

}

// GetCryptoContext get crypto ctx
func GetCryptoContext() CryptoContext {
	lock.RLock()
	defer lock.RUnlock()
	return *ctx
}

// SetQueueAPI set api
func SetQueueAPI(api client.QueueProtocolAPI) {
	lock.Lock()
	defer lock.Unlock()
	ctx.API = api
}

// SetCurrentBlock set block
func SetCurrentBlock(blockHeight, blockTime int64) {
	lock.Lock()
	defer lock.Unlock()
	ctx.CurrBlockHeight = blockHeight
	ctx.CurrBlockTime = blockTime
}
