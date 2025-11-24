// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package accounts

import (
	"time"

	l "github.com/33cn/chain33/common/log/log15"

	"github.com/33cn/chain33/types"

	//"encoding/json"
	//"io/ioutil"
	"fmt"
	"strconv"

	rpctypes "github.com/33cn/chain33/rpc/types"
)

const secondsPerBlock = 5
const btyPreBlock = 5
const baseInterval = 3600
const maxInterval = 15 * 24 * 3600
const monitorBtyLowLimit = 3 * 1e7

var log = l.New("module", "accounts")

// ShowMinerAccount 挖矿账户
type ShowMinerAccount struct {
	DataDir       string
	Addrs         []string
	CoinPrecision int64
}

// Echo 打印
func (*ShowMinerAccount) Echo(in *string, out *interface{}) error {
	if in == nil {
		return types.ErrInvalidParam
	}
	*out = *in
	return nil
}

// TimeAt time
type TimeAt struct {
	// YYYY-mm-dd-HH
	TimeAt string   `json:"timeAt,omitempty"`
	Addrs  []string `json:"addrs,omitempty"`
}

// Get get
func (show *ShowMinerAccount) Get(in *TimeAt, out *interface{}) error {
	if in == nil {
		log.Error("show", "in", "nil")
		return types.ErrInvalidParam
	}
	addrs := show.Addrs
	if in.Addrs != nil && len(in.Addrs) > 0 {
		addrs = in.Addrs
	}
	log.Info("show", "miners", addrs)

	height, err := toBlockHeight(in.TimeAt)
	if err != nil {
		return err
	}
	log.Info("show", "header", height)

	header, curAcc, err := cache.getBalance(addrs, "ticket", height)
	if err != nil {
		log.Error("show", "getBalance failed", err, "height", height)
		return nil
	}

	totalBty := int64(0)
	for _, acc := range curAcc {
		totalBty += acc.Frozen
	}
	log.Info("show 1st balance", "utc", header.BlockTime, "total", totalBty)

	monitorInterval := calcMoniterInterval(totalBty, show.CoinPrecision)
	log.Info("show", "monitor Interval", monitorInterval)

	lastHourHeader, lastAcc, err := cache.getBalance(addrs, "ticket", header.Height-monitorInterval)
	if err != nil {
		log.Error("show", "getBalance failed", err, "ts", header.BlockTime-monitorInterval)
		return nil
	}
	fmt.Print(curAcc, lastAcc)
	log.Info("show 2nd balance", "utc", *lastHourHeader)

	miner := &MinerAccounts{}
	miner.Seconds = header.BlockTime - lastHourHeader.BlockTime
	miner.Blocks = header.Height - lastHourHeader.Height
	miner.ExpectBlocks = miner.Seconds / secondsPerBlock

	miner = calcIncrease(miner, curAcc, lastAcc, header, show.CoinPrecision)
	*out = &miner

	return nil
}

// 找指定时间最接近的区块， 默认是当前时间
func toBlockHeight(timeAt string) (int64, error) {
	seconds := time.Now().Unix()
	if len(timeAt) != 0 {
		tm, err := time.Parse("2006-01-02-15", timeAt)
		if err != nil {
			log.Error("show", "in.TimeAt Parse", err)
			return 0, types.ErrInvalidParam
		}
		seconds = tm.Unix()
	}
	log.Info("show", "utc-init", seconds)

	realTs, header := cache.findBlock(seconds)
	if realTs == 0 || header == nil {
		log.Error("show", "findBlock", "nil")
		return 0, types.ErrNotFound
	}
	return header.Height, nil
}

// 计算监控区块的范围
// 做对小额帐号限制，不然监控范围过大， 如9000个币需要138天
func calcMoniterInterval(totalBty, coinPrecision int64) int64 {
	monitorInterval := int64(baseInterval)
	if totalBty < monitorBtyLowLimit*coinPrecision && totalBty > 0 {
		monitorInterval = int64(float64(monitorBtyLowLimit*coinPrecision) / float64(totalBty) * float64(baseInterval))
	}
	if monitorInterval > maxInterval {
		monitorInterval = maxInterval
	}
	log.Info("show", "monitor Interval", monitorInterval)
	return monitorInterval / secondsPerBlock
}

