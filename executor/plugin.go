package executor

import (
	"sync/atomic"

	"gitlab.33.cn/chain33/chain33/types"
)

//plugin 主要用于处理 execlocal 和 execdellocal 时候的全局kv的处理
//每个插件都有插件是否开启这个插件的判断，如果不开启，执行的时候会被忽略

type plugin interface {
	CheckEnable() (ok bool, err error)
	ExecLocal(exec *Executor, executor *executor, data *types.BlockDetail) ([]types.KeyValue, error)
	ExecDelLocal(exec *Executor, executor *executor, data *types.BlockDetail) ([]types.KeyValue, error)
}

var globalPlugins = make(map[string]plugin)

func RegisterPlugin(name string, p plugin) {
	if _, ok := globalPlugins[name]; ok {
		panic("plugin exist " + name)
	}
	globalPlugins[name] = p
}

type pluginBase struct {
}

func loadFlag(localDB dbm.KVDB, key []byte) (int64, error) {
	flag := &types.Int64{}
	flagBytes, err := localDB.Get(key)
	if err == nil {
		err = types.Decode(flagBytes, flag)
		if err != nil {
			return 0, err
		}
		return flag.GetData(), nil
	} else if err == types.ErrNotFound {
		return 0, nil
	}
	return 0, err
}

func (exec *Executor) checkMVCCFlag(db dbm.KVDB, datas *types.BlockDetail) ([]*types.KeyValue, error) {
	//flag = 0 : init
	//flag = 1 : start from zero
	//flag = 2 : start from no zero //不允许flag = 2的情况
	b := datas.Block
	if atomic.LoadInt64(&exec.flagMVCC) == FlagInit {
		flag, err := loadFlag(db, types.FlagKeyMVCC)
		if err != nil {
			panic(err)
		}
		atomic.StoreInt64(&exec.flagMVCC, flag)
	}
	var kvset []*types.KeyValue
	if atomic.LoadInt64(&exec.flagMVCC) == FlagInit {
		if b.Height != 0 {
			atomic.StoreInt64(&exec.flagMVCC, FlagNotFromZero)
		} else {
			//区块为0, 写入标志
			if atomic.CompareAndSwapInt64(&exec.flagMVCC, FlagInit, FlagFromZero) {
				kvset = append(kvset, types.FlagKV(types.FlagKeyMVCC, FlagFromZero))
			}
		}
	}
	if atomic.LoadInt64(&exec.flagMVCC) != FlagFromZero {
		panic("config set enableMVCC=true, it must be synchronized from 0 height")
	}
	return kvset, nil
}

func (exec *Executor) stat(execute *executor, datas *types.BlockDetail) ([]*types.KeyValue, error) {
	// 开启数据统计，需要从0开始同步数据
	b := datas.Block
	if atomic.LoadInt64(&exec.enableStatFlag) == 0 {
		flag, err := loadFlag(execute.localDB, StatisticFlag())
		if err != nil {
			panic(err)
		}
		atomic.StoreInt64(&exec.enableStatFlag, flag)
	}
	if b.Height != 0 && atomic.LoadInt64(&exec.enableStatFlag) == 0 {
		elog.Error("chain33.toml enableStat = true, it must be synchronized from 0 height")
		panic("chain33.toml enableStat = true, it must be synchronized from 0 height")
	}
	// 初始状态置为开启状态
	var kvset []*types.KeyValue
	if atomic.CompareAndSwapInt64(&exec.enableStatFlag, 0, 1) {
		kvset = append(kvset, types.FlagKV(StatisticFlag(), 1))
	}
	kvs, err := countInfo(execute, datas)
	if err != nil {
		return nil, err
	}
	kvset = append(kvset, kvs.KV...)
	return kvset, nil
}
