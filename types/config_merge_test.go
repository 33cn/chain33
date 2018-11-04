package types

import (
	"bytes"
	"testing"

	tml "github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
)

var defconfig = `Title="bityuan"
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

[consensus]
name="ticket"
minerstart=true
genesisBlockTime=1514533394
genesis="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

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

#系统中所有的fork,默认用chain33的测试网络的
#但是我们可以替换
[fork.system]
ForkCheckTxDup=0
ForkChainParamV1= 0
ForkBlockHash= 1
ForkMinerTime= 0
ForkTransferExec= 100000
ForkExecKey= 200000
ForkTxGroup= 200000
ForkResetTx0= 200000
ForkWithdraw= 200000
ForkExecRollback= 450000
ForkTxHeight= -1
ForkTxGroupPara= -1

[fork.sub.coins]
Enable=0

[fork.sub.ticket]
Enable=0
ForkTicketId = 1200000

[fork.sub.retrieve]
Enable=0
ForkRetrive=0

[fork.sub.hashlock]
Enable=0

[fork.sub.manage]
Enable=0
ForkManageExec=100000

[fork.sub.token]
Enable=0
ForkTokenBlackList= 0
ForkBadTokenSymbol= 0
ForkTokenPrice= 300000

[fork.sub.trade]
Enable=0
ForkTradeBuyLimit= 0
ForkTradeAsset= -1
`

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
	//1. def
	def := make(map[string]interface{})
	_, err := tml.Decode(defconfig, &def)
	assert.Nil(t, err)

	//2. userconfig
	conf := make(map[string]interface{})
	_, err = tml.DecodeFile("../cmd/chain33/bityuan.toml", &conf)
	assert.Nil(t, err)

	//3. merge
	errstr := MergeConfig(conf, def)
	if errstr != "" {
		t.Log(errstr)
	}
	//4. check ok
	buf := new(bytes.Buffer)
	err = tml.NewEncoder(buf).Encode(conf)
	assert.Nil(t, err)
	assert.NotEqual(t, buf.String(), "")
	cfg1, err := initString(buf.String())
	assert.Nil(t, err)

	cfg2, err := initCfg("testdata/bityuan.toml")
	assert.Nil(t, err)
	assert.Equal(t, cfg1, cfg2)
}
