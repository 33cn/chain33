// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"strings"

	"github.com/33cn/chain33/common/address"
)

/*
MaxHeight 出于forks 过程安全的考虑，比如代码更新，出现了新的fork，旧的链只要不明确指定 fork的高度，那么默认fork高度为 MaxHeight
也就是新的代码默认不会被启用，直到使用的人明确指定了fork的高度
*/
const MaxHeight = 10000000000000000

// Forks fork分叉结构体
type Forks struct {
	forks map[string]int64
}

func checkKey(key string) {
	if strings.Contains(key, ".") {
		panic("name must not have dot")
	}
}

// SetFork 设置fork信息
func (f *Forks) SetFork(key string, height int64) {
	checkKey(key)
	f.setFork(key, height)
}

// ReplaceFork 替换fork信息
func (f *Forks) ReplaceFork(key string, height int64) {
	checkKey(key)
	f.replaceFork(key, height)
}

// SetDappFork 设置dapp的fork信息
func (f *Forks) SetDappFork(dapp, key string, height int64) {
	checkKey(key)
	f.setFork(dapp+"."+key, height)
}

// ReplaceDappFork 替换dapp的fork信息
func (f *Forks) ReplaceDappFork(dapp, key string, height int64) {
	checkKey(key)
	f.replaceFork(dapp+"."+key, height)
}

func (f *Forks) replaceFork(key string, height int64) {
	if f.forks == nil {
		f.forks = make(map[string]int64)
	}
	if _, ok := f.forks[key]; !ok {
		panic("replace a not exist key " + " " + key)
	}
	f.forks[key] = height
}

func (f *Forks) setFork(key string, height int64) {
	if f.forks == nil {
		f.forks = make(map[string]int64)
	}
	f.forks[key] = height
}

// GetFork 如果不存在，那么fork高度为0
func (f *Forks) GetFork(key string) int64 {
	height, ok := f.forks[key]
	if !ok {
		tlog.Error("get fork key not exisit -> " + key)
		return MaxHeight
	}
	return height
}

// HasFork fork信息是否存在
func (f *Forks) HasFork(key string) bool {
	_, ok := f.forks[key]
	return ok
}

// GetDappFork 获取dapp fork信息
func (f *Forks) GetDappFork(app string, key string) int64 {
	return f.GetFork(app + "." + key)
}

// SetAllFork 设置所有fork的高度
func (f *Forks) SetAllFork(height int64) {
	for k := range f.forks {
		f.forks[k] = height
	}
}

// GetAll 获取所有fork信息
func (f *Forks) GetAll() map[string]int64 {
	return f.forks
}

// IsFork 是否fork高度
func (f *Forks) IsFork(height int64, fork string) bool {
	ifork := f.GetFork(fork)
	if height == -1 || height >= ifork {
		return true
	}
	return false
}

// IsDappFork 是否dapp fork高度
func (f *Forks) IsDappFork(height int64, dapp, fork string) bool {
	return f.IsFork(height, dapp+"."+fork)
}

// RegisterSystemFork 注册系统分叉, 部分分叉高度设为测试网分叉值
func (f *Forks) RegisterSystemFork() {
	f.SetFork("ForkChainParamV1", 110000)
	f.SetFork("ForkChainParamV2", 1692674)
	f.SetFork("ForkCheckTxDup", 75260)
	f.SetFork("ForkBlockHash", 209186)
	f.SetFork("ForkMinerTime", 350000)
	f.SetFork("ForkTransferExec", 408400)
	f.SetFork("ForkExecKey", 408400)
	f.SetFork("ForkWithdraw", 480000)
	f.SetFork("ForkTxGroup", 408400)
	f.SetFork("ForkResetTx0", 453400)
	f.SetFork("ForkExecRollback", 706531)
	f.SetFork("ForkTxHeight", 806578)
	f.SetFork("ForkCheckBlockTime", 1200000)
	f.SetFork("ForkMultiSignAddress", 1298600)
	f.SetFork("ForkStateDBSet", 1572391)
	f.SetFork("ForkBlockCheck", 1560000)
	f.SetFork("ForkLocalDBAccess", 1572391)
	f.SetFork("ForkTxGroupPara", 1687250)
	f.SetFork("ForkBase58AddressCheck", 1800000)
	//这个fork只影响平行链，注册类似user.p.x.exec的driver，新开的平行链设为0即可，老的平行链要设置新的高度
	f.SetFork("ForkEnableParaRegExec", 0)
	f.SetFork("ForkCacheDriver", 2580000)
	f.SetFork("ForkTicketFundAddrV1", 3350000)
	f.SetFork("ForkRootHash", 4500000)
	f.SetFork(address.ForkFormatAddressKey, 0)
	f.SetFork(address.ForkEthAddressFormat, 0)
	f.setFork("ForkCheckEthTxSort", 0)
	f.setFork("ForkProxyExec", 0)
	f.setFork("ForkMaxTxFeeV1", 0)

}

