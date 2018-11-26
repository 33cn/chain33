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

const secondsPerBlock = 15
const btyPreBlock = 18
const statInterval = 3600
const monitorBtyLowLimit = 3 * 1e7 * types.Coin

var log = l.New("module", "accounts")

//ShowMinerAccount 挖矿账户
type ShowMinerAccount struct {
	DataDir string
	Addrs   []string
}

//Echo 打印
func (*ShowMinerAccount) Echo(in *string, out *interface{}) error {
	if in == nil {
		return types.ErrInvalidParam
	}
	*out = *in
	return nil
}

//TimeAt time
type TimeAt struct {
	// YYYY-mm-dd-HH
	TimeAt string   `json:"timeAt,omitempty"`
	Addrs  []string `json:"addrs,omitempty"`
}

//Get get
func (show *ShowMinerAccount) Get(in *TimeAt, out *interface{}) error {
	if in == nil {
		log.Error("show", "in", "nil")
		return types.ErrInvalidParam
	}
	seconds := time.Now().Unix()
	if len(in.TimeAt) != 0 {
		tm, err := time.Parse("2006-01-02-15", in.TimeAt)
		if err != nil {
			log.Error("show", "in.TimeAt Parse", err)
			return types.ErrInvalidParam
		}
		seconds = tm.Unix()
	}
	log.Info("show", "utc", seconds)

	addrs := show.Addrs
	if in.Addrs != nil && len(in.Addrs) > 0 {
		addrs = in.Addrs
	}
	log.Info("show", "miners", addrs)
	//for i := int64(0); i < 200; i++ {
	header, curAcc, err := cache.getBalance(addrs, "ticket", seconds)
	if err != nil {
		log.Error("show", "getBalance failed", err, "ts", seconds)
		return nil
	}

	totalBty := int64(0)
	for _, acc := range curAcc {
		totalBty += acc.Frozen
	}

	monitorInterval := int64(statInterval)
	if totalBty < monitorBtyLowLimit && totalBty > 0 {
		monitorInterval = int64(float64(statInterval) * float64(monitorBtyLowLimit) / float64(totalBty))
	}
	log.Info("show", "monitor Interval", monitorInterval)
	lastHourHeader, lastAcc, err := cache.getBalance(addrs, "ticket", header.BlockTime-monitorInterval)
	if err != nil {
		log.Error("show", "getBalance failed", err, "ts", header.BlockTime-monitorInterval)
		return nil
	}
	fmt.Print(curAcc, lastAcc)

	miner := &MinerAccounts{}
	miner.Seconds = header.BlockTime - lastHourHeader.BlockTime
	miner.Blocks = header.Height - lastHourHeader.Height
	miner.ExpectBlocks = miner.Seconds / secondsPerBlock

	miner = calcIncrease(miner, curAcc, lastAcc, header)
	*out = &miner

	//}

	return nil
}

func calcIncrease(miner *MinerAccounts, acc1, acc2 []*rpctypes.Account, header *rpctypes.Header) *MinerAccounts {
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

	totalIncrease := float64(0)
	expectTotalIncrease := float64(0)
	totalFrozen := float64(0)
	for _, v := range miners {
		if v.lastAcc != nil && v.curAcc != nil {
			totalFrozen += float64(v.curAcc.Frozen) / float64(types.Coin)
		}
	}
	ticketTotal := float64(30000 * 10000)
	_, ticketAcc, err := cache.getBalance([]string{"16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"}, "coins", header.BlockTime)
	if err == nil && len(ticketAcc) == 1 {
		ticketTotal = float64(ticketAcc[0].Balance+ticketAcc[0].Frozen) / float64(types.Coin)
	}
	for k, v := range miners {
		if v.lastAcc != nil && v.curAcc != nil {
			total := v.curAcc.Balance + v.curAcc.Frozen
			increase := total - v.lastAcc.Balance - v.lastAcc.Frozen
			expectIncrease := float64(miner.Blocks) * float64(btyPreBlock) * (float64(v.curAcc.Frozen) / float64(types.Coin)) / ticketTotal

			m := &MinerAccount{
				Addr:           k,
				Total:          strconv.FormatFloat(float64(total)/float64(types.Coin), 'f', 4, 64),
				Increase:       strconv.FormatFloat(float64(increase)/float64(types.Coin), 'f', 4, 64),
				Frozen:         strconv.FormatFloat(float64(v.curAcc.Frozen)/float64(types.Coin), 'f', 4, 64),
				ExpectIncrease: strconv.FormatFloat(expectIncrease, 'f', 4, 64),
			}

			//if m.Addr == "1Lw6QLShKVbKM6QvMaCQwTh5Uhmy4644CG" {
			//	log.Info("acount", "Increase", m.Increase, "expectIncrease", m.ExpectIncrease)
			//	fmt.Println(os.Stderr, "Increase", m.Increase, "expectIncrease", m.ExpectIncrease)
			//}

			// 由于取不到挖矿的交易， 通过预期挖矿数， 推断间隔多少个区块能挖到。
			// 由于挖矿分布的波动， 用双倍的预期能挖到区块的时间间隔来预警
			expectBlocks := (expectIncrease / btyPreBlock)     // 一个小时预期挖多少个块
			expectMinerInterval := statInterval / expectBlocks // 预期多少秒可以挖一个块
			moniterInterval := int64(2*expectMinerInterval) + 1

			m.ExpectMinerBlocks = strconv.FormatFloat(expectBlocks, 'f', 4, 64)
			_, acc, err := cache.getBalance([]string{m.Addr}, "ticket", header.BlockTime-moniterInterval)
			if err != nil || len(acc) == 0 {
				m.MinerBtyDuring = "0.0000"
			} else {
				minerDelta := total - acc[0].Balance - acc[0].Frozen
				m.MinerBtyDuring = strconv.FormatFloat(float64(minerDelta)/float64(types.Coin), 'f', 4, 64)
			}

			miner.MinerAccounts = append(miner.MinerAccounts, m)
			totalIncrease += float64(increase) / float64(types.Coin)
			expectTotalIncrease += expectIncrease
		}
	}
	miner.TotalIncrease = strconv.FormatFloat(totalIncrease, 'f', 4, 64)
	miner.ExpectTotalIncrease = strconv.FormatFloat(expectTotalIncrease, 'f', 4, 64)

	return miner

}

/*
func readJson(jsonFile string) (*Accounts, error) {
	d1, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		log.Error("show", "read json", jsonFile, "err", err)
		return nil, err
	}
	var acc Accounts
	err = json.Unmarshal([]byte(d1), &acc)
	if err != nil {
		log.Error("show", "read json", jsonFile, "err", err)
		return nil, err
	}
	return &acc, nil
}

func parseAccounts(acc *Accounts) (*map[string]float64, error) {
	result := map[string]float64{}
	for _, a := range acc.Accounts {
		f1, e1 := strconv.ParseFloat(a.Balance, 64)
		f2, e2 := strconv.ParseFloat(a.Frozen, 64)
		if e1 != nil || e2 != nil {
			log.Error("show", "account2  len", e1, "account  len", e2)
			return nil, types.ErrNotFound
		}
		result[a.Addr] = f1 + f2
	}
	return &result, nil
}

func getAccountDetail(jsonFile string) (*map[string]float64, error) {
	acc, err := readJson(jsonFile)
	if err != nil {
		return nil, err
	}
	return parseAccounts(acc)
}

*/
