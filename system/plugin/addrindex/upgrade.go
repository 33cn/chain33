package addrindex

import (
	"fmt"

	plugins "github.com/33cn/chain33/system/plugin"
	"github.com/33cn/chain33/types"
)

/*
types.CalcAddrTxsCountKey(addr)
types.CalcTxAddrDirHashKey
types.CalcTxAddrHashKey
*/

// CalcAddrTxsCountPrefixOld 获得老的前缀
func CalcAddrTxsCountPrefixOld() []byte {
	return types.AddrTxsCount
}

// CalcTxAddrHashPrefixOld 获得老的前缀
func CalcTxAddrHashPrefixOld() []byte {
	return types.TxAddrHash
}

// CalcTxAddrDirHashPrefixOld 获得老的前缀
func CalcTxAddrDirHashPrefixOld() []byte {
	return types.TxAddrDirHash
}

// CalcTxAddrHashPrefix 用于存储地址相关的hash列表，key=TxAddrHash:addr:height*100000 + index
func CalcTxAddrHashPrefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, TxAddrHash))
}

// CalcTxAddrDirHashPrefix 用于存储地址相关的hash列表，key=TxAddrHash:addr:flag:height*100000 + index
func CalcTxAddrDirHashPrefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, TxAddrDirHash))
}

// CalcAddrTxsCountPrefix 存储地址参与的交易数量。add时加一，del时减一
func CalcAddrTxsCountPrefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, AddrTxsCount))
}

// Upgrade TODO 数量多, 需要测试不分批是否可以
func (p *addrindexPlugin) Upgrade(count int32) (bool, error) {
	toVersion := 2
	prefixes := []plugins.Prefixes{
		{CalcAddrTxsCountPrefixOld(), CalcAddrTxsCountPrefix(name)},
		{CalcTxAddrDirHashPrefixOld(), CalcTxAddrDirHashPrefix(name)},
		{CalcAddrTxsCountPrefixOld(), CalcAddrTxsCountPrefix(name)},
	}
	return plugins.Upgrade(p.GetLocalDB(), name, toVersion, prefixes, count)
}
