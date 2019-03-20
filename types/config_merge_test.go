// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeConfig(t *testing.T) {
	conf := map[string]interface{}{
		"key1": "value1",
	}
	def := map[string]interface{}{
		"key2": "value2",
	}
	errstr := MergeConfig(conf, def)
	assert.Equal(t, errstr, "")
	assert.Equal(t, conf["key1"], "value1")
	assert.Equal(t, conf["key2"], "value2")

	conf = map[string]interface{}{
		"key1": "value1",
	}
	def = map[string]interface{}{
		"key1": "value2",
	}
	errstr = MergeConfig(conf, def)
	assert.Equal(t, errstr, "rewrite defalut key key1\n")
	assert.Equal(t, conf["key1"], "value1")

	//level2
	conf1 := map[string]interface{}{
		"key1": "value1",
	}
	def1 := map[string]interface{}{
		"key2": "value2",
	}
	conf = map[string]interface{}{
		"key1": conf1,
	}
	def = map[string]interface{}{
		"key2": def1,
	}
	errstr = MergeConfig(conf, def)
	assert.Equal(t, errstr, "")
	assert.Equal(t, conf["key1"].(map[string]interface{})["key1"], "value1")
	assert.Equal(t, conf["key2"].(map[string]interface{})["key2"], "value2")
}

func TestMergeLevel2Error(t *testing.T) {
	//level2
	conf1 := map[string]interface{}{
		"key1": "value1",
	}
	def1 := map[string]interface{}{
		"key1": "value2",
	}
	conf := map[string]interface{}{
		"key1": conf1,
	}
	def := map[string]interface{}{
		"key1": def1,
	}
	errstr := MergeConfig(conf, def)
	assert.Equal(t, errstr, "rewrite defalut key key1.key1\n")
	assert.Equal(t, conf["key1"].(map[string]interface{})["key1"], "value1")
}

func TestMergeToml(t *testing.T) {
	S("cfg.bityuan", bityuancfg)
	newcfg := mergeCfg(readFile("../cmd/chain33/bityuan.toml"))
	cfg1, err := initCfgString(newcfg)
	assert.Nil(t, err)
	cfg2, err := initCfgString(readFile("testdata/bityuan.toml"))
	assert.Nil(t, err)
	assert.Equal(t, cfg1, cfg2)
}

var bityuancfg = `
TestNet=false
[blockchain]
defCacheSize=128
maxFetchBlockNum=128
timeoutSeconds=5
batchBlockNum=128
driver="leveldb"
isStrongConsistency=false
singleMode=false

[p2p]
enable=true
serverStart=true
msgCacheSize=10240
driver="leveldb"

[mempool]
poolCacheSize=102400
minTxFee=100000
maxTxFee=1000000000

[consensus]
name="ticket"
minerstart=true
genesisBlockTime=1514533394
genesis="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

[mver.consensus]
fundKeyAddr = "1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
coinReward = 18
coinDevFund = 12
ticketPrice = 10000
powLimitBits = "0x1f00ffff"
retargetAdjustmentFactor = 4
futureBlockTime = 15
ticketFrozenTime = 43200
ticketWithdrawTime = 172800
ticketMinerWaitTime = 7200
maxTxNumber = 1500
targetTimespan = 2160
targetTimePerBlock = 15

[consensus.sub.ticket]
genesisBlockTime=1526486816
[[consensus.sub.ticket.genesis]]
minerAddr="184wj4nsgVxKyz2NhM3Yb5RK5Ap6AFRFq2"
returnAddr="1FB8L3DykVF7Y78bRfUrRcMZwesKue7CyR"
count=3000

[[consensus.sub.ticket.genesis]]
minerAddr="1M4ns1eGHdHak3SNc2UTQB75vnXyJQd91s"
returnAddr="1Lw6QLShKVbKM6QvMaCQwTh5Uhmy4644CG"
count=3000

[[consensus.sub.ticket.genesis]]
minerAddr="19ozyoUGPAQ9spsFiz9CJfnUCFeszpaFuF"
returnAddr="1PSYYfCbtSeT1vJTvSKmQvhz8y6VhtddWi"
count=3000

[[consensus.sub.ticket.genesis]]
minerAddr="1MoEnCDhXZ6Qv5fNDGYoW6MVEBTBK62HP2"
returnAddr="1BG9ZoKtgU5bhKLpcsrncZ6xdzFCgjrZud"
count=3000

[[consensus.sub.ticket.genesis]]
minerAddr="1FjKcxY7vpmMH6iB5kxNYLvJkdkQXddfrp"
returnAddr="1G7s64AgX1ySDcUdSW5vDa8jTYQMnZktCd"
count=3000

[[consensus.sub.ticket.genesis]]
minerAddr="12T8QfKbCRBhQdRfnAfFbUwdnH7TDTm4vx"
returnAddr="1FiDC6XWHLe7fDMhof8wJ3dty24f6aKKjK"
count=3000

[[consensus.sub.ticket.genesis]]
minerAddr="1bgg6HwQretMiVcSWvayPRvVtwjyKfz1J"
returnAddr="1AMvuuQ7V7FPQ4hkvHQdgNWy8wVL4d4hmp"
count=3000

[[consensus.sub.ticket.genesis]]
minerAddr="1EwkKd9iU1pL2ZwmRAC5RrBoqFD1aMrQ2"
returnAddr="1ExRRLoJXa8LzXdNxnJvBkVNZpVw3QWMi4"
count=3000

[[consensus.sub.ticket.genesis]]
minerAddr="1HFUhgxarjC7JLru1FLEY6aJbQvCSL58CB"
returnAddr="1KNGHukhbBnbWWnMYxu1C7YMoCj45Z3amm"
count=3000

[[consensus.sub.ticket.genesis]]
minerAddr="1C9M1RCv2e9b4GThN9ddBgyxAphqMgh5zq"
returnAddr="1AH9HRd4WBJ824h9PP1jYpvRZ4BSA4oN6Y"
count=4733

[store]
name="mavl"
driver="leveldb"

[wallet]
minFee=100000
driver="leveldb"
signType="secp256k1"

[exec]
isFree=false
minExecFee=100000
maxExecFee=1000000000

[exec.sub.token]
#配置一个空值，防止配置文件被覆盖
tokenApprs = []

[exec.sub.relay]
genesis="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

[exec.sub.manage]
superManager=[
    "1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP", 
]

#系统中所有的fork,默认用chain33的测试网络的
#但是我们可以替换
[fork.system]
ForkChainParamV1= 0
ForkStateDBSet=-1
ForkCheckTxDup=0
ForkBlockHash= 1
ForkMinerTime= 0
ForkTransferExec= 100000
ForkExecKey= 200000
ForkTxGroup= 200000
ForkResetTx0= 200000
ForkWithdraw= 200000
ForkExecRollback= 450000
ForkCheckBlockTime=1200000
ForkMultiSignAddress=1298600
ForkTxHeight= -1
ForkTxGroupPara= -1
ForkChainParamV2= -1
ForkBlockCheck=1725000
ForkLocalDBAccess=1

[fork.sub.coins]
Enable=0

[fork.sub.manage]
Enable=0
ForkManageExec=100000
`
