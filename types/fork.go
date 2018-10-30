package types

/*
出于forks 过程安全的考虑，比如代码更新，出现了新的fork，旧的链只要不明确指定 fork的高度，那么默认fork高度为 MaxHeight
也就是新的代码默认不会被启用，知道使用的人明确指定了fork的高度
*/
const MaxHeight = 10000000000000000

var systemFork = &Forks{}

func init() {
	//先要初始化
	SetTestNetFork()
	SetBityuanFork()
	SetLocalFork()
}

type Forks struct {
	forks map[string]map[string]int64
}

func (f *Forks) SetFork(title, key string, height int64) {
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

func (f *Forks) ReplaceFork(title, key string, height int64) {
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

//如果不存在，那么fork高度为0
func (f *Forks) GetFork(title, key string) int64 {
	forkitem, ok := f.forks[title]
	if !ok {
		return MaxHeight
	}
	height, ok := forkitem[key]
	if !ok {
		return MaxHeight
	}
	return height
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

func (f *Forks) CloneZero(from, to string) {
	f.Clone(from, to)
	f.SetAllFork(to, 0)
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

func (f *Forks) IsFork(title string, height int64, fork string) bool {
	ifork := f.GetFork(title, fork)
	//fork 不存在，默认跑最新代码，这样只要不设置就跑最新代码
	if height == -1 || height >= ifork {
		return true
	}
	return false
}

//default hard fork block height for bityuan real network
func SetBityuanFork() {
	systemFork.CloneZero("chain33", "bityuan")
	systemFork.ReplaceFork("bityuan", "ForkBlockHash", 1)
	systemFork.ReplaceFork("bityuan", "ForkV11ManageExec", 100000)
	systemFork.ReplaceFork("bityuan", "ForkV12TransferExec", 100000)
	systemFork.ReplaceFork("bityuan", "ForkV13ExecKey", 200000)
	systemFork.ReplaceFork("bityuan", "ForkV14TxGroup", 200000)
	systemFork.ReplaceFork("bityuan", "ForkV15ResetTx0", 200000)
	systemFork.ReplaceFork("bityuan", "ForkV16Withdraw", 200000)
	systemFork.ReplaceFork("bityuan", "ForkV17EVM", 250000)
	systemFork.ReplaceFork("bityuan", "ForkV18Relay", 500000)
	systemFork.ReplaceFork("bityuan", "ForkV19TokenPrice", 300000)
	systemFork.ReplaceFork("bityuan", "ForkV20EVMState", 350000)
	systemFork.ReplaceFork("bityuan", "ForkV21Privacy", MaxHeight)
	systemFork.ReplaceFork("bityuan", "ForkV22ExecRollback", 450000)
	systemFork.ReplaceFork("bityuan", "ForkV23TxHeight", MaxHeight)
	systemFork.ReplaceFork("bityuan", "ForkV24TxGroupPara", MaxHeight)
	systemFork.ReplaceFork("bityuan", "ForkV25BlackWhite", MaxHeight)
	systemFork.ReplaceFork("bityuan", "ForkV25BlackWhiteV2", MaxHeight)
	systemFork.ReplaceFork("bityuan", "ForkV26EVMKVHash", MaxHeight)
	systemFork.ReplaceFork("bityuan", "ForkV27TradeAsset", MaxHeight)
}

//bityuan test net fork
func SetTestNetFork() {
	systemFork.SetFork("chain33", "ForkV1", 75260)
	systemFork.SetFork("chain33", "ForkV2AddToken", 100899)
	systemFork.SetFork("chain33", "ForkV3", 110000)
	systemFork.SetFork("chain33", "ForkV4AddManage", 120000)
	systemFork.SetFork("chain33", "ForkV5Retrive", 180000)
	systemFork.SetFork("chain33", "ForkV6TokenBlackList", 190000)
	systemFork.SetFork("chain33", "ForkV7BadTokenSymbol", 184000)
	systemFork.SetFork("chain33", "ForkBlockHash", 208986+200)
	systemFork.SetFork("chain33", "ForkV9", 350000)
	systemFork.SetFork("chain33", "ForkV10TradeBuyLimit", 301000)
	systemFork.SetFork("chain33", "ForkV11ManageExec", 400000)
	systemFork.SetFork("chain33", "ForkV12TransferExec", 408400)
	systemFork.SetFork("chain33", "ForkV13ExecKey", 408400)
	systemFork.SetFork("chain33", "ForkV14TxGroup", 408400)
	systemFork.SetFork("chain33", "ForkV15ResetTx0", 453400)
	systemFork.SetFork("chain33", "ForkV16Withdraw", 480000)
	systemFork.SetFork("chain33", "ForkV17EVM", 500000)
	systemFork.SetFork("chain33", "ForkV18Relay", 570000)
	systemFork.SetFork("chain33", "ForkV19TokenPrice", 560000)
	systemFork.SetFork("chain33", "ForkV20EVMState", 650000)
	systemFork.SetFork("chain33", "ForkV21Privacy", 980000)
	systemFork.SetFork("chain33", "ForkV22ExecRollback", 706531)
	systemFork.SetFork("chain33", "ForkV23TxHeight", 806578)
	systemFork.SetFork("chain33", "ForkV24TxGroupPara", 806578)
	systemFork.SetFork("chain33", "ForkV25BlackWhite", 850000)
	systemFork.SetFork("chain33", "ForkV25BlackWhiteV2", 900000)
	systemFork.SetFork("chain33", "ForkV26EVMKVHash", 1000000)
	systemFork.SetFork("chain33", "ForkV27TradeAsset", 1010000)
}

func SetLocalFork() {
	systemFork.CloneZero("chain33", "local")
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

func IsEnableFork(height int64, fork string, enable bool) bool {
	if !enable {
		return false
	}
	return IsFork(height, fork)
}
