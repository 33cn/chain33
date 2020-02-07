// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package plugin 执行器插件
package plugin

import (
	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/client/api"
	dbm "github.com/33cn/chain33/common/db"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

var blog = log.New("module", "plugin.base")

var globalPlugins = make(map[string]Plugin)

// RegisterPlugin register plugin
func RegisterPlugin(name string, p Plugin) {
	if _, ok := globalPlugins[name]; ok {
		panic("plugin exist " + name)
	}
	globalPlugins[name] = p
}

// GetPlugin by name
func GetPlugin(name string) (p Plugin, err error) {
	if p, ok := globalPlugins[name]; ok {
		return p, nil
	}
	return nil, types.ErrUnknowPlugin
}

// Plugin defines some interface
type Plugin interface {
	// 执行时需要调用
	CheckEnable(enable bool) (kvs []*types.KeyValue, ok bool, err error)
	ExecLocal(data *types.BlockDetail) ([]*types.KeyValue, error)
	ExecDelLocal(data *types.BlockDetail) ([]*types.KeyValue, error)
	/*
		// 设置执行环境相关
		SetLocalDB(dbm.KVDB)

		GetExecutorAPI() api.ExecutorAPI

		GetName() string
		SetName(string)

		SetAPI(client.QueueProtocolAPI)
		SetEnv(height, blocktime int64, difficulty uint64)
	*/
}

// Base defines plugin base type
type Base struct {
	localdb              dbm.KVDB
	coinsaccount         *account.DB
	height               int64
	blocktime            int64
	parentHash, mainHash []byte
	mainHeight           int64
	name                 string
	curname              string
	child                Plugin
	api                  client.QueueProtocolAPI
	execapi              api.ExecutorAPI
	txs                  []*types.Transaction
	receipts             []*types.ReceiptData
}
