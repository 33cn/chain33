package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigFlat(t *testing.T) {
	conf := map[string]interface{}{
		"key1": "value1",
		"key2": map[string]interface{}{
			"key21": "value21",
		},
	}
	flat := FlatConfig(conf)
	assert.Equal(t, "value1", flat["key1"])
	assert.Equal(t, "value21", flat["key2.key21"])
}

func TestConfigMverInit(t *testing.T) {
	cfg, _ := InitCfg("testdata/local.mvertest.toml")
	Init(cfg.Title, cfg)
	assert.Equal(t, MGStr("mver.consensus.name2", 0), "ticket-bityuan")
	assert.Equal(t, MGStr("mver.consensus.name2", 10), "ticket-bityuanv5")
	assert.Equal(t, MGStr("mver.hello", 0), "world")
	assert.Equal(t, MGStr("mver.hello", 11), "forkworld")
	assert.Equal(t, MGStr("mver.nofork", 0), "nofork")
	assert.Equal(t, MGStr("mver.nofork", 9), "nofork")
	assert.Equal(t, MGStr("mver.nofork", 11), "nofork")

	assert.Equal(t, MGStr("mver.exec.sub.token.name2", -1), "ticket-bityuan")
	assert.Equal(t, MGStr("mver.exec.sub.token.name2", 0), "ticket-bityuanv5-enable")
	assert.Equal(t, MGStr("mver.exec.sub.token.name2", 9), "ticket-bityuanv5-enable")
	assert.Equal(t, MGStr("mver.exec.sub.token.name2", 10), "ticket-bityuanv5")
	assert.Equal(t, MGStr("mver.exec.sub.token.name2", 11), "ticket-bityuanv5")
}

var chainBaseParam *ChainParam
var chainV3Param *ChainParam

func initChainBase() {
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
}

func initChainTestNet() {
	chainV3Param = &ChainParam{}
	tmp := *chainBaseParam
	//copy base param
	chainV3Param = &tmp
	chainV3Param.MaxTxNumber = 10000
	chainV3Param.TicketFrozenTime = 5                   //5s only for test
	chainV3Param.TicketWithdrawTime = 10                //10s only for test
	chainV3Param.TicketMinerWaitTime = 2                // 2s only for test
	chainV3Param.TargetTimespan = 144 * 2 * time.Second //only for test
	chainV3Param.TargetTimePerBlock = 2 * time.Second   //only for test
	//chainV3Param.PowLimitBits = uint32(0x1f2fffff)
}

func getP(height int64) *ChainParam {
	initChainBase()
	initChainTestNet()
	if IsFork(height, "ticket.ForkChainParamV1") {
		return chainV3Param
	}
	return chainBaseParam
}

func TestInitChainParam(t *testing.T) {
	cfg, _ := InitCfg("../cmd/chain33/chain33.toml")
	Init(cfg.Title, cfg)
	forkid := GetFork("ticket.ForkChainParamV1")
	assert.Equal(t, GetP(0), getP(0))
	assert.Equal(t, GetP(forkid-1), getP(forkid-1))
	assert.Equal(t, GetP(forkid), getP(forkid))
	assert.Equal(t, GetP(forkid+1), getP(forkid+1))
	assert.Equal(t, GetFundAddr(), "1BQXS6TxaYYG5mADaWij4AxhZZUTpw95a5")
}
