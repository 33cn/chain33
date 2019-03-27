// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/33cn/chain33/types/chaincfg"
	tml "github.com/BurntSushi/toml"
)

//区块链共识相关的参数，重要参数不要随便修改
var (
	AllowUserExec = [][]byte{ExecerNone}
	//挖矿的合约名单，适配旧配置，默认ticket
	minerExecs  = []string{"ticket"}
	EmptyValue  = []byte("FFFFFFFFemptyBVBiCj5jvE15pEiwro8TQRGnJSNsJF") //这字符串表示数据库中的空值
	title       string
	mu          sync.Mutex
	titles      = map[string]bool{}
	chainConfig = make(map[string]interface{})
	mver        = make(map[string]*mversion)
	coinSymbol  = "bty"
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

func init() {
	S("TestNet", false)
	SetMinFee(1e5)
	for key, cfg := range chaincfg.LoadAll() {
		S("cfg."+key, cfg)
	}
	//防止报error 错误，不影响功能
	if !HasConf("cfg.chain33") {
		S("cfg.chain33", "")
	}
	if !HasConf("cfg.local") {
		S("cfg.local", "")
	}
}

// ChainParam 结构体
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

// GetP 获取ChainParam
func GetP(height int64) *ChainParam {
	conf := Conf("mver.consensus")
	c := &ChainParam{}
	c.CoinDevFund = conf.MGInt("coinDevFund", height) * Coin
	c.CoinReward = conf.MGInt("coinReward", height) * Coin
	c.FutureBlockTime = conf.MGInt("futureBlockTime", height)
	c.TicketPrice = conf.MGInt("ticketPrice", height) * Coin
	c.TicketFrozenTime = conf.MGInt("ticketFrozenTime", height)
	c.TicketWithdrawTime = conf.MGInt("ticketWithdrawTime", height)
	c.TicketMinerWaitTime = conf.MGInt("ticketMinerWaitTime", height)
	c.MaxTxNumber = conf.MGInt("maxTxNumber", height)
	c.PowLimitBits = uint32(conf.MGInt("powLimitBits", height))
	c.TargetTimespan = time.Duration(conf.MGInt("targetTimespan", height)) * time.Second
	c.TargetTimePerBlock = time.Duration(conf.MGInt("targetTimePerBlock", height)) * time.Second
	c.RetargetAdjustmentFactor = conf.MGInt("retargetAdjustmentFactor", height)
	return c
}

// GetMinerExecs 获取挖矿的合约名单
func GetMinerExecs() []string {
	return minerExecs
}

func setMinerExecs(execs []string) {
	if len(execs) > 0 {
		minerExecs = execs
	}
}

// GetFundAddr 获取基金账户地址
func GetFundAddr() string {
	return MGStr("mver.consensus.fundKeyAddr", 0)
}

func setChainConfig(key string, value interface{}) {
	chainConfig[key] = value
}

func getChainConfig(key string) (value interface{}, err error) {
	if data, ok := chainConfig[key]; ok {
		return data, nil
	}
	//报错警告
	tlog.Error("chain config " + key + " not found")
	return nil, ErrNotFound
}

// G 获取ChainConfig中的配置
func G(key string) (value interface{}, err error) {
	mu.Lock()
	value, err = getChainConfig(key)
	mu.Unlock()
	return
}

// MG 获取mver config中的配置
func MG(key string, height int64) (value interface{}, err error) {
	mu.Lock()
	defer mu.Unlock()
	mymver, ok := mver[title]
	if !ok {
		tlog.Error("mver config " + title + " not found")
		return nil, ErrNotFound
	}
	return mymver.Get(key, height)
}

// GStr 获取ChainConfig中的字符串格式
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

// MGStr 获取mver config 中的字符串格式
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

func parseInt(value interface{}) int64 {
	if i, ok := value.(int64); ok {
		return i
	}
	if s, ok := value.(string); ok {
		if strings.HasPrefix(s, "0x") {
			i, err := strconv.ParseUint(s, 0, 64)
			if err == nil {
				return int64(i)
			}
		}
	}
	return 0
}

// GInt 解析ChainConfig配置
func GInt(name string) int64 {
	value, err := G(name)
	if err != nil {
		return 0
	}
	return parseInt(value)
}

// MGInt 解析mver config 配置
func MGInt(name string, height int64) int64 {
	value, err := MG(name, height)
	if err != nil {
		return 0
	}
	return parseInt(value)
}

// IsEnable 解析ChainConfig配置
func IsEnable(name string) bool {
	isenable, err := G(name)
	if err == nil && isenable.(bool) {
		return true
	}
	return false
}

// MIsEnable 解析mver config 配置
func MIsEnable(name string, height int64) bool {
	isenable, err := MG(name, height)
	if err == nil && isenable.(bool) {
		return true
	}
	return false
}

// HasConf 解析chainConfig配置
func HasConf(key string) bool {
	mu.Lock()
	defer mu.Unlock()
	_, ok := chainConfig[key]
	return ok
}

// S 设置chainConfig配置
func S(key string, value interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if strings.HasPrefix(key, "config.") {
		if !isLocal() { //only local can modify for test
			panic("prefix config. is readonly")
		} else {
			tlog.Error("modify " + key + " is only for test")
		}
	}
	setChainConfig(key, value)
}

//SetTitleOnlyForTest set title only for test use
func SetTitleOnlyForTest(ti string) {
	mu.Lock()
	title = ti
	mu.Unlock()
}

// Init 初始化
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
		if cfg.Exec.MinExecFee > cfg.Mempool.MinTxFee || cfg.Mempool.MinTxFee > cfg.Wallet.MinFee {
			panic("config must meet: wallet.minFee >= mempool.minTxFee >= exec.minExecFee")
		}
		if cfg.Exec.MaxExecFee < cfg.Mempool.MaxTxFee {
			panic("config must meet: mempool.maxTxFee <= exec.maxExecFee")
		}
		setMinerExecs(cfg.Consensus.MinerExecs)
		setMinFee(cfg.Exec.MinExecFee)
		setChainConfig("FixTime", cfg.FixTime)
		if cfg.Exec.MaxExecFee > 0 {
			setChainConfig("MaxFee", cfg.Exec.MaxExecFee)
		}
		if cfg.CoinSymbol != "" {
			if strings.Contains(cfg.CoinSymbol, "-") {
				panic("config CoinSymbol must without '-'")
			}
			coinSymbol = cfg.CoinSymbol
		}
	}
	//local 只用于单元测试
	if isLocal() {
		setLocalFork()
		setChainConfig("TxHeight", true)
		setChainConfig("Debug", true)
		//更新fork配置信息
		if mver[title] != nil {
			mver[title].UpdateFork()
		}
		return
	}
	//如果para 没有配置fork，那么默认所有的fork 为 0（一般只用于测试）
	if isPara() && (cfg == nil || cfg.Fork == nil || cfg.Fork.System == nil) {
		//keep superManager same with mainnet
		setForkForPara(title)
		if mver[title] != nil {
			mver[title].UpdateFork()
		}
		return
	}
	if cfg != nil && cfg.Fork != nil {
		initForkConfig(title, cfg.Fork)
	}
	if mver[title] != nil {
		mver[title].UpdateFork()
	}
}

