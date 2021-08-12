// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package client 实现系统消息队列事件处理，新增client包避免循环引用
package client

import (
	"sync"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var (
	ctx  = &CryptoContext{}
	lock = sync.RWMutex{}
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

// SetQueueClient set queue client
func (m module) SetQueueClient(cli queue.Client) {

	crypto.Init(cli.GetConfig().GetModuleConfig().Crypto, cli.GetConfig().GetSubConfig().Crypto)
	api, err := client.New(cli, nil)
	if err != nil {
		panic("crypto SetQueueClient, new client api err:" + err.Error())
	}
	SetQueueAPI(api)
	cli.Sub("crypto")
	go func() {

		for {

			select {
			case msg := <-cli.Recv():
				if msg.Ty == types.EventAddBlock {
					if header, ok := msg.Data.(types.Header); ok {
						SetCurrentBlock(header.GetHeight(), header.GetBlockTime())
					}

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
