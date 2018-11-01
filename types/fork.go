package types

import "strings"

/*
出于forks 过程安全的考虑，比如代码更新，出现了新的fork，旧的链只要不明确指定 fork的高度，那么默认fork高度为 MaxHeight
也就是新的代码默认不会被启用，知道使用的人明确指定了fork的高度
*/
const MaxHeight = 10000000000000000

var systemFork = &Forks{}

func init() {
	//先要初始化
	SetTestNetFork()
}

type Forks struct {
	forks map[string]map[string]int64
}

func checkKey(key string) {
	if strings.Contains(key, ".") {
		panic("name must not have dot")
	}
}

func (f *Forks) SetFork(title, key string, height int64) {
	checkKey(key)
	f.setFork(title, key, height)
}

func (f *Forks) ReplaceFork(title, key string, height int64) {
	checkKey(key)
	f.replaceFork(title, key, height)
}

func (f *Forks) SetDappFork(title, dapp, key string, height int64) {
	checkKey(key)
	f.setFork(title, dapp+"."+key, height)
}

func (f *Forks) ReplaceDappFork(title, dapp, key string, height int64) {
	checkKey(key)
	f.replaceFork(title, dapp+"."+key, height)
}

func (f *Forks) replaceFork(title, key string, height int64) {
	if f.forks == nil {
		f.forks = make(map[string]map[string]int64)
	}
	_, ok := f.forks[title]
	if !ok {
		panic("replace a not exist title " + title)
	}
	if _, ok := f.forks[title][key]; !ok {
		panic("replace a not exist key " + title + " " + key)
	}
	f.forks[title][key] = height
}

func (f *Forks) setFork(title, key string, height int64) {
	if f.forks == nil {
		f.forks = make(map[string]map[string]int64)
	}
	_, ok := f.forks[title]
	if !ok {
		f.forks[title] = make(map[string]int64)
	}
	if _, ok := f.forks[title][key]; ok {
		panic("set dup fork " + title + " " + key)
	}
	f.forks[title][key] = height
}

//如果不存在，那么fork高度为0
func (f *Forks) GetFork(title, key string) int64 {
	forkitem, ok := f.forks[title]
	if !ok {
		if title == "local" {
			panic("title not exisit -> " + title)
		}
		return MaxHeight
	}
	height, ok := forkitem[key]
	if !ok {
		if title == "local" {
			panic("key not exisit -> " + key)
		}
		return MaxHeight
	}
	return height
}

func (f *Forks) HasFork(title, key string) bool {
	forkitem, ok := f.forks[title]
	if !ok {
		return false
	}
	_, ok = forkitem[key]
	return ok
}

func (f *Forks) GetDappFork(title, app string, key string) int64 {
	return f.GetFork(title, app+"."+key)
}

func (f *Forks) Clone(from, to string) error {
	forkitem, ok := f.forks[from]
	if !ok {
		return ErrCloneForkFrom
	}
	_, ok = f.forks[to]
	if ok {
		return ErrCloneForkToExist
	}
	f.forks[to] = make(map[string]int64)
	for k, v := range forkitem {
		f.forks[to][k] = v
	}
	return nil
}

func (f *Forks) CloneZero(from, to string) error {
	err := f.Clone(from, to)
	if err != nil {
		return err
	}
	f.SetAllFork(to, 0)
	return nil
}

func (f *Forks) CloneMaxHeight(from, to string) {
	f.Clone(from, to)
	f.SetAllFork(to, MaxHeight)
}

func (f *Forks) SetAllFork(title string, height int64) {
	forkitem, ok := f.forks[title]
	if !ok {
		return
	}
	for k := range forkitem {
		forkitem[k] = height
	}
}

func (f *Forks) GetAll(title string) map[string]int64 {
	forkitem, ok := f.forks[title]
	if !ok {
		return nil
	}
	return forkitem
}

func (f *Forks) IsFork(title string, height int64, fork string) bool {
	ifork := f.GetFork(title, fork)
	if height == -1 || height >= ifork {
		return true
	}
	return false
}