// GetTitle 获取title
func GetTitle() string {
	mu.Lock()
	s := title
	mu.Unlock()
	return s
}

// GetCoinSymbol 获取 coin symbol
func GetCoinSymbol() string {
	mu.Lock()
	s := coinSymbol
	mu.Unlock()
	return s
}

func isLocal() bool {
	return title == "local"
}

// IsLocal 是否locak title
func IsLocal() bool {
	mu.Lock()
	is := isLocal()
	mu.Unlock()
	return is
}

// SetMinFee 设置最小费用
func SetMinFee(fee int64) {
	mu.Lock()
	setMinFee(fee)
	mu.Unlock()
}

func isPara() bool {
	return strings.Count(title, ".") == 3 && strings.HasPrefix(title, ParaKeyX)
}

// IsPara 是否平行链
func IsPara() bool {
	mu.Lock()
	is := isPara()
	mu.Unlock()
	return is
}

// IsParaExecName 是否平行链执行器
func IsParaExecName(exec string) bool {
	return strings.HasPrefix(exec, ParaKeyX)
}

//IsMyParaExecName 是否是我的para链的执行器
func IsMyParaExecName(exec string) bool {
	return IsParaExecName(exec) && strings.HasPrefix(exec, GetTitle())
}

func setTestNet(isTestNet bool) {
	if !isTestNet {
		setChainConfig("TestNet", false)
		return
	}
	setChainConfig("TestNet", true)
	//const 初始化TestNet 的初始化参数
}

// IsTestNet 是否测试链
func IsTestNet() bool {
	return IsEnable("TestNet")
}

func setMinFee(fee int64) {
	if fee < 0 {
		panic("fee less than zero")
	}
	setChainConfig("MinFee", fee)
	setChainConfig("MaxFee", fee*10000)
	setChainConfig("MinBalanceTransfer", fee*10)
}

