package types

import "reflect"

var slash = []byte("-")
var Debug = false

type LogErr []byte
type LogReserved []byte

type LogInfo struct {
	Ty   reflect.Type
	Name string
}

const (
	CoinsX    = "coins"
	UserKeyX  = "user."
	ParaKeyX  = "user.p."
	TicketX   = "ticket"
	HashlockX = "hashlock"
	NoneX     = "none"
	TokenX    = "token"
	TradeX    = "trade"
	ManageX   = "manage"
	PrivacyX  = "privacy"
	RelayX    = "relay"
	Normx     = "norm"
	ParaX     = "paracross"
	ValNodeX  = "valnode"
)

var (
	ExecerCoins    = []byte(CoinsX)
	ExecerTicket   = []byte(TicketX)
	ExecerManage   = []byte(ManageX)
	ExecerToken    = []byte(TokenX)
	ExecerPrivacy  = []byte(PrivacyX)
	ExecerRelay    = []byte(RelayX)
	ExecerHashlock = []byte(HashlockX)
	ExecerNone     = []byte(NoneX)
	ExecerTrade    = []byte(TradeX)
	ExecerNorm     = []byte(Normx)
	ExecerConfig   = []byte("config")
	ExecerPara     = []byte(ParaX)
	UserKey        = []byte(UserKeyX)
	ParaKey        = []byte(ParaKeyX)
	ExecerValNode  = []byte(ValNodeX)
)

const (
	InputPrecision        float64 = 1e4
	Multiple1E4           int64   = 1e4
	TokenNameLenLimit             = 128
	TokenSymbolLenLimit           = 16
	TokenIntroLenLimit            = 1024
	InvalidStartTime              = 0
	InvalidStopTime               = 0
	BlockDurPerSecCnt             = 15
	BTY                           = "BTY"
	BTYDustThreshold              = Coin
	ConfirmedHeight               = 12
	UTXOCacheCount                = 256
	M_1_TIMES                     = 1
	M_2_TIMES                     = 2
	M_5_TIMES                     = 5
	M_10_TIMES                    = 10
	SignatureSize                 = (4 + 33 + 65)
	PrivacyMaturityDegree         = 12
	TxGroupMaxCount               = 20
	MinerAction                   = "miner"
)

var (
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
	Invalid           = 0
	SECP256K1         = 1
	ED25519           = 2
	SM2               = 3
	OnetimeED25519    = 4
	RingBaseonED25519 = 5
	AUTH_ECDSA        = 257
	AUTH_SM2          = 258
)

//const (
//	SignTypeInvalid        = 0
//	SignTypeSecp256k1      = 1
//	SignTypeED25519        = 2
//	SignTypeSM2            = 3
//	SignTypeOnetimeED25519 = 4
//	SignTypeRing           = 5
//)

const (
	SignNameSecp256k1      = "secp256k1"
	SignNameED25519        = "ed25519"
	SignNameSM2            = "sm2"
	SignNameOnetimeED25519 = "onetimeed25519"
	SignNameRing           = "RingSignatue"
	SignNameAuthECDSA      = "auth_ecdsa"
	SignNameAuthSM2        = "auth_sm2"
)

// 创建隐私交易的类型定义
const (
	PrivacyTypePublic2Privacy = iota + 1
	PrivacyTypePrivacy2Privacy
	PrivacyTypePrivacy2Public
)

var MapSignType2name = map[int]string{
	SECP256K1:         SignNameSecp256k1,
	ED25519:           SignNameED25519,
	SM2:               SignNameSM2,
	OnetimeED25519:    SignNameOnetimeED25519,
	RingBaseonED25519: SignNameRing,
	AUTH_ECDSA:        SignNameAuthECDSA,
	AUTH_SM2:          SignNameAuthSM2,
}

var MapSignName2Type = map[string]int{
	SignNameSecp256k1:      SECP256K1,
	SignNameED25519:        ED25519,
	SignNameSM2:            SM2,
	SignNameOnetimeED25519: OnetimeED25519,
	SignNameRing:           RingBaseonED25519,
	SignNameAuthECDSA:      AUTH_ECDSA,
	SignNameAuthSM2:        AUTH_SM2,
}

//log type
const (
	TyLogReserved = 0
	TyLogErr      = 1
	TyLogFee      = 2
	//coins
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
)

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
	TyLogExecActive:      {reflect.TypeOf(ReceiptAccountTransfer{}), "LogExecActive"},
	TyLogGenesisTransfer: {reflect.TypeOf(ReceiptAccountTransfer{}), "LogGenesisTransfer"},
	TyLogGenesisDeposit:  {reflect.TypeOf(ReceiptAccountTransfer{}), "LogGenesisDeposit"},
}

const (
	//log for trade
	TyLogTradeSellLimit  = 310
	TyLogTradeBuyMarket  = 311
	TyLogTradeSellRevoke = 312

	TyLogTradeSellMarket        = 330
	TyLogTradeBuyLimit          = 331
	TyLogTradeBuyRevoke         = 332
	TyLogParaTokenAssetTransfer = 333
	TyLogParaTokenAssetWithdraw = 334

	// log for config
	TyLogModifyConfig = 410
)

//exec type
const (
	ExecErr  = 0
	ExecPack = 1
	ExecOk   = 2
)

// manager action
const (
	ManageActionModifyConfig = iota
)

// config items
const (
	ConfigItemArrayConfig = iota
	ConfigItemIntConfig
	ConfigItemStringConfig
)

// RescanUtxoFlag
const (
	UtxoFlagNoScan  int32 = 0
	UtxoFlagScaning int32 = 1
	UtxoFlagScanEnd int32 = 2
)

var RescanFlagMapint2string = map[int32]string{
	UtxoFlagNoScan:  "UtxoFlagNoScan",
	UtxoFlagScaning: "UtxoFlagScaning",
	UtxoFlagScanEnd: "UtxoFlagScanEnd",
}

//flag:
var FlagTxQuickIndex = []byte("FLAG:FlagTxQuickIndex")
var FlagKeyMVCC = []byte("FLAG:keyMVCCFlag")

//TxHeight 选项
//设计思路:
//提供一种可以快速查重的交易类型，和原来的交易完全兼容
//并且可以通过开关控制是否开启这样的交易

//标记是一个时间还是一个 TxHeight
var TxHeightFlag int64 = 1 << 62

//是否开启TxHeight选项
var EnableTxHeight = false

//eg: current Height is 10000
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
var LowAllowPackHeight int64 = 30

//默认情况下不开启fork
var EnableTxGroupParaFork = false
