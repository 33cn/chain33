package executor

if exec.enableMVCC {
	kvs, err := exec.checkMVCCFlag(execute.localDB, datas)
	if err != nil {
		panic(err)
	}
	kvset.KV = append(kvset.KV, kvs...)
	kvs = AddMVCC(execute.localDB, datas)
	if kvs != nil {
		kvset.KV = append(kvset.KV, kvs...)
	}
	for _, kv := range kvset.KV {
		execute.localDB.Set(kv.Key, kv.Value)
	}
}
execute.enableMVCC()

if exec.enableMVCC {
	kvs := DelMVCC(execute.localDB, datas)
	if kvs != nil {
		kvset.KV = append(kvset.KV, kvs...)
	}
}