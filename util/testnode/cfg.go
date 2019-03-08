// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testnode

var cfgstring = `
Title="local"
TestNet=true
FixTime=false

[log]
# 日志级别，支持debug(dbug)/info/warn/error(eror)/crit
loglevel = "debug"
logConsoleLevel = "info"
# 日志文件名，可带目录，所有生成的日志文件都放到此目录下
logFile = "logs/chain33.log"
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
defCacheSize=128
maxFetchBlockNum=128
timeoutSeconds=5
batchBlockNum=128
driver="leveldb"
dbPath="datadir"
dbCache=64
isStrongConsistency=false
singleMode=true
batchsync=false
isRecordBlockSequence=true
isParaChain=false
enableTxQuickIndex=true

[p2p]
seeds=[]
enable=false
isSeed=false
serverStart=true
innerSeedEnable=true
useGithub=true
innerBounds=300
msgCacheSize=10240
driver="leveldb"
dbPath="datadir/addrbook"
dbCache=4
grpcLogFile="grpc33.log"

[rpc]
jrpcBindAddr="localhost:0"
grpcBindAddr="localhost:0"
whitelist=["127.0.0.1"]
jrpcFuncWhitelist=["*"]
grpcFuncWhitelist=["*"]

[mempool]
name="timeline"
poolCacheSize=102400
minTxFee=100000
maxTxNumPerAccount=100000

[consensus]
name="solo"
minerstart=true
genesisBlockTime=1514533394
genesis="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

[mver.consensus]
fundKeyAddr = "1BQXS6TxaYYG5mADaWij4AxhZZUTpw95a5"
coinReward = 18
coinDevFund = 12
ticketPrice = 10000
powLimitBits = "0x1f00ffff"
retargetAdjustmentFactor = 4
futureBlockTime = 16
ticketFrozenTime = 5
ticketWithdrawTime = 10
ticketMinerWaitTime = 2
maxTxNumber = 10000
targetTimespan = 2304
targetTimePerBlock = 16

[mver.consensus.ForkChainParamV1]
maxTxNumber = 10000
targetTimespan = 288 #only for test
targetTimePerBlock = 2

[mver.consensus.ForkChainParamV2]
powLimitBits = "0x1f2fffff"

[consensus.sub.para]
ParaRemoteGrpcClient="localhost:8802"
#主链指定高度的区块开始同步
startHeight=345850
#打包时间间隔，单位秒
writeBlockSeconds=2
#主链每隔几个没有相关交易的区块，平行链上打包空区块
emptyBlockInterval=50
#验证账户，验证节点需要配置自己的账户，并且钱包导入对应种子，非验证节点留空
authAccount=""
#等待平行链共识消息在主链上链并成功的块数，超出会重发共识消息，最小是2
waitBlocks4CommitMsg=2
searchHashMatchedBlockDepth=100

[consensus.sub.solo]
genesis="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
genesisBlockTime=1514533394
waitTxMs=10

[consensus.sub.ticket]
genesisBlockTime=1514533394
[[consensus.sub.ticket.genesis]]
minerAddr="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
returnAddr="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
count=10000

[[consensus.sub.ticket.genesis]]
minerAddr="1PUiGcbsccfxW3zuvHXZBJfznziph5miAo"
returnAddr="1EbDHAXpoiewjPLX9uqoz38HsKqMXayZrF"
count=1000

[[consensus.sub.ticket.genesis]]
minerAddr="1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX"
returnAddr="1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"
count=1000

[store]
name="mavl"
driver="leveldb"
dbPath="datadir/mavltree"
dbCache=128

[store.sub.mavl]
enableMavlPrefix=false
enableMVCC=false

[wallet]
minFee=100000
driver="leveldb"
dbPath="wallet"
dbCache=16
signType="secp256k1"

[wallet.sub.ticket]
minerdisable=false
minerwhitelist=["*"]

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

[exec.sub.relay]
genesis="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

[exec.sub.cert]
# 是否启用证书验证和签名
enable=false
# 加密文件路径
cryptoPath="authdir/crypto"
# 带证书签名类型，支持"auth_ecdsa", "auth_sm2"
signType="auth_ecdsa"

[exec.sub.manage]
superManager=[
    "1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S", 
    "12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv", 
    "1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK"
]
`