func (f *Forks) setLocalFork() {
	f.SetAllFork(0)
	f.ReplaceFork("ForkBlockHash", 1)
	f.ReplaceFork("ForkRootHash", 1)
}

// paraName not used currently
func (f *Forks) setForkForParaZero() {
	f.SetAllFork(0)
	f.ReplaceFork("ForkBlockHash", 1)
	f.ReplaceFork("ForkRootHash", 1)
}

// IsFork 是否系统 fork高度
func (c *Chain33Config) IsFork(height int64, fork string) bool {
	return c.forks.IsFork(height, fork)
}

// IsDappFork 是否dapp fork高度
func (c *Chain33Config) IsDappFork(height int64, dapp, fork string) bool {
	return c.forks.IsDappFork(height, dapp, fork)
}

// GetDappFork 获取dapp fork高度
func (c *Chain33Config) GetDappFork(dapp, fork string) int64 {
	return c.forks.GetDappFork(dapp, fork)
}

// SetDappFork 设置dapp fork高度
func (c *Chain33Config) SetDappFork(dapp, fork string, height int64) {
	if c.needSetForkZero() {
		height = 0
		if fork == "ForkBlockHash" {
			height = 1
		}
	}
	c.forks.SetDappFork(dapp, fork, height)
}

// RegisterDappFork 注册dapp fork高度
func (c *Chain33Config) RegisterDappFork(dapp, fork string, height int64) {
	if c.needSetForkZero() {
		height = 0
		if fork == "ForkBlockHash" {
			height = 1
		}
	}
	c.forks.SetDappFork(dapp, fork, height)
}

// GetFork 获取系统fork高度
func (c *Chain33Config) GetFork(fork string) int64 {
	return c.forks.GetFork(fork)
}

// HasFork 是否有系统fork
func (c *Chain33Config) HasFork(fork string) bool {
	return c.forks.HasFork(fork)
}

// IsEnableFork 是否使能了fork
func (c *Chain33Config) IsEnableFork(height int64, fork string, enable bool) bool {
	if !enable {
		return false
	}
	return c.IsFork(height, fork)
}

// fork 设置规则：
// 所有的fork都需要有明确的配置，不开启fork 配置为 -1; forks即为从toml中读入文件
func (c *Chain33Config) initForkConfig(forks *ForkList) {
	chain33fork := c.forks.GetAll()
	if chain33fork == nil {
		panic("chain33 fork not init")
	}
	//开始判断chain33fork中的system部分是否已经设置
	s := ""
	for k := range chain33fork {
		if !strings.Contains(k, ".") {
			if _, ok := forks.System[k]; !ok {
				s += "system fork " + k + " not config\n"
			}
		}
	}
	for k := range chain33fork {
		forkname := strings.Split(k, ".")
		if len(forkname) == 1 {
			continue
		}
		if len(forkname) > 2 {
			panic("fork name has too many dot")
		}
		exec := forkname[0]
		name := forkname[1]
		if forks.Sub != nil {
			//如果这个执行器，用户没有enable
			if _, ok := forks.Sub[exec]; !ok {
				s += "exec " + exec + " not enable in config file\n"
				continue
			}
			if _, ok := forks.Sub[exec][name]; ok {
				continue
			}
		}
		s += "exec " + exec + " name " + name + " not config in config file\n"
	}
	//配置检查没有问题后，开始设置配置
	for k, v := range forks.System {
		if v == -1 {
			v = MaxHeight
		}
		if !c.HasFork(k) {
			s += "system fork not exist : " + k + "\n"
		}
		// 由于toml文件中保存的是新的fork所以需要替换已有的初始化的fork
		c.forks.SetFork(k, v)
	}
	//重置allow exec 的权限，让他只限制在配置文件设置的
	AllowUserExec = [][]byte{ExecerNone}
	for dapp, forklist := range forks.Sub {
		AllowUserExec = append(AllowUserExec, []byte(dapp))
		for k, v := range forklist {
			if v == -1 {
				v = MaxHeight
			}
			if !c.HasFork(dapp + "." + k) {
				s += "exec fork not exist in code : exec = " + dapp + " key = " + k + "\n"
			}
			// 由于toml文件中保存的是新的fork所以需要替换已有的初始化的fork
			c.forks.SetDappFork(dapp, k, v)
		}
	}
	if !c.disableCheckFork {
		if len(s) > 0 {
			panic(s)
		}
	}
}