func (f *Forks) IsDappFork(title string, height int64, dapp, fork string) bool {
	return f.IsFork(title, height, dapp+"."+fork)
}

//bityuan test net fork
func SetTestNetFork() {
	systemFork.SetFork("chain33", "ForkCheckTxDup", 75260)
	systemFork.SetFork("chain33", "ForkChainParamV1", 110000)
	systemFork.SetFork("chain33", "ForkBlockHash", 209186)
	systemFork.SetFork("chain33", "ForkMinerTime", 350000)
	systemFork.SetFork("chain33", "ForkTransferExec", 408400)
	systemFork.SetFork("chain33", "ForkExecKey", 408400)
	systemFork.SetFork("chain33", "ForkWithdraw", 480000)
	systemFork.SetFork("chain33", "ForkTxGroup", 408400)
	systemFork.SetFork("chain33", "ForkResetTx0", 453400)
	systemFork.SetFork("chain33", "ForkExecRollback", 706531)
	systemFork.SetFork("chain33", "ForkTxHeight", 806578)
	systemFork.SetFork("chain33", "ForkTxGroupPara", 806578)
}

func SetLocalFork() {
	err := systemFork.CloneZero("chain33", "local")
	if err != nil {
		panic(err)
	}
	systemFork.ReplaceFork("local", "ForkBlockHash", 1)
}

//paraName not used currently
func SetForkForPara(paraName string) {
	systemFork.CloneZero("chain33", paraName)
	systemFork.ReplaceFork(paraName, "ForkBlockHash", 1)
}

func IsFork(height int64, fork string) bool {
	return systemFork.IsFork(GetTitle(), height, fork)
}

func IsDappFork(height int64, dapp, fork string) bool {
	return systemFork.IsDappFork(GetTitle(), height, dapp, fork)
}

func GetDappFork(dapp, fork string) int64 {
	return systemFork.GetDappFork(GetTitle(), dapp, fork)
}

func SetDappFork(title, dapp, fork string, height int64) {
	systemFork.SetDappFork(title, dapp, fork, height)
}

func RegisterDappFork(dapp, fork string, height int64) {
	systemFork.SetDappFork("chain33", dapp, fork, height)
}

func GetFork(fork string) int64 {
	return systemFork.GetFork(GetTitle(), fork)
}

func HasFork(fork string) bool {
	return systemFork.HasFork("chain33", fork)
}

func IsEnableFork(height int64, fork string, enable bool) bool {
	if !enable {
		return false
	}
	return IsFork(height, fork)
}

//fork 设置规则：
//所有的fork都需要有明确的配置，不开启fork 配置为 -1
func InitForkConfig(title string, forks *ForkList) {
	if title == "chain33" { //chain33 fork is default set in code
		return
	}
	chain33fork := systemFork.GetAll("chain33")
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
				continue
			}
			if _, ok := forks.Sub[exec][name]; ok {
				continue
			}
		}
		s += "exec " + exec + " name " + name + " not config\n"
	}
	//配置检查没有问题后，开始设置配置
	for k, v := range forks.System {
		if v == -1 {
			v = MaxHeight
		}
		if !HasFork(k) {
			s += "system fork not exist : " + k + "\n"
		}
		systemFork.SetFork(title, k, v)
	}
	//重置allow exec 的权限，让他只限制在配置文件设置的
	AllowUserExec = [][]byte{ExecerNone}
	for dapp, forklist := range forks.Sub {
		AllowUserExec = append(AllowUserExec, []byte(dapp))
		for k, v := range forklist {
			if v == -1 {
				v = MaxHeight
			}
			if !HasFork(dapp + "." + k) {
				s += "exec fork not exist : exec = " + dapp + " key = " + k + "\n"
			}
			systemFork.SetDappFork(title, dapp, k, v)
		}
	}
	if len(s) > 0 {
		panic(s)
	}
}
