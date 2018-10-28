package types

import (
	"strings"
	"sync"
	"time"
)

var chainBaseParam *ChainParam
var chainV3Param *ChainParam
var chainConfig map[string]interface{}
var configMutex sync.Mutex

func init() {
	initChainBase()
	initChainBityuanV3()

	chainConfig = make(map[string]interface{})
}

func initChainBase() {
	chainBaseParam = &ChainParam{}
	chainBaseParam.CoinReward = 18 * Coin  //用户回报
	chainBaseParam.CoinDevFund = 12 * Coin //发展基金回报
	chainBaseParam.TicketPrice = 10000 * Coin
	chainBaseParam.PowLimitBits = uint32(0x1f00ffff)
	chainBaseParam.RetargetAdjustmentFactor = 4
	chainBaseParam.FutureBlockTime = 16
	chainBaseParam.TicketFrozenTime = 5    //5s only for test
	chainBaseParam.TicketWithdrawTime = 10 //10s only for test
	chainBaseParam.TicketMinerWaitTime = 2 // 2s only for test
	chainBaseParam.MaxTxNumber = 1600      //160
	chainBaseParam.TargetTimespan = 144 * 16 * time.Second
	chainBaseParam.TargetTimePerBlock = 16 * time.Second
}

func initChainBityuanV3() {
	chainV3Param = &ChainParam{}
	tmp := *chainBaseParam
	//copy base param
	chainV3Param = &tmp
	//修改的值
	chainV3Param.FutureBlockTime = 15
	chainV3Param.TicketFrozenTime = 12 * 3600
	chainV3Param.TicketWithdrawTime = 48 * 3600
	chainV3Param.TicketMinerWaitTime = 2 * 3600
	chainV3Param.MaxTxNumber = 1500
	chainV3Param.TargetTimespan = 144 * 15 * time.Second
	chainV3Param.TargetTimePerBlock = 15 * time.Second
}

//title is local
func initChainTestNet() {
	chainV3Param.MaxTxNumber = 10000
	chainV3Param.TicketFrozenTime = 5                   //5s only for test
	chainV3Param.TicketWithdrawTime = 10                //10s only for test
	chainV3Param.TicketMinerWaitTime = 2                // 2s only for test
	chainV3Param.TargetTimespan = 144 * 2 * time.Second //only for test
	chainV3Param.TargetTimePerBlock = 2 * time.Second   //only for test
}

type ChainParam struct {
	CoinDevFund              int64
	CoinReward               int64
	FutureBlockTime          int64
	TicketPrice              int64
	TicketFrozenTime         int64
	TicketWithdrawTime       int64
	TicketMinerWaitTime      int64
	MaxTxNumber              int64
	PowLimitBits             uint32
	TargetTimespan           time.Duration
	TargetTimePerBlock       time.Duration
	RetargetAdjustmentFactor int64
}

func SetChainConfig(key string, value interface{}) {
	configMutex.Lock()
	chainConfig[key] = value
	configMutex.Unlock()
}

func GetChainConfig(key string) (value interface{}, err error) {
	configMutex.Lock()
	if data, ok := chainConfig[key]; ok {
		configMutex.Unlock()
		return data, nil
	}
	configMutex.Unlock()
	return nil, ErrNotFound
}

func GetP(height int64) *ChainParam {
	if height < ForkV3 {
		return chainBaseParam
	}
	return chainV3Param
}

//区块链共识相关的参数，重要参数不要随便修改
var (
	AllowDepositExec = [][]byte{ExecerTicket}
	AllowUserExec    = [][]byte{ExecerNone}

	GenesisAddr              = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
	GenesisBlockTime   int64 = 1526486816
	HotkeyAddr               = "12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
	FundKeyAddr              = "1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	EmptyValue               = []byte("FFFFFFFFemptyBVBiCj5jvE15pEiwro8TQRGnJSNsJF") //这字符串表示数据库中的空值
	SuperManager             = []string{"1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"}
	TokenApprs               = []string{}
	MinFee             int64 = 1e5
	MinBalanceTransfer int64 = 1e6
	// 隐私交易中最大的混淆度
	PrivacyMaxCount = 16
)

// coin conversation
const (
	Coin            int64 = 1e8
	MaxCoin         int64 = 1e17
	MaxTxSize             = 100000 //100K
	MaxTxGroupSize  int32 = 20
	MaxBlockSize          = 20000000 //20M
	MaxTxsPerBlock        = 100000
	TokenPrecision  int64 = 1e8
	MaxTokenBalance int64 = 900 * 1e8 * TokenPrecision //900亿
)

var (
	testNet      bool
	title        string
	FeePerKB     = MinFee
	PrivacyTxFee = Coin
	//used in Getname for exec driver
	ExecNamePrefix       string
	ParaRemoteGrpcClient string
	SaveTokenTxList      bool
)

func SetTitle(t string) {
	title = t
	if IsBityuan() {
		AllowUserExec = [][]byte{[]byte("coins"), ExecerTicket, ExecerHashlock, ExecerNone, ExecerToken, ExecerTrade, ExecerManage}
		return
	}
	if IsLocal() {
		SetForkToOne()
		initChainTestNet()
		EnableTxHeight = true
		Debug = true
		return
	}
	if IsPara() {
		//keep superManager same with mainnet
		ExecNamePrefix = title
		SetForkForPara(title)
	}
}

func GetTitle() string {
	return title
}

func IsBityuan() bool {
	return title == "bityuan"
}

func IsLocal() bool {
	return title == "local"
}

func IsYcc() bool {
	return title == "yuanchain"
}

func IsPara() bool {
	//user.p.guodun.
	return strings.Count(title, ".") == 3 && strings.HasPrefix(title, ParaKeyX)
}

func IsParaExecName(name string) bool {
	return strings.HasPrefix(name, ParaKeyX)
}

func IsPublicChain() bool {
	return IsBityuan() || IsYcc()
}

func SetTestNet(isTestNet bool) {
	if !isTestNet {
		testNet = false
		return
	}
	testNet = true
	//const 初始化TestNet 的初始化参数
	GenesisBlockTime = 1514533394
	FundKeyAddr = "1BQXS6TxaYYG5mADaWij4AxhZZUTpw95a5"
	SuperManager = []string{"1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S", "1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK"}
	TokenApprs = []string{
		"1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S",
		"1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK",
		"1LY8GFia5EiyoTodMLfkB5PHNNpXRqxhyB",
		"1GCzJDS6HbgTQ2emade7mEJGGWFfA15pS9",
		"1JYB8sxi4He5pZWHCd3Zi2nypQ4JMB6AxN",
	}
	if IsLocal() {
		return
	}
	//测试网络的Fork
	SetTestNetFork()
}

func IsTestNet() bool {
	return testNet
}

func SetMinFee(fee int64) {
	if fee < 0 {
		panic("fee less than zero")
	}
	MinFee = fee
	MinBalanceTransfer = fee * 10
}

func GetParaName() string {
	if IsPara() {
		return title
	}
	return ""
}

func FlagKV(key []byte, value int64) *KeyValue {
	return &KeyValue{Key: key, Value: Encode(&Int64{Data: value})}
}

func SetParaRemoteGrpcClient(grpc string) {
	if IsPara() {
		ParaRemoteGrpcClient = grpc
	}
}

func GetParaRemoteGrpcClient() string {
	if IsPara() {
		return ParaRemoteGrpcClient
	}
	return ""
}

func SetSaveTokenTxList(v bool) {
	SaveTokenTxList = v
}

func GetSaveTokenTxList() bool {
	return SaveTokenTxList
}
