package util

import (
	"fmt"
	"testing"

	"github.com/33cn/chain33/types"
)

const length = 1e4
const repeat = 2

func BenchmarkDelDupKeyOld(b *testing.B) {
	kvs := make([]*types.KeyValue, 0, repeat*length)
	for r := 0; r < repeat; r++ {
		for i := 0; i < length; i++ {
			kvs = append(kvs, &types.KeyValue{Key: []byte(fmt.Sprintf("12kezKA11ZDcEuYukFV43oKqX89koycuHY:0x0dfed23208a15efe272d5095cce5fbc9a74ffa79c157564e239f6112e6eb6974:0000000000:test%d", i))})
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DelDupKeyOld(kvs)
	}
}

func BenchmarkDelDupKeyNew(b *testing.B) {
	kvs := make([]*types.KeyValue, 0, repeat*length)
	for r := 0; r < repeat; r++ {
		for i := 0; i < length; i++ {
			kvs = append(kvs, &types.KeyValue{Key: []byte(fmt.Sprintf("12kezKA11ZDcEuYukFV43oKqX89koycuHY:0x0dfed23208a15efe272d5095cce5fbc9a74ffa79c157564e239f6112e6eb6974:0000000000:test%d", i))})
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DelDupKey(kvs)
	}
}

// DelDupKey 删除重复的key
func DelDupKeyOld(kvs []*types.KeyValue) []*types.KeyValue {
	dupindex := make(map[string]int)
	n := 0
	for _, kv := range kvs {
		skey := string(kv.Key)
		if index, ok := dupindex[skey]; ok {
			//重复的key 替换老的key
			kvs[index] = kv
		} else {
			dupindex[skey] = n
			kvs[n] = kv
			n++
		}
	}
	return kvs[0:n]
}
