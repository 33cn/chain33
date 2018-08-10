package types

var userKey = []byte("user.")
var slash = []byte("-")

const Debug = false
const (
	CoinsX          = "coins"
	TicketX         = "ticket"
	HashlockX       = "hashlock"
	RetrieveX       = "retrieve"
	NoneX           = "none"
	TokenX          = "token"
	TradeX          = "trade"
	ManageX         = "manage"
	PrivacyX        = "privacy"
	ExecerEvmString = "evm"
	EvmX            = "evm"
	RelayX          = "relay"
	Normx           = "norm"
	UserEvmX        = "user.evm."
	CertX           = "cert"
	GameX           = "game"
)

var (
	ExecerCoins    = []byte(CoinsX)
	ExecerTicket   = []byte(TicketX)
	ExecerManage   = []byte(ManageX)
	ExecerToken    = []byte(TokenX)
	ExecerEvm      = []byte(EvmX)
	ExecerPrivacy  = []byte(PrivacyX)
	ExecerRelay    = []byte(RelayX)
	ExecerHashlock = []byte(HashlockX)
	ExecerRetrieve = []byte(RetrieveX)
	ExecerNone     = []byte(NoneX)
	ExecerTrade    = []byte(TradeX)
	ExecerNorm     = []byte(Normx)
	ExecerConfig   = []byte("config")
	ExecerCert     = []byte(CertX)
	UserEvm        = []byte(UserEvmX)
	ExecerGame     = []byte(GameX)
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

	//log for ticket
	TyLogNewTicket   = 111
	TyLogCloseTicket = 112
	TyLogMinerTicket = 113
	TyLogTicketBind  = 114

	//log for token create
	TyLogPreCreateToken    = 211
	TyLogFinishCreateToken = 212
	TyLogRevokeCreateToken = 213

	//log for trade
	TyLogTradeSellLimit       = 310
	TyLogTradeBuyMarket       = 311
	TyLogTradeSellRevoke      = 312
	TyLogTokenTransfer        = 313
	TyLogTokenGenesis         = 314
	TyLogTokenDeposit         = 315
	TyLogTokenExecTransfer    = 316
	TyLogTokenExecWithdraw    = 317
	TyLogTokenExecDeposit     = 318
	TyLogTokenExecFrozen      = 319
	TyLogTokenExecActive      = 320
	TyLogTokenGenesisTransfer = 321
	TyLogTokenGenesisDeposit  = 322
	TyLogTradeSellMarket      = 330
	TyLogTradeBuyLimit        = 331
	TyLogTradeBuyRevoke       = 332

	//log for relay
	TyLogRelayCreate       = 350
	TyLogRelayRevokeCreate = 351
	TyLogRelayAccept       = 352
	TyLogRelayRevokeAccept = 353
	TyLogRelayConfirmTx    = 354
	TyLogRelayFinishTx     = 355
	TyLogRelayRcvBTCHead   = 356

	// log for config
	TyLogModifyConfig = 410

	// log for privacy
	TyLogPrivacyFee    = 500
	TyLogPrivacyInput  = 501
	TyLogPrivacyOutput = 502

	// log for evm
	// 合约代码变更日志
	TyLogContractData = 601
	// 合约状态数据变更日志
	TyLogContractState = 602
	// 合约状态数据变更日志
	TyLogCallContract = 603
	// 合约状态数据变更项日志
	TyLogEVMStateChangeItem = 604

	//log for game
	TyLogCreateGame = 711
	TyLogMatchGame  = 712
	TyLogCancleGame = 713
	TyLogCloseGame  = 714
)

//exec type
const (
	ExecErr  = 0
	ExecPack = 1
	ExecOk   = 2
)

