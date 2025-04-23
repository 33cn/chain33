// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

var cfgstring = `
Title="local"
TestNet=true
FixTime=false
TxHeight=true
CoinSymbol="bty"
ChainID=33

[address]
defaultDriver="btc"
[address.enableHeight]
eth=-2

# crypto模块配置
[crypto]
enableTypes=[]    #设置启用的加密插件名称，不配置启用所有
[crypto.enableHeight]  #配置已启用插件的启用高度，不配置采用默认高度0， 负数表示不启用
secp256k1=0
[crypto.sub.secp256k1] #支持插件子配置

[crypto.sub.secp256k1eth]
#兼容EVM链的链ID
evmChainID=3999
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
txHeight=true

# 使能精简localdb
enableReduceLocaldb=false
# 关闭分片存储,默认false为开启分片存储;平行链不需要分片需要修改此默认参数为true
disableShard=false
# 分片存储中每个大块包含的区块数
chunkblockNum=1000
# 使能从P2pStore中获取数据
enableFetchP2pstore=false
# 使能假设已删除已归档数据后,获取数据情况
enableIfDelLocalChunk=false

enablePushSubscribe=true
maxActiveBlockNum=1024
maxActiveBlockSize=100


[p2p]
enable=false
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
[rpc.sub.eth]
enable=false
web3CliVer="Geth/v1.8.15-omnibus-255989da/linux-amd64/go1.10.1"

[rpc.parachain]
mainChainGrpcAddr="localhost:8802"

[mempool]
name="timeline"
poolCacheSize=102400
# 最小得交易手续费率，这个没有默认值，必填，一般是0.001 coins
minTxFeeRate=100000
maxTxNumPerAccount=100000

[consensus]
name="solo"
minerstart=true
genesisBlockTime=1514533394
genesis="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
minerExecs=["ticket", "autonomy"]

[mver.consensus]
fundKeyAddr = "1BQXS6TxaYYG5mADaWij4AxhZZUTpw95a5"
powLimitBits = "0x1f00ffff"
maxTxNumber = 10000

[mver.consensus.ForkChainParamV1]
maxTxNumber = 10000

[mver.consensus.ForkChainParamV2]
powLimitBits = "0x1f2fffff"

[mver.consensus.ForkTicketFundAddrV1]
fundKeyAddr = "1Ji3W12KGScCM7C2p8bg635sNkayDM8MGY"

[mver.consensus.ticket]
coinReward = 18
coinDevFund = 12
ticketPrice = 10000
retargetAdjustmentFactor = 4
futureBlockTime = 16
ticketFrozenTime = 5
ticketWithdrawTime = 10
ticketMinerWaitTime = 2
targetTimespan = 2304
targetTimePerBlock = 16

[mver.consensus.ticket.ForkChainParamV1]
targetTimespan = 288 #only for test
targetTimePerBlock = 2

[consensus.sub.para]
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
waitTxMs=1

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
coinType="bty"

[wallet.sub.ticket]
minerdisable=false
minerwhitelist=["*"]

[exec]
enableStat=false
enableMVCC=false

#代理执行器地址
proxyExecAddress="0x0000000000000000000000000000000000200005"
alias=["token1:token","token2:token","token3:token"]
[exec.sub.coins]
#允许evm执行器操作coins
friendExecer=["evm"]

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
autonomyExec=""

[exec.sub.autonomy]
total="16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"
useBalance=false

[exec.sub.jvm]
jdkPath="../../../../build/j2sdk-image"
`

// GetDefaultCfgstring ...
func GetDefaultCfgstring() string {
	return cfgstring
}
