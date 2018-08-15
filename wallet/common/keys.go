package common

import "fmt"

const (
	keyAccount         = "Account"
	keyAddr            = "Addr"
	keyLabel           = "Label"
	keyTx              = "Tx"
	keyWalletFeeAmount = "WalletFeeAmount"
	keyWalletAutoMiner = "WalletAutoMiner"
	keyEncryptionFlag  = "EncryptionFlag"
	keyPasswordHash    = "PasswordHash"
	keyWalletSeed      = "walletseed"
)

//用于所有Account账户的输出list，需要安装时间排序
func CalcAccountKey(timestamp string, addr string) []byte {
	return []byte(fmt.Sprintf("%s:%s:%s", keyAccount, timestamp, addr))
}

//通过addr地址查询Account账户信息
func CalcAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("%s:%s", keyAddr, addr))
}

//通过label查询Account账户信息
func CalcLabelKey(label string) []byte {
	return []byte(fmt.Sprintf("%s:%s", keyLabel, label))
}

//通过height*100000+index 查询Tx交易信息
//key:Tx:height*100000+index
func CalcTxKey(key string) []byte {
	return []byte(fmt.Sprintf("%s:%s", keyTx, key))
}

func CalcEncryptionFlag() []byte {
	return []byte(keyEncryptionFlag)
}

func CalcPasswordHash() []byte {
	return []byte(keyPasswordHash)
}

func CalcWalletSeed() []byte {
	return []byte(keyWalletSeed)
}
