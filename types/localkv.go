package types

import (
	"fmt"
	"time"
)

//CalcTxKey local db中保存交易的方法
func CalcTxKey(hash []byte) []byte {
	if IsEnable("quickIndex") {
		txhash := []byte("TX:")
		return append(txhash, hash...)
	}
	return hash
}

func CalcTxShortKey(hash []byte) []byte {
	txhash := []byte("STX:")
	return append(txhash, hash[0:8]...)
}

//用于存储地址相关的hash列表，key=TxAddrHash:addr:height*100000 + index
//地址下面所有的交易
func CalcTxAddrHashKey(addr string, heightindex string) []byte {
	return []byte(fmt.Sprintf("TxAddrHash:%s:%s", addr, heightindex))
}

//用于存储地址相关的hash列表，key=TxAddrHash:addr:flag:height*100000 + index
//地址下面某个分类的交易
func CalcTxAddrDirHashKey(addr string, flag int32, heightindex string) []byte {
	return []byte(fmt.Sprintf("TxAddrDirHash:%s:%d:%s", addr, flag, heightindex))
}

//存储地址参与的交易数量。add时加一，del时减一
func CalcAddrTxsCountKey(addr string) []byte {
	return []byte(fmt.Sprintf("AddrTxsCount:%s", addr))
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
