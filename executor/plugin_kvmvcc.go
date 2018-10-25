package executor

import "gitlab.33.cn/chain33/chain33/types"

func init() {
	RegisterPlugin("mvcc", &mvccPlugin{})
}

type mvccPlugin struct {
	*pluginBase
}

func (p *mvccPlugin) CheckEnable(executor *executor, enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	kvs, ok, err = p.checkFlag(executor, types.FlagKeyMVCC, enable)
	if err == types.ErrDBFlag {
		panic("mvcc config is enable, it must be synchronized from 0 height ")
	}
	return kvs, ok, err
}

func (p *mvccPlugin) ExecLocal(executor *executor, data *types.BlockDetail) (kvs []*types.KeyValue, err error) {
	kvs = AddMVCC(executor.localDB, data)
	for _, kv := range kvs {
		executor.localDB.Set(kv.Key, kv.Value)
	}
	return kvs, nil
}

func (p *mvccPlugin) ExecDelLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error) {
	kvs := DelMVCC(executor.localDB, data)
	return kvs, nil
}