// GetParaName 获取平行链name
func GetParaName() string {
	if IsPara() {
		return GetTitle()
	}
	return ""
}

// FlagKV 获取kv对
func FlagKV(key []byte, value int64) *KeyValue {
	return &KeyValue{Key: key, Value: Encode(&Int64{Data: value})}
}

// MergeConfig Merge配置
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
	if HasConf("cfg." + cfg.Title) {
		return GStr("cfg." + cfg.Title)
	}
	return ""
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
	tml.NewEncoder(buf).Encode(conf)
	return buf.String()
}

func initCfgString(cfgstring string) (*Config, error) {
	var cfg Config
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// InitCfg 初始化配置
func InitCfg(path string) (*Config, *ConfigSubModule) {
	return InitCfgString(readFile(path))
}

func setFlatConfig(cfgstring string) {
	mu.Lock()
	cfg := make(map[string]interface{})
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		panic(err)
	}
	flat := FlatConfig(cfg)
	for k, v := range flat {
		setChainConfig("config."+k, v)
	}
	mu.Unlock()
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

// FlatConfig Flat配置
func FlatConfig(conf map[string]interface{}) map[string]interface{} {
	flat := make(map[string]interface{})
	flatConfig("", conf, flat)
	return flat
}

func setMver(title string, cfgstring string) {
	mu.Lock()
	mver[title] = newMversion(title, cfgstring)
	mu.Unlock()
}

// InitCfgString 初始化配置
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

// subModule 子模块结构体
type subModule struct {
	Store     map[string]interface{}
	Exec      map[string]interface{}
	Consensus map[string]interface{}
	Wallet    map[string]interface{}
	Mempool   map[string]interface{}
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
	subcfg.Mempool = parseItem(cfg.Mempool)
	return &subcfg, nil
}

//ModifySubConfig json data modify
func ModifySubConfig(sub []byte, key string, value interface{}) ([]byte, error) {
	var data map[string]interface{}
	err := json.Unmarshal(sub, &data)
	if err != nil {
		return nil, err
	}
	data[key] = value
	return json.Marshal(data)
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

// ConfQuery 结构体
type ConfQuery struct {
	prefix string
}

// Conf 配置
func Conf(prefix string) *ConfQuery {
	if prefix == "" || (!strings.HasPrefix(prefix, "config.") && !strings.HasPrefix(prefix, "mver.")) {
		panic("ConfQuery must init buy prefix config. or mver.")
	}
	return &ConfQuery{prefix}
}

// ConfSub 子模块配置
func ConfSub(name string) *ConfQuery {
	return Conf("config.exec.sub." + name)
}

// G 获取指定key的配置信息
func (query *ConfQuery) G(key string) (interface{}, error) {
	return G(getkey(query.prefix, key))
}

func parseStrList(data interface{}) []string {
	var list []string
	if item, ok := data.([]interface{}); ok {
		for i := 0; i < len(item); i++ {
			one, ok := item[i].(string)
			if ok {
				list = append(list, one)
			}
		}
	}
	return list
}

// GStrList 解析字符串列表
func (query *ConfQuery) GStrList(key string) []string {
	data, err := query.G(key)
	if err == nil {
		return parseStrList(data)
	}
	return []string{}
}

// GInt 解析int类型
func (query *ConfQuery) GInt(key string) int64 {
	return GInt(getkey(query.prefix, key))
}

// GStr 解析string类型
func (query *ConfQuery) GStr(key string) string {
	return GStr(getkey(query.prefix, key))
}

// IsEnable 解析bool类型
func (query *ConfQuery) IsEnable(key string) bool {
	return IsEnable(getkey(query.prefix, key))
}

// MG 解析mversion
func (query *ConfQuery) MG(key string, height int64) (interface{}, error) {
	return MG(getkey(query.prefix, key), height)
}

// MGInt 解析mversion int类型配置
func (query *ConfQuery) MGInt(key string, height int64) int64 {
	return MGInt(getkey(query.prefix, key), height)
}

// MGStr 解析mversion string类型配置
func (query *ConfQuery) MGStr(key string, height int64) string {
	return MGStr(getkey(query.prefix, key), height)
}

// MGStrList 解析mversion string list类型配置
func (query *ConfQuery) MGStrList(key string, height int64) []string {
	data, err := query.MG(key, height)
	if err == nil {
		return parseStrList(data)
	}
	return []string{}
}

// MIsEnable 解析mversion bool类型配置
func (query *ConfQuery) MIsEnable(key string, height int64) bool {
	return MIsEnable(getkey(query.prefix, key), height)
}
