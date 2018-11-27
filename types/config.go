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
	//这里又限制了一次，因为挖矿的合约不会太多，所以这里配置死了，如果要扩展，需要改这里的代码
	AllowDepositExec = [][]byte{[]byte("ticket")}
	EmptyValue       = []byte("FFFFFFFFemptyBVBiCj5jvE15pEiwro8TQRGnJSNsJF") //这字符串表示数据库中的空值
	title            string
	mu               sync.Mutex
	titles           = map[string]bool{}
	chainConfig      = make(map[string]interface{})
	mver             = make(map[string]*mversion)
)

// coin conversation
const (
	Coin            int64 = 1e8
	MaxCoin         int64 = 9e18
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
		tlog.Error("mver config " + title + " not found")
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

func GInt(name string) int64 {
	value, err := G(name)
	if err != nil {
		return 0
	}
	return parseInt(value)
}

func MGInt(name string, height int64) int64 {
	value, err := MG(name, height)
	if err != nil {
		return 0
	}
	return parseInt(value)
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

func HasConf(key string) bool {
	mu.Lock()
	defer mu.Unlock()
	_, ok := chainConfig[key]
	return ok
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

func ConfSub(name string) *ConfQuery {
	return Conf("config.exec.sub." + name)
}

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

func (query *ConfQuery) GStrList(key string) []string {
	data, err := query.G(key)
	if err == nil {
		return parseStrList(data)
	}
	return []string{}
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

func (query *ConfQuery) MGStrList(key string, height int64) []string {
	data, err := query.MG(key, height)
	if err == nil {
		return parseStrList(data)
	}
	return []string{}
}

func (query *ConfQuery) MIsEnable(key string, height int64) bool {
	return MIsEnable(getkey(query.prefix, key), height)
}
