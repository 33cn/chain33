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

	tml "github.com/BurntSushi/toml"
)

//区块链共识相关的参数，重要参数不要随便修改
var (
	//AllowUserExec = [][]byte{ExecerNone}
	//挖矿的合约名单，适配旧配置，默认ticket
	//minerExecs  = []string{"ticket"}
	EmptyValue  = []byte("FFFFFFFFemptyBVBiCj5jvE15pEiwro8TQRGnJSNsJF") //这字符串表示数据库中的空值
	//title       string
	//mu          sync.Mutex
	//titles      = map[string]bool{}
	//chainConfig = make(map[string]interface{})
	//mver        = make(map[string]*mversion)
	//coinSymbol  = "bty"
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

// TODO 查看下是否可以直接删除
//func init() {
//	S("TestNet", false)
//	SetMinFee(1e5)
//	for key, cfg := range chaincfg.LoadAll() {
//		S("cfg."+key, cfg)
//	}
//	//防止报error 错误，不影响功能
//	if !HasConf("cfg.chain33") {
//		S("cfg.chain33", "")
//	}
//	if !HasConf("cfg.local") {
//		S("cfg.local", "")
//	}
//}

type Chain33Config struct {
	AllowUserExec [][]byte
	minerExecs    []string
	title         string
	mu            sync.Mutex
	chainConfig   map[string]interface{}
	mver          *mversion
	coinSymbol    string
	forks         *Forks
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

func NewChain33Config(cfgstring string) *Chain33Config {
	cfg, _ := InitCfgString(cfgstring)
	chain33Cfg := &Chain33Config{
		AllowUserExec: [][]byte{ExecerNone},
		minerExecs: []string{"ticket"},     //挖矿的合约名单，适配旧配置，默认ticket
		coinSymbol: "bty",
		forks: &Forks{},
	}
	chain33Cfg.setFlatConfig(cfgstring)
	chain33Cfg.setMver(cfg.Title, cfgstring)
	chain33Cfg.chainParamInit(cfg.Title, cfg)
	return chain33Cfg
}

func (c *Chain33Config) setFlatConfig(cfgstring string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cfg := make(map[string]interface{})
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		panic(err)
	}
	flat := FlatConfig(cfg)
	for k, v := range flat {
		c.setChainConfig("config."+k, v)
	}
}

func (c *Chain33Config) setChainConfig(key string, value interface{}) {
	c.chainConfig[key] = value
}

func (c *Chain33Config) getChainConfig(key string) (value interface{}, err error) {
	if data, ok := c.chainConfig[key]; ok {
		return data, nil
	}
	//报错警告
	tlog.Error("chain config " + key + " not found")
	return nil, ErrNotFound
}

// Init 初始化
func (c *Chain33Config) chainParamInit(t string, cfg *Config) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.title = t

	if cfg != nil {
		if c.isLocal() {
			c.setTestNet(true)
		} else {
			c.setTestNet(cfg.TestNet)
		}
		if cfg.Exec.MinExecFee > cfg.Mempool.MinTxFee || cfg.Mempool.MinTxFee > cfg.Wallet.MinFee {
			panic("config must meet: wallet.minFee >= mempool.minTxFee >= exec.minExecFee")
		}
		if cfg.Exec.MaxExecFee < cfg.Mempool.MaxTxFee {
			panic("config must meet: mempool.maxTxFee <= exec.maxExecFee")
		}
		if cfg.Consensus != nil {
			c.setMinerExecs(cfg.Consensus.MinerExecs)
		}
		if cfg.Exec != nil {
			c.setMinFee(cfg.Exec.MinExecFee)
		}
		c.setChainConfig("FixTime", cfg.FixTime)
		if cfg.Exec.MaxExecFee > 0 {
			c.setChainConfig("MaxFee", cfg.Exec.MaxExecFee)
		}
		if cfg.CoinSymbol != "" {
			if strings.Contains(cfg.CoinSymbol, "-") {
				panic("config CoinSymbol must without '-'")
			}
			c.coinSymbol = cfg.CoinSymbol
		} else {
			if c.isPara() {
				panic("must config CoinSymbol in para chain")
			} else {
				c.coinSymbol = DefaultCoinsSymbol
			}
		}
	}
	//local 只用于单元测试
	if c.isLocal() {
		c.forks.setLocalFork()
		c.setChainConfig("TxHeight", true)
		c.setChainConfig("Debug", true)
		//更新fork配置信息
		if c.mver != nil {
			c.mver.UpdateFork(c.forks)
		}
		return
	}
	//如果para 没有配置fork，那么默认所有的fork 为 0（一般只用于测试）
	if c.isPara() && (cfg == nil || cfg.Fork == nil || cfg.Fork.System == nil) {
		//keep superManager same with mainnet
		if !cfg.EnableParaFork {
			c.forks.setForkForParaZero(c.title)
		}
		if c.mver != nil {
			c.mver.UpdateFork(c.forks)
		}
		return
	}
	if cfg != nil && cfg.Fork != nil {
		c.initForkConfig(c.title, cfg.Fork)
	}
	if c.mver != nil {
		c.mver.UpdateFork(c.forks)
	}
}

