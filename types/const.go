// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"

	"github.com/33cn/chain33/system/crypto/ed25519"
	"github.com/33cn/chain33/system/crypto/secp256k1"
	"github.com/33cn/chain33/system/crypto/secp256k1eth"
	"github.com/33cn/chain33/system/crypto/sm2"
)

var slash = []byte("-")
var sharp = []byte("#")

// Debug 调试开关
var Debug = false

// LogErr log错误信息
type LogErr []byte

// LogReserved LogReserved信息
type LogReserved []byte

// LogInfo loginfo信息
type LogInfo struct {
	Ty   reflect.Type
	Name string
}

// UserKeyX 用户自定义执行器前缀字符串
const (
	UserKeyX = "user."
	ParaKeyX = "user.p."
	NoneX    = "none"
)

// DefaultCoinsSymbol 默认的主币名称
const (
	DefaultCoinsExec     = "coins"
	DefaultCoinsSymbol   = "bty"
	DefaultCoinPrecision = int64(1e8)
)

// UserKeyX 用户自定义执行器前缀byte类型
var (
	UserKey    = []byte(UserKeyX)
	ParaKey    = []byte(ParaKeyX)
	ExecerNone = []byte(NoneX)
)

// 基本全局常量定义
const (
	BTY                          = "BTY"
	MinerAction                  = "miner"
	AirDropMinIndex       uint32 = 100000000         //通过钱包的seed生成一个空投地址，最小index索引
	AirDropMaxIndex       uint32 = 101000000         //通过钱包的seed生成一个空投地址，最大index索引
	MaxBlockCountPerTime  int64  = 1000              //从数据库中一次性获取block的最大数 1000个
	MaxBlockSizePerTime          = 100 * 1024 * 1024 //从数据库中一次性获取block的最大size100M
	AddBlock              int64  = 1
	DelBlock              int64  = 2
	MainChainName                = "main"
	MaxHeaderCountPerTime int64  = 10000 //从数据库中一次性获取header的最大数 10000个
	AutonomyCfgKey               = "autonomyExec"
)

// ty = 1 -> secp256k1
// ty = 2 -> ed25519
// ty = 3 -> sm2
// ty = 4 -> onetimeed25519
// ty = 5 -> RingBaseonED25519
// ty = 1+offset(1<<8) -> secp256r1
// ty = 2+offset(1<<8) -> sm2
// ty=  3+offset(1<<8) -> bls
// ty = 4+offset(1<<8) -> secp256k1eth
const (
	Invalid      = 0
	SECP256K1    = secp256k1.ID
	ED25519      = ed25519.ID
	SM2          = sm2.ID
	SECP256K1ETH = secp256k1eth.ID
)

// log type
const (
	TyLogReserved = 0
	TyLogErr      = 1
	TyLogFee      = 2
	//TyLogTransfer coins
	TyLogTransfer        = 3
	TyLogGenesis         = 4
	TyLogDeposit         = 5
	TyLogExecTransfer    = 6
	TyLogExecWithdraw    = 7
	TyLogExecDeposit     = 8
	TyLogExecFrozen      = 9
	TyLogExecActive      = 10
	TyLogGenesisTransfer = 11
	TyLogGenesisDeposit  = 12
	TyLogRollback        = 13
	TyLogMint            = 14
	TyLogBurn            = 15
)

// SystemLog 系统log日志
var SystemLog = map[int64]*LogInfo{
	TyLogReserved:        {reflect.TypeOf(LogReserved{}), "LogReserved"},
	TyLogErr:             {reflect.TypeOf(LogErr{}), "LogErr"},
	TyLogFee:             {reflect.TypeOf(ReceiptAccountTransfer{}), "LogFee"},
	TyLogTransfer:        {reflect.TypeOf(ReceiptAccountTransfer{}), "LogTransfer"},
	TyLogDeposit:         {reflect.TypeOf(ReceiptAccountTransfer{}), "LogDeposit"},
	TyLogExecTransfer:    {reflect.TypeOf(ReceiptExecAccountTransfer{}), "LogExecTransfer"},
	TyLogExecWithdraw:    {reflect.TypeOf(ReceiptExecAccountTransfer{}), "LogExecWithdraw"},
	TyLogExecDeposit:     {reflect.TypeOf(ReceiptExecAccountTransfer{}), "LogExecDeposit"},
	TyLogExecFrozen:      {reflect.TypeOf(ReceiptExecAccountTransfer{}), "LogExecFrozen"},
	TyLogExecActive:      {reflect.TypeOf(ReceiptExecAccountTransfer{}), "LogExecActive"},
	TyLogGenesisTransfer: {reflect.TypeOf(ReceiptAccountTransfer{}), "LogGenesisTransfer"},
	TyLogGenesisDeposit:  {reflect.TypeOf(ReceiptAccountTransfer{}), "LogGenesisDeposit"},
	TyLogRollback:        {reflect.TypeOf(LocalDBSet{}), "LogRollback"},
	TyLogMint:            {reflect.TypeOf(ReceiptAccountMint{}), "LogMint"},
	TyLogBurn:            {reflect.TypeOf(ReceiptAccountBurn{}), "LogBurn"},
}

// exec type
const (
	ExecErr  = 0
	ExecPack = 1
	ExecOk   = 2
)

// TODO 后续调试确认放的位置
//func init() {
//	S("TxHeight", false)
//}

//flag:

//TxHeight 选项
//设计思路:
//提供一种可以快速查重的交易类型，和原来的交易完全兼容
//并且可以通过开关控制是否开启这样的交易

// TxHeightFlag 标记是一个时间还是一个 TxHeight
var TxHeightFlag int64 = 1 << 62

// HighAllowPackHeight txHeight打包上限高度
// eg: currentHeight = 10000
// 某交易的expire=TxHeightFlag+ currentHeight + 10, 则TxHeight=10010
// 打包的区块高度必须满足， Height >= TxHeight - LowAllowPackHeight && Height <= TxHeight + HighAllowPackHeight
// 那么交易可以打包的范围是: 10010 - 100 = 9910 , 10010 + 200 =  10210 (9910,10210)
// 注意，这两个条件必须同时满足.
// 关于交易查重:
// 也就是说，两笔相同的交易必然有相同的expire，即TxHeight相同，以及对应的打包区间一致，只能被打包在这个区间(9910,10210)。
// 那么检查交易重复的时候，我只要检查 9910 - currentHeight 这个区间的交易是否有重复
var HighAllowPackHeight int64 = 600

// LowAllowPackHeight 允许打包的low区块高度
var LowAllowPackHeight int64 = 200

// MaxAllowPackInterval 允许打包的最大区间值
var MaxAllowPackInterval int64 = 5000
