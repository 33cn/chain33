// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
)

// 定义key值
var (
	LocalPrefix       = []byte("LODB")
	FlagTxQuickIndex  = []byte("FLAG:FlagTxQuickIndex")
	FlagKeyMVCC       = []byte("FLAG:keyMVCCFlag")
	TxHashPerfix      = []byte("TX:")
	TxShortHashPerfix = []byte("STX:")
	TxAddrHash        = []byte("TxAddrHash:")
	TxAddrDirHash     = []byte("TxAddrDirHash:")
	AddrTxsCount      = []byte("AddrTxsCount:")
)

// GetLocalDBKeyList 获取localdb的key列表
func GetLocalDBKeyList() [][]byte {
	return [][]byte{
		FlagTxQuickIndex, FlagKeyMVCC, TxHashPerfix, TxShortHashPerfix,
	}
}

//CalcTxKey local db中保存交易的方法
func CalcTxKey(hash []byte) []byte {
	if IsEnable("quickIndex") {
		return append(TxHashPerfix, hash...)
	}
	return hash
}

//CalcTxShortKey local db中保存交易的方法
func CalcTxShortKey(hash []byte) []byte {
	return append(TxShortHashPerfix, hash[0:8]...)
}

//CalcTxAddrHashKey 用于存储地址相关的hash列表，key=TxAddrHash:addr:height*100000 + index
//地址下面所有的交易
func CalcTxAddrHashKey(addr string, heightindex string) []byte {
	return append(TxAddrHash, []byte(fmt.Sprintf("%s:%s", addr, heightindex))...)
}

//CalcTxAddrDirHashKey 用于存储地址相关的hash列表，key=TxAddrHash:addr:flag:height*100000 + index
//地址下面某个分类的交易
func CalcTxAddrDirHashKey(addr string, flag int32, heightindex string) []byte {
	return append(TxAddrDirHash, []byte(fmt.Sprintf("%s:%d:%s", addr, flag, heightindex))...)
}

//CalcAddrTxsCountKey 存储地址参与的交易数量。add时加一，del时减一
func CalcAddrTxsCountKey(addr string) []byte {
	return append(AddrTxsCount, []byte(addr)...)
}

//StatisticFlag 用于记录统计的key
func StatisticFlag() []byte {
	return []byte("Statistics:Flag")
}

//TotalFeeKey 统计所有费用的key
func TotalFeeKey(hash []byte) []byte {
	key := []byte("TotalFeeKey:")
	return append(key, hash...)
}

//CalcLocalPrefix 计算localdb key
func CalcLocalPrefix(execer []byte) []byte {
	s := append([]byte("LODB-"), execer...)
	s = append(s, byte('-'))
	return s
}

//CalcStatePrefix 计算localdb key
func CalcStatePrefix(execer []byte) []byte {
	s := append([]byte("mavl-"), execer...)
	s = append(s, byte('-'))
	return s
}

//CalcRollbackKey 计算回滚的key
func CalcRollbackKey(execer []byte, hash []byte) []byte {
	prefix := CalcLocalPrefix(execer)
	key := append(prefix, []byte("rollback-")...)
	key = append(key, hash...)
	return key
}
