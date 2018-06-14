package accounts

import (
	"fmt"
	"os"
	"time"

	chain33rpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"

)

// ts/height -> blockHeader
type chain33 struct {
	lastHeader * chain33rpc.Header
	// height -> ts -> header
	Headers map[int64]* chain33rpc.Header
	Height2Ts map[int64]int64
	// new only cache ticket for miner
	accountCache map[int64]*types.Accounts
}

func (b chain33) findBlock(ts int64) (int64, *chain33rpc.Header) {
	log.Info("show", "utc", ts, "lastBlockTime", b.lastHeader.BlockTime)
	if ts > b.lastHeader.BlockTime {
		ts = b.lastHeader.BlockTime
		return ts, b.lastHeader
	}
	// 到底怎么样才能快速根据时间找到block
	//  1. 一般情况下， 几秒内会有块， 所以直接根据 ts ， 进行前后搜索
	for delta := int64(1); delta < int64(100); delta ++ {

		if _, ok := b.Headers[ts-delta]; ok {
			log.Info("show", "utc", ts, "find", b.Headers[ts-delta])
			return ts-delta, b.Headers[ts-delta]
		}
	}

	return 0, nil
}
func (b chain33) getBalance(addrs []string, exec string, timeNear int64) (int64, []*chain33rpc.Account, error) {
	rpcCli, err := chain33rpc.NewJSONClient("http://127.0.0.1:8801")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return timeNear, nil, err
	}
	realTs, header := b.findBlock(timeNear)
	log.Info("show", "utc", realTs)
	if realTs == 0 || header == nil {
		return timeNear, nil, types.ErrNotFound
	}

	acc, err := getBalanceAt(rpcCli, addrs, exec, header.StateHash)
	return realTs, acc, err
}

func (b chain33) addBlock(h *chain33rpc.Header) error {
	b.Headers[h.BlockTime] = h
	b.Height2Ts[h.Height] = h.BlockTime
	if h.Height > b.lastHeader.Height {
		cache.lastHeader = h
	}
	// TODO check fork
	return nil
}

var cache = chain33 {
	lastHeader:&chain33rpc.Header{Height:0},
	Headers: map[int64]* chain33rpc.Header{},
	Height2Ts: map[int64]int64{},
	accountCache : map[int64]*types.Accounts{},
}

func getLastHeader(cli *chain33rpc.JSONClient) (*chain33rpc.Header, error) {
	method := "Chain33.GetLastHeader"
	var res chain33rpc.Header
	err := cli.Call(method, nil, &res)
	return &res, err
}

func getHeaders(cli *chain33rpc.JSONClient, start, end int64) (*chain33rpc.Headers, error) {
	method := "Chain33.GetHeaders"
	params := &types.ReqBlocks{Start: start, End: end, IsDetail: false}
	var res chain33rpc.Headers
	err := cli.Call(method, params, &res)
	return &res, err
}

func getBalanceAt(cli *chain33rpc.JSONClient, addrs []string, exec , stateHash string) ([]*chain33rpc.Account, error) {
	method := "Chain33.GetBalance"
	params := &types.ReqBalance{Addresses: addrs, Execer: exec, StateHash: stateHash}
	var res []*chain33rpc.Account
	err := cli.Call(method, params, &res)
	return res, err
}

func syncHeaders() {
	rpcCli, err := chain33rpc.NewJSONClient("http://127.0.0.1:8801")
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
	if lastHeight - 15000 > curHeight {
		curHeight = lastHeight - 15000
	}

	for curHeight < lastHeight {
		hs, err := getHeaders(rpcCli, curHeight, curHeight + 100)
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

func SyncBlock() {
	syncHeaders()

	for true {
		timeout := time.NewTicker(time.Minute)
		select {
		case <-timeout.C:
			syncHeaders()
		}
	}
}