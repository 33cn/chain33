/*
 * Copyright Fuzamei Corp. 2018 All Rights Reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package strategy

//const
const (
	// 創建main.go的文件模板
	CpftMainGo = `package main

import (
	_ "github.com/33cn/chain33/system"
	_ "${PROJECTPATH}/plugin"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/cli"
)

func main() {
	types.S("cfg.${PROJECTNAME}", ${PROJECTNAME})
	cli.RunChain33("${PROJECTNAME}")
}
`

	// 生成的配置文件模板 xxx.toml
	CpftCfgToml = `
Title="${PROJECTNAME}"
FixTime=false

[log]
# 日志级别，支持debug(dbug)/info/warn/error(eror)/crit
loglevel = "debug"
logConsoleLevel = "info"
# 日志文件名，可带目录，所有生成的日志文件都放到此目录下
logFile = "logs/${PROJECTNAME}.log"
# 单个日志文件的最大值（单位：兆）
maxFileSize = 300
# 最多保存的历史日志文件个数
maxBackups = 100
# 最多保存的历史日志消息（单位：天）
maxAge = 28
# 日志文件名是否使用本地事件（否则使用UTC时间）
localTime = true
# 历史日志文件是否压缩（压缩格式为gz）
compress = true
# 是否打印调用源文件和行号
callerFile = false
# 是否打印调用方法
callerFunction = false

[blockchain]
dbPath="datadir"
dbCache=64
batchsync=false
isRecordBlockSequence=false
enableTxQuickIndex=false

[p2p]
seeds=[]
isSeed=false
innerSeedEnable=true
useGithub=true
innerBounds=300
dbPath="datadir/addrbook"
dbCache=4
grpcLogFile="grpc33.log"

[rpc]
jrpcBindAddr="localhost:8801"
grpcBindAddr="localhost:8802"
whitelist=["127.0.0.1"]
jrpcFuncWhitelist=["*"]
grpcFuncWhitelist=["*"]

[mempool]
maxTxNumPerAccount=100

[store]
dbPath="datadir/mavltree"
dbCache=128
enableMavlPrefix=false
enableMVCC=false
enableMavlPrune=false
pruneHeight=10000

[wallet]
dbPath="wallet"
dbCache=16

[wallet.sub.ticket]
minerdisable=false
minerwhitelist=["*"]

[exec]
enableStat=false
enableMVCC=false

[exec.sub.token]
saveTokenTxList=false
`

	CpftRunmainBlock = `package main

var ${PROJECTNAME} = `

	// 生成项目运行主程序的模板 xxx.go
	// 顶部还需要加上package main
	//var bityuan = `CPFT_RUNMAIN`
	CpftRunMain = `TestNet=false
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

	// 生成项目Makefile文件的模板 Makefile
	CpftMakefile = `
CHAIN33=github.com/33cn/chain33
CHAIN33_PATH=vendor/${CHAIN33}
all: vendor proto build

build:
	go build -i -o ${PROJECTNAME}
	go build -i -o ${PROJECTNAME}-cli ${PROJECTPATH}/cli

vendor:
	make update
	make updatevendor

proto: 
	cd ${GOPATH}/src/${PROJECTPATH}/plugin/dapp/${EXECNAME}/proto && sh create_protobuf.sh

update:
	go get -u -v github.com/kardianos/govendor
	rm -rf ${CHAIN33_PATH}
	git clone --depth 1 -b master https://${CHAIN33}.git ${CHAIN33_PATH}
	rm -rf vendor/${CHAIN33}/.git
	rm -rf vendor/${CHAIN33}/vendor/github.com/apache/thrift/tutorial/erl/
	cp -Rf vendor/${CHAIN33}/vendor/* vendor/
	rm -rf vendor/${CHAIN33}/vendor
	govendor init
	go build -i -o tool github.com/33cn/plugin/vendor/github.com/33cn/chain33/cmd/tools
	./tool import --path "plugin" --packname "${PROJECTPATH}/plugin" --conf "plugin/plugin.toml"

updatevendor:
	govendor add +e
	govendor fetch -v +m

clean:
	@rm -rf vendor
	@rm -rf datadir
	@rm -rf logs
	@rm -rf wallet
	@rm -rf grpc33.log
	@rm -rf ${PROJECTNAME}
	@rm -rf ${PROJECTNAME}-cli
	@rm -rf tool
	@rm -rf plugin/init.go
	@rm -rf plugin/consensus/init
	@rm -rf plugin/dapp/init
	@rm -rf plugin/crypto/init
	@rm -rf plugin/store/init
	@rm -rf plugin/mempool/init
`

	// 生成 .travis.yml 文件模板
	CpftTravisYml = `
language: go

go:
  - "1.9"
  - master
`

	// 生成 plugin/plugin.toml的文件模板
	CpftPluginToml = `
# type字段仅支持 consensus  dapp store mempool
[dapp-ticket]
gitrepo = "github.com/33cn/plugin/plugin/dapp/ticket"

[consensus-ticket]
gitrepo = "github.com/33cn/plugin/plugin/consensus/ticket"

[dapp-retrieve]
gitrepo = "github.com/33cn/plugin/plugin/dapp/retrieve"

[dapp-hashlock]
gitrepo = "github.com/33cn/plugin/plugin/dapp/hashlock"

[dapp-token]
gitrepo = "github.com/33cn/plugin/plugin/dapp/token"

[dapp-trade]
gitrepo = "github.com/33cn/plugin/plugin/dapp/trade"

[mempool-price]
gitrepo = "github.com/33cn/plugin/plugin/mempool/price"

[mempool-score]
gitrepo = "github.com/33cn/plugin/plugin/mempool/score"
`
	// 项目 cli/main.go 文件模板
	CpftCliMain = `package main

import (
	_ "${PROJECTPATH}/plugin"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/util/cli"
)

func main() {
	cli.Run("", "")
}
`
	// plugin/dapp/xxxx/commands/cmd.go文件的模板c
	CpftDappCommands = `package commands

import (
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	return nil
}`

	// plugin/dapp/xxxx/plugin.go文件的模板
	CpftDappPlugin = `package ${PROJECTNAME}

import (
	"github.com/33cn/chain33/pluginmgr"
	"${PROJECTPATH}/plugin/dapp/${PROJECTNAME}/commands"
	"${PROJECTPATH}/plugin/dapp/${PROJECTNAME}/executor"
	"${PROJECTPATH}/plugin/dapp/${PROJECTNAME}/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.${EXECNAME_FB}X,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.Cmd,
		RPC:      nil,
	})
}
`

	// plugin/dapp/xxxx/executor/xxxx.go文件模板
	CpftDappExec = `package executor

import (
	log "github.com/inconshreveable/log15"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
)

var clog = log.New("module", "execs.${EXECNAME}")
var driverName = "${EXECNAME}"

func init() {
	ety := types.LoadExecutorType(driverName)
	if ety != nil {
		ety.InitFuncList(types.ListMethod(&${CLASSNAME}{}))
	}
}

func Init(name string, sub []byte) {
	ety := types.LoadExecutorType(driverName)
	if ety != nil {
		ety.InitFuncList(types.ListMethod(&${CLASSNAME}{}))
	}

	clog.Debug("register ${EXECNAME} execer")
	drivers.Register(GetName(), new${CLASSNAME}, types.GetDappFork(driverName, "Enable"))
}

func GetName() string {
	return new${CLASSNAME}().GetName()
}

type ${CLASSNAME} struct {
	drivers.DriverBase
}

func new${CLASSNAME}() drivers.Driver {
	n := &${CLASSNAME}{}
	n.SetChild(n)
	n.SetIsFree(true)
	n.SetExecutorType(types.LoadExecutorType(driverName))
	return n
}

func (this *${CLASSNAME}) GetDriverName() string {
	return driverName
}

func (this *${CLASSNAME}) CheckTx(tx *types.Transaction, index int) error {
	return nil
}
`
	// plugin/dapp/xxxx/proto/create_protobuf.sh文件模板
	CpftDappCreatepb = `#!/bin/sh
protoc --go_out=plugins=grpc:../types ./*.proto --proto_path=. 
`

	// plugin/dapp/xxxx/proto/Makefile 文件模板
	CpftDappMakefile = `all:
	sh ./create_protobuf.sh
`

	// plugin/dapp/xxxx/proto/xxxx.proto的文件模板
	CpftDappProto = `syntax = "proto3";
package types;

message ${ACTIONNAME} {
    int32 ty = 1;
	oneof value {
		${ACTIONNAME}None none = 2;
	}
}

message ${ACTIONNAME}None {
}
`

	// plugin/dapp/xxxx/types/types.go的文件模板cd
	CpftDappTypefile = `package types

import (
	"github.com/33cn/chain33/types"
)

var (
	${EXECNAME_FB}X      = "${EXECNAME}"
	Execer${EXECNAME_FB} = []byte(${EXECNAME_FB}X)
	actionName  = map[string]int32{}
	logmap = map[int64]*types.LogInfo{}
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, Execer${EXECNAME_FB})
	types.RegistorExecutor("${EXECNAME}", NewType())

	types.RegisterDappFork(${EXECNAME_FB}X, "Enable", 0)
}

type ${CLASSTYPENAME} struct {
	types.ExecTypeBase
}

func NewType() *${CLASSTYPENAME} {
	c := &${CLASSTYPENAME}{}
	c.SetChild(c)
	return c
}

func (coins *${CLASSTYPENAME}) GetPayload() types.Message {
	return &${ACTIONNAME}{}
}

func (coins *${CLASSTYPENAME}) GetName() string {
	return ${EXECNAME_FB}X
}

func (coins *${CLASSTYPENAME}) GetLogMap() map[int64]*types.LogInfo {
	return logmap
}

func (c *${CLASSTYPENAME}) GetTypeMap() map[string]int32 {
	return actionName
}
`
)
