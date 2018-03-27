package types

import (
	"time"
)

var chainBaseParam *ChainParam
var chainV3Param *ChainParam

func init() {
	chainBaseParam = &ChainParam{}
	chainBaseParam.CoinReward = 18 * Coin  //用户回报
	chainBaseParam.CoinDevFund = 12 * Coin //发展基金回报
	chainBaseParam.TicketPrice = 10000 * Coin
	chainBaseParam.PowLimitBits = uint32(0x1f00ffff)

	chainBaseParam.RetargetAdjustmentFactor = 4

	chainBaseParam.FutureBlockTime = 16
	chainBaseParam.TicketFrozenTime = 5    //5s only for test
	chainBaseParam.TicketWithdrawTime = 10 //10s only for test
	chainBaseParam.TicketMinerWaitTime = 2 // 2s only for test
	chainBaseParam.MaxTxNumber = 1600      //160
	chainBaseParam.TargetTimespan = 144 * 16 * time.Second
	chainBaseParam.TargetTimePerBlock = 16 * time.Second

	chainV3Param = &ChainParam{}
	tmp := *chainBaseParam
	//copy base param
	chainV3Param = &tmp
	//修改的值
	chainV3Param.FutureBlockTime = 15
	chainV3Param.TicketFrozenTime = 12 * 3600
	chainV3Param.TicketWithdrawTime = 48 * 3600
	chainV3Param.TicketMinerWaitTime = 2 * 3600
	chainV3Param.MaxTxNumber = 1500
	chainV3Param.TargetTimespan = 144 * 15 * time.Second
	chainV3Param.TargetTimePerBlock = 15 * time.Second
}

type ChainParam struct {
	CoinDevFund              int64
	CoinReward               int64
	FutureBlockTime          int64
	TicketPrice              int64
	TicketFrozenTime         int64
	TicketWithdrawTime       int64
	TicketMinerWaitTime      int64
	MaxTxNumber              int64
	PowLimitBits             uint32
	TargetTimespan           time.Duration
	TargetTimePerBlock       time.Duration
	RetargetAdjustmentFactor int64
}

func GetP(height int64) *ChainParam {
	if height < ForkV3 {
		return chainBaseParam
	}
	return chainV3Param
}