const (
	InvalidAction       = 0
	CoinsActionTransfer = 1
	CoinsActionGenesis  = 2
	CoinsActionWithdraw = 3

	//action for token
	ActionTransfer            = 4
	ActionGenesis             = 5
	ActionWithdraw            = 6
	TokenActionPreCreate      = 7
	TokenActionFinishCreate   = 8
	TokenActionRevokeCreate   = 9
	CoinsActionTransferToExec = 10
	TokenActionTransferToExec = 11
	//action type for privacy
	ActionPublic2Privacy = iota + 100
	ActionPrivacy2Privacy
	ActionPrivacy2Public
)

//ticket
const (
	TicketActionGenesis = 11
	TicketActionOpen    = 12
	TicketActionClose   = 13
	TicketActionList    = 14 //读的接口不直接经过transaction
	TicketActionInfos   = 15 //读的接口不直接经过transaction
	TicketActionMiner   = 16
	TicketActionBind    = 17
)

// hashlock status
const (
	HashlockActionLock   = 1
	HashlockActionSend   = 2
	HashlockActionUnlock = 3
)

//norm
const (
	NormActionPut = 1
)

//cert
const (
	CertActionNew    = 1
	CertActionUpdate = 2
	CertActionNormal = 3
)

// retrieve op
const (
	RetrievePre    = 1
	RetrievePerf   = 2
	RetrieveBackup = 3
	RetrieveCancel = 4
)

// token status
const (
	TokenStatusPreCreated = iota
	TokenStatusCreated
	TokenStatusCreateRevoked
)

// trade op
const (
	TradeSellLimit = iota
	TradeBuyMarket
	TradeRevokeSell
	TradeSellMarket
	TradeBuyLimit
	TradeRevokeBuy
)

// 0->not start, 1->on sale, 2->sold out, 3->revoke, 4->expired
const (
	TradeOrderStatusNotStart = iota
	TradeOrderStatusOnSale
	TradeOrderStatusSoldOut
	TradeOrderStatusRevoked
	TradeOrderStatusExpired
	TradeOrderStatusOnBuy
	TradeOrderStatusBoughtOut
	TradeOrderStatusBuyRevoked
)

var SellOrderStatus = map[int32]string{
	TradeOrderStatusNotStart:   "NotStart",
	TradeOrderStatusOnSale:     "OnSale",
	TradeOrderStatusSoldOut:    "SoldOut",
	TradeOrderStatusRevoked:    "Revoked",
	TradeOrderStatusExpired:    "Expired",
	TradeOrderStatusOnBuy:      "OnBuy",
	TradeOrderStatusBoughtOut:  "BoughtOut",
	TradeOrderStatusBuyRevoked: "BuyRevoked",
}

var SellOrderStatus2Int = map[string]int32{
	"NotStart":   TradeOrderStatusNotStart,
	"OnSale":     TradeOrderStatusOnSale,
	"SoldOut":    TradeOrderStatusSoldOut,
	"Revoked":    TradeOrderStatusRevoked,
	"Expired":    TradeOrderStatusExpired,
	"OnBuy":      TradeOrderStatusOnBuy,
	"BoughtOut":  TradeOrderStatusBoughtOut,
	"BuyRevoked": TradeOrderStatusBuyRevoked,
}

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

var MapSellOrderStatusStr2Int = map[string]int32{
	"onsale":  TradeOrderStatusOnSale,
	"soldout": TradeOrderStatusSoldOut,
	"revoked": TradeOrderStatusRevoked,
}

// relay
const (
	RelayRevokeCreate = iota
	RelayRevokeAccept
)

const (
	RelayOrderBuy = iota
	RelayOrderSell
)

var RelayOrderOperation = map[uint32]string{
	RelayOrderBuy:  "buy",
	RelayOrderSell: "sell",
}

const (
	RelayUnlock = iota
	RelayCancel
)

//relay action ty
const (
	RelayActionCreate = iota
	RelayActionAccept
	RelayActionRevoke
	RelayActionConfirmTx
	RelayActionVerifyTx
	RelayActionVerifyCmdTx
	RelayActionRcvBTCHeaders
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

//game action ty
const (
	GameActionCreate = iota
	GameActionMatch
	GameActionCancel
	GameActionClose
)