func (c *Chain33Config) setTestNet(isTestNet bool) {
	if !isTestNet {
		c.setChainConfig("TestNet", false)
		return
	}
	c. setChainConfig("TestNet", true)
	//const 初始化TestNet 的初始化参数
}

func (c *Chain33Config) SetTestNetFork() {
	c.forks.SetFork("chain33", "ForkChainParamV1", 110000)
	c.forks.SetFork("chain33", "ForkChainParamV2", 1692674)
	c.forks.SetFork("chain33", "ForkCheckTxDup", 75260)
	c.forks.SetFork("chain33", "ForkBlockHash", 209186)
	c.forks.SetFork("chain33", "ForkMinerTime", 350000)
	c.forks.SetFork("chain33", "ForkTransferExec", 408400)
	c.forks.SetFork("chain33", "ForkExecKey", 408400)
	c.forks.SetFork("chain33", "ForkWithdraw", 480000)
	c.forks.SetFork("chain33", "ForkTxGroup", 408400)
	c.forks.SetFork("chain33", "ForkResetTx0", 453400)
	c.forks.SetFork("chain33", "ForkExecRollback", 706531)
	c.forks.SetFork("chain33", "ForkTxHeight", 806578)
	c.forks.SetFork("chain33", "ForkCheckBlockTime", 1200000)
	c.forks.SetFork("chain33", "ForkMultiSignAddress", 1298600)
	c.forks.SetFork("chain33", "ForkStateDBSet", 1572391)
	c.forks.SetFork("chain33", "ForkBlockCheck", 1560000)
	c.forks.SetFork("chain33", "ForkLocalDBAccess", 1572391)
	c.forks.SetFork("chain33", "ForkTxGroupPara", 1687250)
	c.forks.SetFork("chain33", "ForkBase58AddressCheck", 1800000)
	//这个fork只影响平行链，注册类似user.p.x.exec的driver，新开的平行链设为0即可，老的平行链要设置新的高度
	c.forks.SetFork("chain33", "ForkEnableParaRegExec", 0)
	c.forks.SetFork("chain33", "ForkCacheDriver", 2580000)
	c.forks.SetFork("chain33", "ForkTicketFundAddrV1", 3350000)
}

