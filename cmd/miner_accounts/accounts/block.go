// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package accounts 实现了挖矿监控账户相关的功能
package accounts

import (
	"fmt"
	"os"
	"time"

	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
)

// ts/height -> blockHeader
type chain33 struct {
	lastHeader *rpctypes.Header
	// height -> ts -> header
	Headers   map[int64]*rpctypes.Header
	Height2Ts map[int64]int64
	// new only cache ticket for miner
	accountCache map[int64]*types.Accounts
	Host         string
}

func (b chain33) findBlock(ts int64) (int64, *rpctypes.Header) {
	log.Info("show", "utc", ts, "lastBlockTime", b.lastHeader.BlockTime)
	if ts > b.lastHeader.BlockTime {
		ts = b.lastHeader.BlockTime
		return ts, b.lastHeader
	}
	// 到底怎么样才能快速根据时间找到block
	//  1. 一般情况下， 几秒内会有块， 所以直接根据 ts ， 进行前后搜索
	for delta := int64(1); delta < int64(100); delta++ {

		if _, ok := b.Headers[ts-delta]; ok {
			log.Info("show", "utc", ts, "find", b.Headers[ts-delta])
			return ts - delta, b.Headers[ts-delta]
		}
	}

	return 0, nil
}
func (b chain33) getBalance(addrs []string, exec string, timeNear int64) (*rpctypes.Header, []*rpctypes.Account, error) {
	rpcCli, err := jsonclient.NewJSONClient(b.Host)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, nil, err
	}
	realTs, header := b.findBlock(timeNear)
	log.Info("show", "utc", realTs)
	if realTs == 0 || header == nil {
		return nil, nil, types.ErrNotFound
	}

	acc, err := getBalanceAt(rpcCli, addrs, exec, header.StateHash)
	return header, acc, err
}

func (b chain33) addBlock(h *rpctypes.Header) error {
	b.Headers[h.BlockTime] = h
	b.Height2Ts[h.Height] = h.BlockTime
	if h.Height > b.lastHeader.Height {
		cache.lastHeader = h
	}

	return nil
}

var cache = chain33{
	lastHeader:   &rpctypes.Header{Height: 0},
	Headers:      map[int64]*rpctypes.Header{},
	Height2Ts:    map[int64]int64{},
	accountCache: map[int64]*types.Accounts{},
}

func getLastHeader(cli *jsonclient.JSONClient) (*rpctypes.Header, error) {
	method := "Chain33.GetLastHeader"
	var res rpctypes.Header
	err := cli.Call(method, nil, &res)
	return &res, err
}

func getHeaders(cli *jsonclient.JSONClient, start, end int64) (*rpctypes.Headers, error) {
	method := "Chain33.GetHeaders"
	params := &types.ReqBlocks{Start: start, End: end, IsDetail: false}
	var res rpctypes.Headers
	err := cli.Call(method, params, &res)
	return &res, err
}

func getBalanceAt(cli *jsonclient.JSONClient, addrs []string, exec, stateHash string) ([]*rpctypes.Account, error) {
	method := "Chain33.GetBalance"
	params := &types.ReqBalance{Addresses: addrs, Execer: exec, StateHash: stateHash}
	var res []*rpctypes.Account
	err := cli.Call(method, params, &res)
	return res, err
}

func syncHeaders(host string) {
	rpcCli, err := jsonclient.NewJSONClient(host)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	last, err := getLastHeader(rpcCli)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	curHeight := cache.lastHeader.Height
	lastHeight := last.Height

	// 节省监控的内存， 15000的区块头大约10M
	if lastHeight-15000 > curHeight { //} && false {
		curHeight = lastHeight - 15000
	}

	for curHeight < lastHeight {
		hs, err := getHeaders(rpcCli, curHeight, curHeight+100)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		for _, h := range hs.Items {
			if h.Height > cache.lastHeader.Height {
				curHeight = h.Height
			}
			cache.addBlock(h)
		}
		fmt.Fprintln(os.Stderr, err, cache.lastHeader.Height)
	}

	fmt.Fprintln(os.Stderr, err, cache.lastHeader.Height)
}

//SyncBlock 同步区块
func SyncBlock(host string) {
	cache.Host = host
	syncHeaders(host)

	timeout := time.NewTicker(time.Minute)
	for {
		<-timeout.C
		syncHeaders(host)
	}
}
