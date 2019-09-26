package types

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/33cn/chain33/common/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlock(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	b := &Block{}
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hex.EncodeToString(b.Hash(cfg)))
	assert.Equal(t, b.HashOld(), b.HashNew())
	assert.Equal(t, b.HashOld(), b.Hash(cfg))
	b.Height = 10
	b.Difficulty = 1
	assert.NotEqual(t, b.HashOld(), b.HashNew())
	assert.NotEqual(t, b.HashOld(), b.HashNew())
	assert.Equal(t, b.HashNew(), b.HashByForkHeight(10))
	assert.Equal(t, b.HashOld(), b.HashByForkHeight(11))
	assert.Equal(t, true, b.CheckSign(cfg))

	b.Txs = append(b.Txs, &Transaction{})
	assert.Equal(t, false, b.CheckSign(cfg))
	b.Txs = append(b.Txs, &Transaction{})
	b.Txs = append(b.Txs, &Transaction{})
	b.Txs = append(b.Txs, &Transaction{})
	b.Txs = append(b.Txs, &Transaction{})
	assert.Equal(t, false, b.CheckSign(cfg))
}

func TestFilterParaTxsByTitle(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	to := "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

	//构造一个主链交易
	maintx := &Transaction{Execer: []byte("coins"), Payload: []byte("none")}
	maintx.To = to
	maintx, err := FormatTx(cfg, "coins", maintx)
	require.NoError(t, err)

	//构造一个平行链交易
	execer := "user.p.hyb.none"
	paratx := &Transaction{Execer: []byte(execer), Payload: []byte("none")}
	paratx.To = address.ExecAddress(execer)
	paratx, err = FormatTx(cfg, execer, paratx)
	require.NoError(t, err)

	//构造一个平行链交易组
	execer1 := "user.p.hyb.coins"
	tx1 := &Transaction{Execer: []byte(execer1), Payload: []byte("none")}
	tx1.To = address.ExecAddress(execer1)
	tx1, err = FormatTx(cfg, execer1, tx1)
	require.NoError(t, err)

	execer2 := "user.p.hyb.token"
	tx2 := &Transaction{Execer: []byte(execer2), Payload: []byte("none")}
	tx2.To = address.ExecAddress(execer2)
	tx2, err = FormatTx(cfg, execer2, tx2)
	require.NoError(t, err)

	execer3 := "user.p.hyb.trade"
	tx3 := &Transaction{Execer: []byte(execer3), Payload: []byte("none")}
	tx3.To = address.ExecAddress(execer3)
	tx3, err = FormatTx(cfg, execer3, tx3)
	require.NoError(t, err)

	var txs Transactions
	txs.Txs = append(txs.Txs, tx1)
	txs.Txs = append(txs.Txs, tx2)
	txs.Txs = append(txs.Txs, tx3)
	feeRate := cfg.GInt("MinFee")
	group, err := CreateTxGroup(txs.Txs, feeRate)
	require.NoError(t, err)

	//构造一个有平行链交易的区块
	block := &Block{}
	block.Version = 0
	block.Height = 0
	block.BlockTime = 1
	block.Difficulty = 1
	block.Txs = append(block.Txs, maintx)
	block.Txs = append(block.Txs, paratx)
	block.Txs = append(block.Txs, group.Txs...)

	blockdetal := &BlockDetail{}
	blockdetal.Block = block

	maintxreceipt := &ReceiptData{Ty: ExecOk}
	paratxreceipt := &ReceiptData{Ty: ExecPack}
	grouppara1receipt := &ReceiptData{Ty: ExecPack}
	grouppara2receipt := &ReceiptData{Ty: ExecPack}
	grouppara3receipt := &ReceiptData{Ty: ExecPack}

	blockdetal.Receipts = append(blockdetal.Receipts, maintxreceipt)
	blockdetal.Receipts = append(blockdetal.Receipts, paratxreceipt)
	blockdetal.Receipts = append(blockdetal.Receipts, grouppara1receipt)
	blockdetal.Receipts = append(blockdetal.Receipts, grouppara2receipt)
	blockdetal.Receipts = append(blockdetal.Receipts, grouppara3receipt)

	txDetail := blockdetal.FilterParaTxsByTitle(cfg, "user.p.hyb.")
	for _, tx := range txDetail.TxDetails {
		if tx != nil {
			execer := string(tx.Tx.Execer)
			if !strings.HasPrefix(execer, "user.p.hyb.") && tx.Tx.GetGroupCount() != 0 {
				assert.Equal(t, tx.Receipt.Ty, int32(ExecOk))
			} else {
				assert.Equal(t, tx.Receipt.Ty, int32(ExecPack))
			}
		}
	}
}

func GetDefaultCfgstring() string {
	return `
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
minerExecs=["ticket", "autonomy"]

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

[mver.consensus.ForkTicketFundAddrV1]
fundKeyAddr = "1Ji3W12KGScCM7C2p8bg635sNkayDM8MGY"

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
[exec.sub.autonomy]
total="16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"
useBalance=false
`
}