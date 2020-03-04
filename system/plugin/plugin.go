// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package plugin 执行器插件
package plugin

import (
	"fmt"

	"github.com/33cn/chain33/client"
	dbm "github.com/33cn/chain33/common/db"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

var blog = log.New("module", "plugin.base")

var globalPlugins = make(map[string]func() Plugin)

// queryName -> pluginName -> call plugin.Query(...)
var queryFuns = make(map[string]string)

// Prefix 插件前缀
func Prefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-", types.LocalPluginPrefix, name))
}

// RegisterQuery Register Query function
func RegisterQuery(query, plugin string) {
	if _, ok := queryFuns[query]; ok {
		panic("query function exist " + query)
	}
	queryFuns[query] = plugin
}

// QueryPlugin by func name
func QueryPlugin(queryFun string) (p Plugin, err error) {
	if name, ok := queryFuns[queryFun]; ok {
		return GetPlugin(name)
	}
	return nil, types.ErrUnknowPlugin
}

// RegisterPlugin register plugin
func RegisterPlugin(name string, p func() Plugin) {
	if _, ok := globalPlugins[name]; ok {
		panic("plugin exist " + name)
	}
	globalPlugins[name] = p
}

// GetPlugin by name
func GetPlugin(name string) (p Plugin, err error) {
	if p, ok := globalPlugins[name]; ok {
		return p(), nil
	}
	return nil, types.ErrUnknowPlugin
}

// Plugin defines some interface
type Plugin interface {
	// 执行时需要调用
	CheckEnable(enable bool) (kvs []*types.KeyValue, ok bool, err error)
	ExecLocal(data *types.BlockDetail) ([]*types.KeyValue, error)
	ExecDelLocal(data *types.BlockDetail) ([]*types.KeyValue, error)
	// Query
	Query(funcName string, params []byte) (types.Message, error)

	// 数据升级
	// 升级数据过多, 需要多次升级 done=true 为完成, done=false 需要多次调用
	Upgrade(count int32) (done bool, err error)
	// Get/Set name
	GetName() string
	SetName(string)

	// 设置执行环境相关
	GetLocalDB() dbm.KVDB
	SetLocalDB(dbm.KVDB)
	SetAPI(queueapi client.QueueProtocolAPI)
	GetAPI() client.QueueProtocolAPI

	SetEnv(height, blocktime int64, difficulty uint64)
	GetBlockTime() int64
	GetHeight() int64
	GetDifficulty() uint64
}

// Base defines plugin base type
type Base struct {
	name    string
	localdb dbm.KVDB
	api     client.QueueProtocolAPI

	height     int64
	blockTime  int64
	difficulty uint64
}

// CheckEnable CheckEnable
func (b *Base) CheckEnable(enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	return nil, true, nil
}

// ExecLocal ExecLocal
func (b *Base) ExecLocal(data *types.BlockDetail) ([]*types.KeyValue, error) {
	return nil, nil
}

// ExecDelLocal ExecDelLocal
func (b *Base) ExecDelLocal(data *types.BlockDetail) ([]*types.KeyValue, error) {
	return nil, nil
}

// Query query
func (b *Base) Query(funcName string, params []byte) (types.Message, error) {
	return nil, types.ErrQueryNotSupport
}

// SetLocalDB set localdb
func (b *Base) SetLocalDB(l dbm.KVDB) {
	b.localdb = l
}

// GetLocalDB get localdb
func (b *Base) GetLocalDB() dbm.KVDB {
	return b.localdb
}

// GetHeight get height
func (b *Base) GetHeight() int64 {
	return b.height
}

// SetEnv set block env
func (b *Base) SetEnv(h, t int64, d uint64) {
	b.height = h
	b.blockTime = t
	b.difficulty = d
}

// SetAPI set queue protocol api
func (b *Base) SetAPI(queueapi client.QueueProtocolAPI) {
	b.api = queueapi
}

// GetAPI get queue protocol api
func (b *Base) GetAPI() client.QueueProtocolAPI {
	return b.api
}

// GetBlockTime get block time
func (b *Base) GetBlockTime() int64 {
	return b.blockTime
}

// GetDifficulty get Difficulty
func (b *Base) GetDifficulty() uint64 {
	return b.difficulty
}

// GetName get name
func (b *Base) GetName() string {
	return b.name
}

// SetName set name
func (b *Base) SetName(n string) {
	b.name = n
}

// Upgrade upgrade local data
func (b *Base) Upgrade(count int32) (bool, error) {
	return true, nil
}
