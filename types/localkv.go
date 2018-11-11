// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
	"time"
)

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

func CalcTxShortKey(hash []byte) []byte {
	return append(TxShortHashPerfix, hash[0:8]...)
}

//用于存储地址相关的hash列表，key=TxAddrHash:addr:height*100000 + index
//地址下面所有的交易
func CalcTxAddrHashKey(addr string, heightindex string) []byte {
	return append(TxAddrHash, []byte(fmt.Sprintf("%s:%s", addr, heightindex))...)
}

//用于存储地址相关的hash列表，key=TxAddrHash:addr:flag:height*100000 + index
//地址下面某个分类的交易
func CalcTxAddrDirHashKey(addr string, flag int32, heightindex string) []byte {
	return append(TxAddrDirHash, []byte(fmt.Sprintf("%s:%d:%s", addr, flag, heightindex))...)
}

//存储地址参与的交易数量。add时加一，del时减一
func CalcAddrTxsCountKey(addr string) []byte {
	return append(AddrTxsCount, []byte(addr)...)
}

func StatisticFlag() []byte {
	return []byte("Statistics:Flag")
}

func StatisticTicketInfoKey(ticketId string) []byte {
	return []byte("Statistics:TicketInfo:TicketId:" + ticketId)
}

func StatisticTicketInfoOrderKey(minerAddr string, createTime int64, ticketId string) []byte {
	return []byte("Statistics:TicketInfoOrder:Addr:" + minerAddr + ":CreateTime:" + time.Unix(createTime, 0).Format("20060102150405") + ":TicketId:" + ticketId)
}

func StatisticTicketKey(minerAddr string) []byte {
	return []byte("Statistics:TicketStat:Addr:" + minerAddr)
}

func TotalFeeKey(hash []byte) []byte {
	key := []byte("TotalFeeKey:")
	return append(key, hash...)
}