// GetP 获取ChainParam
func GetP(height int64, cfg *Chain33Config) *ChainParam {
	conf := Conf("mver.consensus", cfg)
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
func (c *Chain33Config) GetMinerExecs() []string {
	return c.minerExecs
}

func (c *Chain33Config) setMinerExecs(execs []string) {
	if len(execs) > 0 {
		c.minerExecs = execs
	}
}

// GetFundAddr 获取基金账户地址
func (c *Chain33Config) GetFundAddr() string {
	return c.MGStr("mver.consensus.fundKeyAddr", 0)
}

// G 获取ChainConfig中的配置
func (c *Chain33Config) G(key string) (value interface{}, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, err = c.getChainConfig(key)
	return
}

// MG 获取mver config中的配置
func (c *Chain33Config) MG(key string, height int64) (value interface{}, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.mver == nil {
		panic("mver is nil")
	}
	return c.mver.Get(key, height)
}

// GStr 获取ChainConfig中的字符串格式
func (c *Chain33Config) GStr(name string) string {
	value, err := c.G(name)
	if err != nil {
		return ""
	}
	if i, ok := value.(string); ok {
		return i
	}
	return ""
}

// MGStr 获取mver config 中的字符串格式
func (c *Chain33Config) MGStr(name string, height int64) string {
	value, err := c.MG(name, height)
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
func (c *Chain33Config) GInt(name string) int64 {
	value, err := c.G(name)
	if err != nil {
		return 0
	}
	return parseInt(value)
}

// MGInt 解析mver config 配置
func (c *Chain33Config) MGInt(name string, height int64) int64 {
	value, err := c.MG(name, height)
	if err != nil {
		return 0
	}
	return parseInt(value)
}

// IsEnable 解析ChainConfig配置
func (c *Chain33Config) IsEnable(name string) bool {
	isenable, err := c.G(name)
	if err == nil && isenable.(bool) {
		return true
	}
	return false
}

// MIsEnable 解析mver config 配置
func (c *Chain33Config) MIsEnable(name string, height int64) bool {
	isenable, err := c.MG(name, height)
	if err == nil && isenable.(bool) {
		return true
	}
	return false
}

// HasConf 解析chainConfig配置
func (c *Chain33Config) HasConf(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.chainConfig[key]
	return ok
}

// S 设置chainConfig配置
func (c *Chain33Config) S(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.HasPrefix(key, "config.") {
		if !c.isLocal() && !c.isTestPara() { //only local and test para can modify for test
			panic("prefix config. is readonly")
		} else {
			tlog.Error("modify " + key + " is only for test")
		}
	}
	c.setChainConfig(key, value)
}

//SetTitleOnlyForTest set title only for test use
func (c *Chain33Config) SetTitleOnlyForTest(ti string) {
	c.mu.Lock()
	defer  c.mu.Unlock()
	c.title = ti

}

// GetTitle 获取title
func (c *Chain33Config) GetTitle() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.title
}

// GetCoinSymbol 获取 coin symbol
func (c *Chain33Config) GetCoinSymbol() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.coinSymbol
}

func (c *Chain33Config) isLocal() bool {
	return c.title == "local"
}

// IsLocal 是否locak title
func (c *Chain33Config) IsLocal() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isLocal()
}

// SetMinFee 设置最小费用
func (c *Chain33Config) SetMinFee(fee int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setMinFee(fee)
}

func (c *Chain33Config) setMinFee(fee int64) {
	if fee < 0 {
		panic("fee less than zero")
	}
	c.setChainConfig("MinFee", fee)
	c.setChainConfig("MaxFee", fee*10000)
	c.setChainConfig("MinBalanceTransfer", fee*10)
}


func (c *Chain33Config) isPara() bool {
	return strings.Count(c.title, ".") == 3 && strings.HasPrefix(c.title, ParaKeyX)
}

func (c *Chain33Config) isTestPara() bool {
	return strings.Count(c.title, ".") == 3 && strings.HasPrefix(c.title, ParaKeyX) && strings.HasSuffix(c.title, "test.")
}

// IsPara 是否平行链
func (c *Chain33Config) IsPara() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isPara()
}

// IsParaExecName 是否平行链执行器
func IsParaExecName(exec string) bool {
	return strings.HasPrefix(exec, ParaKeyX)
}

