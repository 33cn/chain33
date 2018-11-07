## chain33配置文件 说明

### toml 配置文件的基本语法

```
[section]
hello=world
num=1
```

可以用下面的代码获取到这个配置

```
types.Conf("config.section").GStr("hello") == "world"
types.Conf("config.section").GInt("num") == int64(1)
//注意要加上前缀 config.
```

### 配置一个字符串数组

```
[section]
tokenApprs = [
	"1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S",
	"1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK",
	"1LY8GFia5EiyoTodMLfkB5PHNNpXRqxhyB",
	"1GCzJDS6HbgTQ2emade7mEJGGWFfA15pS9",
	"1JYB8sxi4He5pZWHCd3Zi2nypQ4JMB6AxN",
	"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv",
]

可以用下面的代码获取到这个配置

types.Conf("config.section").GStrList("tokenApprs")

```

### 配置一个结构体数组

```
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

解析方法1:
types.Conf("consensus.sub.ticket").G("genesis") 可以获取到一个 []interface{}
的结构，然后再解析。

解析方法2:
在chain33 系统中，解析方法有些不同。chain33 会将存在 .sub. 这样 section
的结构体变成一个 json 字符串，然后传入每个模块，每个模块去解析这个json字符串

type genesisTicket struct {
	MinerAddr  string `json:"minerAddr"`
	ReturnAddr string `json:"returnAddr"`
	Count      int32  `json:"count"`
}

type subConfig struct {
	GenesisBlockTime int64            `json:"genesisBlockTime"`
	Genesis          []*genesisTicket `json:"genesis"`
}

func New(cfg *types.Consensus, sub []byte) queue.Module {
	c := drivers.NewBaseClient(cfg)
	var subcfg subConfig
	if sub != nil {
		types.MustDecode(sub, &subcfg)
	}
    ...
}
```

### 子模块配置

下面的模块 是可以通过 plugin 扩展的，所以，每个 sub module 可以 有专门定义的配置结构

1. consensus 共识
2. store 存储
3. exec 应用
4. wallet 钱包

例子: token 的配置
```
[exec]
isFree=false
minExecFee=100000
enableStat=false
enableMVCC=false
alias=["token1:token","token2:token","token3:token"]

[exec.sub.token]
saveTokenTxList=true
tokenApprs = [
	"1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S",
	"1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK",
	"1LY8GFia5EiyoTodMLfkB5PHNNpXRqxhyB",
	"1GCzJDS6HbgTQ2emade7mEJGGWFfA15pS9",
	"1JYB8sxi4He5pZWHCd3Zi2nypQ4JMB6AxN",
	"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv",
]
```

### Fork 配置
区块链的配置和普通的配置文件有很大的不同，区块链中要通过分叉的方式修复bug, 考虑到
分叉非常重要，每个链到开发人员都应该清楚每个分叉到内容，所以在分叉到配置上有如下到规则:

##### 必须明确配置所有的分叉规则

比如下面是BTY的配置
```
[fork.system]
ForkChainParamV1= 0
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
ForkTxHeight= -1
ForkTxGroupPara= -1
ForkChainParamV2= -1

[fork.sub.coins]
Enable=0
```

0 表示 从区块高度 0 启动分叉
10000 表示从高度 10000 启动分叉
-1 表示禁止启动分叉

每个dapp 都有一个分叉配置，而且一定有一个 Enable 到分叉配置，表示启用这个 dapp 的区块高度

##### 不能配置不存在的分叉

如果配置出错，会有一个提示信息，并且会直接panic

### 多版本配置

因为系统可能会有分叉，每个分叉可能都会有不同到配置。也就是不同的区块高度，可能取到到配置值是不同到

```
[mver.section]
hello="world"
nofork="nofork"

[mver.section.ForkMinerTime]
hello="forkworld"
```

比如我 ForkMinerTime 配置到值是 10, 那么获取配置到方法如下

```
types.Conf("mver.section").MGStr("hello", 9) == "world"
types.Conf("mver.section").MGStr("hello", 10) == "forkworld"
```

注意配置一个不存在的 ForkMinerTime 不会报错，系统会认为这个是一个普通的 名字

### 基础配置文件

在区块链中，我们可能希望有一个基础配置文件，这个配置文件对所有的人都是相同的，是用户不能修改的。
比如出快的时间，每个快的回报。比如我们要给bityuan设置一个基础配置文件，那么可以用下面的方法

```
types.S("cfg.bityuan", "content of config file")
```

这个时候，系统初始化的时候，会先读 用户的配置文件，比如是 title 是bityuan的，那么
就会查找 cfg.bityuan 这个配置的内容，如果配置的内容存在，那么 用户配置文件和系统配置文件
会进行一次合并，这个时候，注意，用户配置文件不允许 有和系统配置文件一样的配置。

比如系统配置文件中有

```
TestNet=false
```

用户配置文件中也有

```
TestNet=true
```

这个时候系统就会panic，覆盖系统默认配置是不被允许的，一般系统配置会写死在链的代码中。
这样防止用户意外修改关键配置，而引起问题。