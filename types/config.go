package types

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	tml "github.com/BurntSushi/toml"
	"gitlab.33.cn/chain33/chain33/types/chaincfg"
)

var chainBaseParam *ChainParam
var chainV3Param *ChainParam
var chainConfig = make(map[string]interface{})
var mver = make(map[string]*mversion)

func init() {
	initChainBase()
	initChainBityuanV3()
	S("TestNet", false)
	SetMinFee(1e5)
	for key, cfg := range chaincfg.LoadAll() {
		S("cfg."+key, cfg)
	}
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
	chainV3Param.PowLimitBits = uint32(0x1f2fffff)
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

func getChainConfig(key string) (value interface{}, err error) {
	if data, ok := chainConfig[key]; ok {
		return data, nil
	}
	if isLocal() {
		tlog.Warn("chain config " + key + " not found")
	}
	return nil, ErrNotFound
}

func G(key string) (value interface{}, err error) {
	mu.Lock()
	defer mu.Unlock()
	return getChainConfig(key)
}

func MG(key string, height int64) (value interface{}, err error) {
	mu.Lock()
	defer mu.Unlock()
	mymver, ok := mver[title]
	if !ok {
		if isLocal() {
			tlog.Warn("mver config " + title + " not found")
		}
		return nil, ErrNotFound
	}
	return mymver.Get(key, height)
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

func MGStr(name string, height int64) string {
	value, err := MG(name, height)
	if err != nil {
		return ""
	}
	if i, ok := value.(string); ok {
		return i
	}
	return ""
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

func MGInt(name string, height int64) int64 {
	value, err := MG(name, height)
	if err != nil {
		return 0
	}
	if i, ok := value.(int64); ok {
		return i
	}
	return 0
}

func IsEnable(name string) bool {
	isenable, err := G(name)
	if err == nil && isenable.(bool) {
		return true
	}
	return false
}

func MIsEnable(name string, height int64) bool {
	isenable, err := MG(name, height)
	if err == nil && isenable.(bool) {
		return true
	}
	return false
}

func S(key string, value interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if strings.HasPrefix(key, "config.") {
		if isLocal() {
			panic("prefix config. is readonly")
		}
		return
	}
	setChainConfig(key, value)
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
	FundKeyAddr      = "1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	EmptyValue       = []byte("FFFFFFFFemptyBVBiCj5jvE15pEiwro8TQRGnJSNsJF") //这字符串表示数据库中的空值
	SuperManager     = []string{"1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"}
	TokenApprs       = []string{}
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
		tlog.Warn("title " + t + " only init once")
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
		//更新fork配置信息
		mver[title].UpdateFork()
		return
	}
	//如果para 没有配置fork，那么默认所有的fork 为 0（一般只用于测试）
	if isPara() && (cfg == nil || cfg.Fork == nil || cfg.Fork.System == nil) {
		//keep superManager same with mainnet
		setForkForPara(title)
		mver[title].UpdateFork()
		return
	}
	if cfg != nil && cfg.Fork != nil {
		InitForkConfig(title, cfg.Fork)
		mver[title].UpdateFork()
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
	return isLocal()
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

func MergeConfig(conf map[string]interface{}, def map[string]interface{}) string {
	errstr := checkConfig("", conf, def)
	if errstr != "" {
		return errstr
	}
	mergeConfig(conf, def)
	return ""
}

//检查默认配置文件
func checkConfig(key string, conf map[string]interface{}, def map[string]interface{}) string {
	errstr := ""
	for key1, value1 := range conf {
		if vdef, ok := def[key1]; ok {
			conf1, ok1 := value1.(map[string]interface{})
			def1, ok2 := vdef.(map[string]interface{})
			if ok1 && ok2 {
				errstr += checkConfig(getkey(key, key1), conf1, def1)
			} else {
				errstr += "rewrite defalut key " + getkey(key, key1) + "\n"
			}
		}
	}
	return errstr
}

func mergeConfig(conf map[string]interface{}, def map[string]interface{}) {
	for key1, value1 := range def {
		if vdef, ok := conf[key1]; ok {
			conf1, ok1 := value1.(map[string]interface{})
			def1, ok2 := vdef.(map[string]interface{})
			if ok1 && ok2 {
				mergeConfig(conf1, def1)
				conf[key1] = conf1
			}
		} else {
			conf[key1] = value1
		}
	}
}

func getkey(key, key1 string) string {
	if key == "" {
		return key1
	}
	return key + "." + key1
}

func getDefCfg(cfg *Config) string {
	if cfg.Title == "" {
		panic("title is not set in cfg")
	}
	return GStr("cfg." + cfg.Title)
}

func mergeCfg(cfgstring string) string {
	cfg, err := initCfgString(cfgstring)
	if err != nil {
		panic(err)
	}
	cfgdefault := getDefCfg(cfg)
	if cfgdefault != "" {
		return mergeCfgString(cfgstring, cfgdefault)
	}
	return cfgstring
}

func mergeCfgString(cfgstring, cfgdefault string) string {
	//1. defconfig
	def := make(map[string]interface{})
	_, err := tml.Decode(cfgdefault, &def)
	if err != nil {
		panic(err)
	}
	//2. userconfig
	conf := make(map[string]interface{})
	_, err = tml.Decode(cfgstring, &conf)
	if err != nil {
		panic(err)
	}
	errstr := MergeConfig(conf, def)
	if errstr != "" {
		panic(errstr)
	}
	buf := new(bytes.Buffer)
	err = tml.NewEncoder(buf).Encode(conf)
	return buf.String()
}

func initCfgString(cfgstring string) (*Config, error) {
	var cfg Config
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func InitCfg(path string) (*Config, *ConfigSubModule) {
	return InitCfgString(readFile(path))
}

func setFlatConfig(cfgstring string) {
	mu.Lock()
	defer mu.Unlock()
	cfg := make(map[string]interface{})
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		panic(err)
	}
	flat := FlatConfig(cfg)
	for k, v := range flat {
		setChainConfig("config."+k, v)
	}
}

func flatConfig(key string, conf map[string]interface{}, flat map[string]interface{}) {
	for key1, value1 := range conf {
		conf1, ok := value1.(map[string]interface{})
		if ok {
			flatConfig(getkey(key, key1), conf1, flat)
		} else {
			flat[getkey(key, key1)] = value1
		}
	}
}

func FlatConfig(conf map[string]interface{}) map[string]interface{} {
	flat := make(map[string]interface{})
	flatConfig("", conf, flat)
	return flat
}

func setMver(title string, cfgstring string) {
	mu.Lock()
	defer mu.Unlock()
	mver[title] = newMversion(title, cfgstring)
}

func InitCfgString(cfgstring string) (*Config, *ConfigSubModule) {
	cfgstring = mergeCfg(cfgstring)
	setFlatConfig(cfgstring)
	cfg, err := initCfgString(cfgstring)
	if err != nil {
		panic(err)
	}
	setMver(cfg.Title, cfgstring)
	sub, err := initSubModuleString(cfgstring)
	if err != nil {
		panic(err)
	}
	return cfg, sub
}

type subModule struct {
	Store     map[string]interface{}
	Exec      map[string]interface{}
	Consensus map[string]interface{}
	Wallet    map[string]interface{}
}

func readFile(path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func initSubModuleString(cfgstring string) (*ConfigSubModule, error) {
	var cfg subModule
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		return nil, err
	}
	return parseSubModule(&cfg)
}

func parseSubModule(cfg *subModule) (*ConfigSubModule, error) {
	var subcfg ConfigSubModule
	subcfg.Store = parseItem(cfg.Store)
	subcfg.Exec = parseItem(cfg.Exec)
	subcfg.Consensus = parseItem(cfg.Consensus)
	subcfg.Wallet = parseItem(cfg.Wallet)
	return &subcfg, nil
}

func parseItem(data map[string]interface{}) map[string][]byte {
	subconfig := make(map[string][]byte)
	if len(data) == 0 {
		return subconfig
	}
	for key := range data {
		if key == "sub" {
			subcfg := data[key].(map[string]interface{})
			for k := range subcfg {
				subconfig[k], _ = json.Marshal(subcfg[k])
			}
		}
	}
	return subconfig
}

type ConfQuery struct {
	prefix string
}

func Conf(prefix string) *ConfQuery {
	if prefix == "" || (!strings.HasPrefix(prefix, "config.") && !strings.HasPrefix(prefix, "mver.")) {
		panic("ConfQuery must init buy prefix config. or mver.")
	}
	return &ConfQuery{prefix}
}

func (query *ConfQuery) G(key string) (interface{}, error) {
	return G(getkey(query.prefix, key))
}

func (query *ConfQuery) GInt(key string) int64 {
	return GInt(getkey(query.prefix, key))
}

func (query *ConfQuery) GStr(key string) string {
	return GStr(getkey(query.prefix, key))
}

func (query *ConfQuery) IsEnable(key string) bool {
	return IsEnable(getkey(query.prefix, key))
}

func (query *ConfQuery) MG(key string, height int64) (interface{}, error) {
	return MG(getkey(query.prefix, key), height)
}

func (query *ConfQuery) MGInt(key string, height int64) int64 {
	return MGInt(getkey(query.prefix, key), height)
}

func (query *ConfQuery) MGStr(key string, height int64) string {
	return MGStr(getkey(query.prefix, key), height)
}

func (query *ConfQuery) MIsEnable(key string, height int64) bool {
	return MIsEnable(getkey(query.prefix, key), height)
}
