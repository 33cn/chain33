package types

import (
	"strings"
	"sync"
	"time"
)

var chainBaseParam *ChainParam
var chainV3Param *ChainParam
var chainConfig = make(map[string]interface{})

func init() {
	initChainBase()
	initChainBityuanV3()
	S("TestNet", false)
	S("MinFee", 100000)
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

func setChainConfig(key string, value interface{}) {
	chainConfig[key] = value
}

func IsEnable(name string) bool {
	isenable, err := G(name)
	if err == nil && isenable.(bool) {
		return true
	}
	return false
}

func GInt(name string) int64 {
	value, err := G(name)
	if err != nil {
		return 0
	}
	if i, ok := value.(int64); ok {
		return i
	}
	return 0
}

func GStr(name string) string {
	value, err := G(name)
	if err != nil {
		return ""
	}
	if i, ok := value.(string); ok {
		return i
	}
	return ""
}

func S(key string, value interface{}) {
	mu.Lock()
	defer mu.Unlock()
	setChainConfig(key, value)
}

func getChainConfig(key string) (value interface{}, err error) {
	if data, ok := chainConfig[key]; ok {
		return data, nil
	}
	if isLocal() {
		panic("chain config " + key + " not found")
	}
	return nil, ErrNotFound
}

func G(key string) (value interface{}, err error) {
	mu.Lock()
	defer mu.Unlock()
	return getChainConfig(key)
}

func GetP(height int64) *ChainParam {
	if IsFork(height, "ForkChainParamV1") {
		return chainV3Param
	}
	return chainBaseParam
}

//区块链共识相关的参数，重要参数不要随便修改
var (
	AllowUserExec = [][]byte{ExecerNone}
	//这里又限制了一次，因为挖矿的合约不会太多，所以这里配置死了，如果要扩展，需要改这里的代码
	AllowDepositExec = [][]byte{[]byte("ticket")}

	GenesisAddr            = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
	GenesisBlockTime int64 = 1526486816
	HotkeyAddr             = "12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
	FundKeyAddr            = "1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	EmptyValue             = []byte("FFFFFFFFemptyBVBiCj5jvE15pEiwro8TQRGnJSNsJF") //这字符串表示数据库中的空值
	SuperManager           = []string{"1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"}
	TokenApprs             = []string{}
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
	title string
	mu    sync.Mutex
)

var titles = map[string]bool{}

func Init(t string, cfg *Config) {
	mu.Lock()
	defer mu.Unlock()
	//每个title 只会初始化一次
	if titles[t] {
		title = t
		return
	}
	titles[t] = true
	title = t
	if cfg != nil {
		if isLocal() {
			setTestNet(true)
		} else {
			setTestNet(cfg.TestNet)
		}
		if cfg.Exec.MinExecFee > cfg.MemPool.MinTxFee || cfg.MemPool.MinTxFee > cfg.Wallet.MinFee {
			panic("config must meet: wallet.minFee >= mempool.minTxFee >= exec.minExecFee")
		}
		setMinFee(cfg.Exec.MinExecFee)
		setChainConfig("FixTime", cfg.FixTime)
	}
	//local 只用于单元测试
	if isLocal() {
		initChainTestNet()
		setLocalFork()
		setChainConfig("TxHeight", true)
		setChainConfig("Debug", true)
		return
	}
	//如果para 没有配置fork，那么默认所有的fork 为 0（一般只用于测试）
	if isPara() && (cfg == nil || cfg.Fork == nil || cfg.Fork.System == nil) {
		//keep superManager same with mainnet
		setForkForPara(title)
		return
	}
	if cfg != nil && cfg.Fork != nil {
		InitForkConfig(title, cfg.Fork)
	}
}

func GetTitle() string {
	mu.Lock()
	defer mu.Unlock()
	return title
}

func isLocal() bool {
	return title == "local"
}

func IsLocal() bool {
	mu.Lock()
	defer mu.Unlock()
	return IsLocal()
}

func SetMinFee(fee int64) {
	mu.Lock()
	defer mu.Unlock()
	setMinFee(fee)
}

func isPara() bool {
	//user.p.guodun.
	return strings.Count(title, ".") == 3 && strings.HasPrefix(title, ParaKeyX)
}

func IsPara() bool {
	mu.Lock()
	defer mu.Unlock()
	return isPara()
}

func IsParaExecName(name string) bool {
	return strings.HasPrefix(name, ParaKeyX)
}

func setTestNet(isTestNet bool) {
	if !isTestNet {
		setChainConfig("TestNet", false)
		return
	}
	setChainConfig("TestNet", true)
	//const 初始化TestNet 的初始化参数
	GenesisBlockTime = 1514533394
	FundKeyAddr = "1BQXS6TxaYYG5mADaWij4AxhZZUTpw95a5"
	SuperManager = []string{"1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S", "12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv", "1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK"}
	TokenApprs = []string{
		"1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S",
		"1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK",
		"1LY8GFia5EiyoTodMLfkB5PHNNpXRqxhyB",
		"1GCzJDS6HbgTQ2emade7mEJGGWFfA15pS9",
		"1JYB8sxi4He5pZWHCd3Zi2nypQ4JMB6AxN",
		"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv",
	}
}

func IsTestNet() bool {
	return IsEnable("TestNet")
}

func setMinFee(fee int64) {
	if fee < 0 {
		panic("fee less than zero")
	}
	setChainConfig("MinFee", fee)
	setChainConfig("MinBalanceTransfer", fee*10)
}

func GetParaName() string {
	if IsPara() {
		return GetTitle()
	}
	return ""
}

func FlagKV(key []byte, value int64) *KeyValue {
	return &KeyValue{Key: key, Value: Encode(&Int64{Data: value})}
}
