// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"
)

var slash = []byte("-")
var sharp = []byte("#")

//Debug 调试开关
var Debug = false

//LogErr log错误信息
type LogErr []byte

//LogReserved LogReserved信息
type LogReserved []byte

//LogInfo loginfo信息
type LogInfo struct {
	Ty   reflect.Type
	Name string
}

//UserKeyX 用户自定义执行器前缀字符串
const (
	UserKeyX = "user."
	ParaKeyX = "user.p."
	NoneX    = "none"
)

//UserKeyX 用户自定义执行器前缀byte类型
var (
	UserKey    = []byte(UserKeyX)
	ParaKey    = []byte(ParaKeyX)
	ExecerNone = []byte(NoneX)
)

//基本全局常量定义
const (
	InputPrecision        float64 = 1e4
	Multiple1E4           int64   = 1e4
	BTY                           = "BTY"
	BTYDustThreshold              = Coin
	ConfirmedHeight               = 12
	UTXOCacheCount                = 256
	SignatureSize                 = (4 + 33 + 65)
	PrivacyMaturityDegree         = 12
	TxGroupMaxCount               = 20
	MinerAction                   = "miner"
	Int1E4                int64   = 10000
	Float1E4              float64 = 10000.0
)

//全局账户私钥/公钥
var (
	//ViewPubFee 公钥
	//addr:1Cbo5u8V5F3ubWBv9L6qu9wWxKuD3qBVpi,这里只是作为测试用，后面需要修改为系统账户
	ViewPubFee  = "0x0f7b661757fe8471c0b853b09bf526b19537a2f91254494d19874a04119415e8"
	SpendPubFee = "0x64204db5a521771eeeddee59c25aaae6bebe796d564effb6ba11352418002ee3"
	ViewPrivFee = "0x0f7b661757fe8471c0b853b09bf526b19537a2f91254494d19874a04119415e8"
)

//ty = 1 -> secp256k1
//ty = 2 -> ed25519
//ty = 3 -> sm2
//ty = 4 -> onetimeed25519
//ty = 5 -> RingBaseonED25519
//ty = 1+offset(1<<8) ->auth_ecdsa
//ty = 2+offset(1<<8) -> auth_sm2
const (
	Invalid   = 0
	SECP256K1 = 1
	ED25519   = 2
	SM2       = 3
)

// 创建隐私交易的类型定义
const (
	PrivacyTypePublic2Privacy = iota + 1
	PrivacyTypePrivacy2Privacy
	PrivacyTypePrivacy2Public
)

//log type
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

//SystemLog 系统log日志
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

//exec type
const (
	ExecErr  = 0
	ExecPack = 1
	ExecOk   = 2
)

func init() {
	S("TxHeight", false)
}

//flag:

//TxHeight 选项
//设计思路:
//提供一种可以快速查重的交易类型，和原来的交易完全兼容
//并且可以通过开关控制是否开启这样的交易

//TxHeightFlag 标记是一个时间还是一个 TxHeight
var TxHeightFlag int64 = 1 << 62

//HighAllowPackHeight eg: current Height is 10000
//TxHeight is  10010
//=> Height <= TxHeight + HighAllowPackHeight
//=> Height >= TxHeight - LowAllowPackHeight
//那么交易可以打包的范围是: 10010 - 100 = 9910 , 10010 + 200 =  10210 (9910,10210)
//可以合法的打包交易
//注意，这两个条件必须同时满足.
//关于交易去重复:
//也就是说，另外一笔相同的交易，只能被打包在这个区间(9910,10210)。
//那么检查交易重复的时候，我只要检查 9910 - currentHeight 这个区间的交易不要重复就好了
var HighAllowPackHeight int64 = 90

//LowAllowPackHeight 允许打包的low区块高度
var LowAllowPackHeight int64 = 30

//EnableTxGroupParaFork 默认情况下不开启fork
var EnableTxGroupParaFork = false