func calcIncrease(miner *MinerAccounts, acc1, acc2 []*rpctypes.Account, header *rpctypes.Header, coinPrecision int64) *MinerAccounts {
	type minerAt struct {
		addr    string
		curAcc  *rpctypes.Account
		lastAcc *rpctypes.Account
	}
	miners := map[string]*minerAt{}
	for _, a := range acc1 {
		miners[a.Addr] = &minerAt{a.Addr, a, nil}
	}
	for _, a := range acc2 {
		if _, ok := miners[a.Addr]; ok {
			miners[a.Addr].lastAcc = a
		}
	}

	totalIncrease := int64(0)
	expectTotalIncrease := float64(0)
	totalFrozen := float64(0)
	for _, v := range miners {
		if v.lastAcc != nil && v.curAcc != nil {
			totalFrozen += float64(v.curAcc.Frozen) / float64(coinPrecision)
		}
	}
	ticketTotal := float64(30000 * 10000)
	_, ticketAcc, err := cache.getBalance([]string{"16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"}, "coins", header.Height)
	if err == nil && len(ticketAcc) == 1 {
		ticketTotal = float64(ticketAcc[0].Balance+ticketAcc[0].Frozen) / float64(coinPrecision)
	}
	for k, v := range miners {
		if v.lastAcc != nil && v.curAcc != nil {
			total := v.curAcc.Balance + v.curAcc.Frozen
			increase := total - v.lastAcc.Balance - v.lastAcc.Frozen
			expectIncrease := float64(miner.Blocks) * float64(btyPreBlock) * (float64(v.curAcc.Frozen) / float64(coinPrecision)) / ticketTotal

			m := &MinerAccount{
				Addr:           k,
				Total:          types.FormatAmount2FloatDisplay(total, coinPrecision, true),
				Increase:       types.FormatAmount2FloatDisplay(increase, coinPrecision, true),
				Frozen:         types.FormatAmount2FloatDisplay(v.curAcc.Frozen, coinPrecision, true),
				ExpectIncrease: strconv.FormatFloat(expectIncrease, 'f', int(types.ShowPrecisionNum), 64),
			}

			//if m.Addr == "1Lw6QLShKVbKM6QvMaCQwTh5Uhmy4644CG" {
			//	log.Info("acount", "Increase", m.Increase, "expectIncrease", m.ExpectIncrease)
			//	fmt.Println(os.Stderr, "Increase", m.Increase, "expectIncrease", m.ExpectIncrease)
			//}

			// 由于取不到挖矿的交易， 通过预期挖矿数， 推断间隔多少个区块能挖到。
			// 由于挖矿分布的波动， 用双倍的预期能挖到区块的时间间隔来预警
			expectBlocks := (expectIncrease / btyPreBlock)                               // 一个小时预期挖多少个块
			expectMinerInterval := float64(miner.Seconds/secondsPerBlock) / expectBlocks // 预期多少秒可以挖一个块
			moniterInterval := int64(2*expectMinerInterval) + 1

			m.ExpectMinerBlocks = strconv.FormatFloat(expectBlocks, 'f', int(types.ShowPrecisionNum), 64)
			_, acc, err := cache.getBalance([]string{m.Addr}, "ticket", header.Height-moniterInterval)
			if err != nil || len(acc) == 0 {
				m.MinerBtyDuring = "0.0000"
			} else {
				minerDelta := total - acc[0].Balance - acc[0].Frozen
				m.MinerBtyDuring = types.FormatAmount2FloatDisplay(minerDelta, coinPrecision, true)
			}

			miner.MinerAccounts = append(miner.MinerAccounts, m)
			totalIncrease += increase
			expectTotalIncrease += expectIncrease
		}
	}
	miner.TotalIncrease = types.FormatAmount2FloatDisplay(totalIncrease, coinPrecision, true)
	miner.ExpectTotalIncrease = strconv.FormatFloat(expectTotalIncrease, 'f', int(types.ShowPrecisionNum), 64)

	return miner

}
