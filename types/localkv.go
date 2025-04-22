// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"fmt"

	"github.com/33cn/chain33/common/address"
)

// 定义key值
var (
	LocalPrefix            = []byte("LODB")
	FlagTxQuickIndex       = []byte("FLAG:FlagTxQuickIndex")
	FlagKeyMVCC            = []byte("FLAG:keyMVCCFlag")
	TxHashPerfix           = []byte("TX:")
	TxShortHashPerfix      = []byte("STX:")
	TxAddrHash             = []byte("TxAddrHash:")
	TxAddrDirHash          = []byte("TxAddrDirHash:")
	TxFeeAddrDirHash       = []byte("TxFeeAddrDirHash:")
	AddrTxsCount           = []byte("AddrTxsCount:")
	ConsensusParaTxsPrefix = []byte("LODBP:Consensus:Para:")            //存贮para共识模块从主链拉取的平行链交易
	FlagReduceLocaldb      = []byte("FLAG:ReduceLocaldb")               // 精简版localdb标记
	ReduceLocaldbHeight    = append(FlagReduceLocaldb, []byte(":H")...) // 精简版localdb高度
	EthTxHashPrefix        = []byte("ETX:")
)

// GetLocalDBKeyList 获取localdb的key列表
func GetLocalDBKeyList() [][]byte {
	return [][]byte{
		FlagTxQuickIndex, FlagKeyMVCC, TxHashPerfix, TxShortHashPerfix, FlagReduceLocaldb, EthTxHashPrefix,
	}
}

// CalcTxKey local db中保存交易的方法
func (c *Chain33Config) CalcTxKey(hash []byte) []byte {
	if c.IsEnable("quickIndex") {
		return append(TxHashPerfix, hash...)
	}
	return hash
}

// CalcEthTxKey local db中保存的eth 交易哈希key
func (c *Chain33Config) CalcEthTxKey(hash []byte) []byte {
	return append(EthTxHashPrefix, hash...)
}

// CalcTxKeyValue 保存local db中保存交易的方法
func (c *Chain33Config) CalcTxKeyValue(txr *TxResult) []byte {
	if c.IsEnable("reduceLocaldb") {
		txres := &TxResult{
			Height:     txr.GetHeight(),
			Index:      txr.GetIndex(),
			Blocktime:  txr.GetBlocktime(),
			ActionName: txr.GetActionName(),
		}
		return Encode(txres)
	}
	return Encode(txr)
}

// CalcTxShortKey local db中保存交易的方法
func CalcTxShortKey(hash []byte) []byte {
	return append(TxShortHashPerfix, hash[0:8]...)
}

// CalcTxAddrHashKey 用于存储地址相关的hash列表，key=TxAddrHash:addr:height*100000 + index
// 地址下面所有的交易
func CalcTxAddrHashKey(addr string, heightindex string) []byte {
	return append(TxAddrHash, []byte(fmt.Sprintf("%s:%s", address.FormatAddrKey(addr), heightindex))...)
}

// CalcTxAddrDirHashKey 用于存储地址相关的hash列表，key=TxAddrHash:addr:flag:height*100000 + index
// 地址下面某个分类的交易
func CalcTxAddrDirHashKey(addr string, flag int32, heightindex string) []byte {
	return append(TxAddrDirHash, []byte(fmt.Sprintf("%s:%d:%s", address.FormatAddrKey(addr), flag, heightindex))...)
}

// CalcTxFeeAddrDirHashKey 用于存储地址相关的hash列表，key=TxAddrHash:addr:flag:height*100000 + index
// 地址下面某个分类的交易
func CalcTxFeeAddrDirHashKey(addr string, flag int32, heightindex string) []byte {
	return append(TxFeeAddrDirHash, []byte(fmt.Sprintf("%s:%d:%s", address.FormatAddrKey(addr), flag, heightindex))...)
}

// CalcAddrTxsCountKey 存储地址参与的交易数量。add时加一，del时减一
func CalcAddrTxsCountKey(addr string) []byte {
	return append(AddrTxsCount, address.FormatAddrKey(addr)...)
}

// StatisticFlag 用于记录统计的key
func StatisticFlag() []byte {
	return []byte("Statistics:Flag")
}

// TotalFeeKey 统计所有费用的key
func TotalFeeKey(hash []byte) []byte {
	key := []byte("TotalFeeKey:")
	return append(key, hash...)
}

// CalcLocalPrefix 计算localdb key
func CalcLocalPrefix(execer []byte) []byte {
	s := append([]byte("LODB-"), execer...)
	s = append(s, byte('-'))
	return s
}

// CalcStatePrefix 计算localdb key
func CalcStatePrefix(execer []byte) []byte {
	s := append([]byte("mavl-"), execer...)
	s = append(s, byte('-'))
	return s
}

// CalcRollbackKey 计算回滚的key
func CalcRollbackKey(execer []byte, hash []byte) []byte {
	prefix := CalcLocalPrefix(execer)
	key := append(prefix, []byte("rollback-")...)
	key = append(key, hash...)
	return key
}

// CalcConsensusParaTxsKey 平行链localdb中保存的平行链title对应的交易
func CalcConsensusParaTxsKey(key []byte) []byte {
	return append(ConsensusParaTxsPrefix, key...)
}

// CheckConsensusParaTxsKey 检测para共识模块需要操作的平行链交易的key值
func CheckConsensusParaTxsKey(key []byte) bool {
	return bytes.HasPrefix(key, ConsensusParaTxsPrefix)
}