//IsMyParaExecName 是否是我的para链的执行器
func (c *Chain33Config) IsMyParaExecName(exec string) bool {
	return IsParaExecName(exec) && strings.HasPrefix(exec, c.GetTitle())
}

//IsSpecificParaExecName 是否是某一个平行链的执行器
func IsSpecificParaExecName(title, exec string) bool {
	return IsParaExecName(exec) && strings.HasPrefix(exec, title)
}

//GetParaExecTitleName 如果是平行链执行器，获取对应title
func GetParaExecTitleName(exec string) (string, bool) {
	if IsParaExecName(exec) {
		for i := len(ParaKey); i < len(exec); i++ {
			if exec[i] == '.' {
				return exec[:i+1], true
			}
		}
	}
	return "", false
}

// IsTestNet 是否测试链
func (c *Chain33Config) IsTestNet() bool {
	return c.IsEnable("TestNet")
}

// GetParaName 获取平行链name
func (c *Chain33Config) GetParaName() string {
	if c.IsPara() {
		return c.GetTitle()
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

func (c *Chain33Config) getDefCfg(cfg *Config) string {
	if cfg.Title == "" {
		panic("title is not set in cfg")
	}
	if c.HasConf("cfg." + cfg.Title) {
		return c.GStr("cfg." + cfg.Title)
	}
	return ""
}

func (c *Chain33Config) mergeCfg(cfgstring string) string {
	cfg, err := initCfgString(cfgstring)
	if err != nil {
		panic(err)
	}
	cfgdefault := c.getDefCfg(cfg)
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

func (c *Chain33Config) setMver(title string, cfgstring string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mver = newMversion(title, cfgstring)
}

// InitCfgString 初始化配置
func InitCfgString(cfgstring string) (*Config, *ConfigSubModule) {
	//cfgstring = c.mergeCfg(cfgstring) // TODO 是否可以去除
	//setFlatConfig(cfgstring)         // 将set的全部去除
	cfg, err := initCfgString(cfgstring)
	if err != nil {
		panic(err)
	}
	//setMver(cfg.Title, cfgstring)    // 将set的全部去除
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

func ReadFile(path string) string {
	return readFile(path)
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
	*Chain33Config
	prefix string
}

// Conf 配置
func Conf(prefix string, cfg *Chain33Config) *ConfQuery {
	if prefix == "" || (!strings.HasPrefix(prefix, "config.") && !strings.HasPrefix(prefix, "mver.")) {
		panic("ConfQuery must init buy prefix config. or mver.")
	}
	return &ConfQuery{Chain33Config: cfg, prefix: prefix}
}

// ConfSub 子模块配置
func ConfSub(name string, cfg *Chain33Config) *ConfQuery {
	return Conf("config.exec.sub." + name, cfg)
}

// G 获取指定key的配置信息
func (query *ConfQuery) Gq(key string) (interface{}, error) {
	return query.G(getkey(query.prefix, key))
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
func (query *ConfQuery) GIntq(key string) int64 {
	return query.GInt(getkey(query.prefix, key))
}

// GStr 解析string类型
func (query *ConfQuery) GStrq(key string) string {
	return query.GStr(getkey(query.prefix, key))
}

// IsEnable 解析bool类型
func (query *ConfQuery) IsEnableq(key string) bool {
	return query.IsEnable(getkey(query.prefix, key))
}

// MG 解析mversion
func (query *ConfQuery) MGq(key string, height int64) (interface{}, error) {
	return query.MG(getkey(query.prefix, key), height)
}

// MGInt 解析mversion int类型配置
func (query *ConfQuery) MGIntq(key string, height int64) int64 {
	return query.MGInt(getkey(query.prefix, key), height)
}

// MGStr 解析mversion string类型配置
func (query *ConfQuery) MGStrq(key string, height int64) string {
	return query.MGStr(getkey(query.prefix, key), height)
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
func (query *ConfQuery) MIsEnableq(key string, height int64) bool {
	return query.MIsEnable(getkey(query.prefix, key), height)
}
