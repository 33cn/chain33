package privacybizpolicy

import "fmt"

const (
	Privacy4Addr     = "Privacy4Addr"
	AvailUTXOs       = "UTXO"
	FrozenUTXOs      = "FTXOs4Tx"
	PrivacySTXO      = "STXO"
	PrivacyTokenMap  = "PrivacyTokenMap"
	STXOs4Tx         = "STXOs4Tx"
	RevertSendtx     = "RevertSendtx"
	RecvPrivacyTx    = "RecvPrivacyTx"
	SendPrivacyTx    = "SendPrivacyTx"
	ScanPrivacyInput = "ScanPrivacyInput"
	ReScanUtxosFlag  = "ReScanUtxosFlag"
)

// calcUTXOKey 计算可用UTXO的健值,为输出交易哈希+输出索引位置
//key and prefix for privacy
//types.PrivacyDBStore的数据存储由calcUTXOKey生成key，
//1.当该utxo的目的地址是钱包管理的其中一个隐私地址时，该key作为value，保存在calcUTXOKey4TokenAddr由生成的key对应的kv中；
//2.当进行支付时，calcUTXOKey4TokenAddr对应的kv被删除，进而由calcPrivacyFUTXOKey生成的key对应kv中，其中平移的只是key，
// 本身的具体数据并不进行重新存储，即将utxo变化为futxo；
//3.当包含该交易的块得到确认时，如果发现输入包含在futxo中，则通过类似的方法，将其key设置到stxo中，
//4.当发生区块链分叉回退时，即发生del block的情况时，同时
// 4.a 当确认其中的输入存在于stxo时，则将其从stxo中转移至ftxo中，
// 4.b 当确认其中的输出存在于utxo或ftxo中时，则将其从utxo或ftxo中同时进行删除，同时删除types.PrivacyDBStore在数据库中的值
// 4.c 当确认其中的输出存在于stxo中时，则发生了异常，正常情况下，花费该笔utxo的交易需要被先回退，进而回退该笔交易，观察此种情况的发生
func calcUTXOKey(txhash string, index int) []byte {
	return []byte(fmt.Sprintf("%s-%s-%d", AvailUTXOs, txhash, index))
}

func calcKey4UTXOsSpentInTx(key string) []byte {
	return []byte(fmt.Sprintf("UTXOsSpentInTx:%s", key))
}

// calcPrivacyAddrKey 获取隐私账户私钥对保存在钱包中的索引串
func calcPrivacyAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("%s-%s", Privacy4Addr, addr))
}

//calcAddrKey 通过addr地址查询Account账户信息
func calcAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("Addr:%s", addr))
}

// calcPrivacyUTXOPrefix4Addr 获取指定地址下可用UTXO信息索引的KEY值前缀
func calcPrivacyUTXOPrefix4Addr(token, addr string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s-", AvailUTXOs, token, addr))
}

// calcFTXOsKeyPrefix 获取指定地址下由于交易未被确认而让交易使用到的UTXO处于冻结状态信息的KEY值前缀
func calcFTXOsKeyPrefix(token, addr string) []byte {
	var prefix string
	if len(token) > 0 && len(addr) > 0 {
		prefix = fmt.Sprintf("%s:%s-%s-", FrozenUTXOs, token, addr)
	} else if len(token) > 0 {
		prefix = fmt.Sprintf("%s:%s-", FrozenUTXOs, token)
	} else {
		prefix = fmt.Sprintf("%s:", FrozenUTXOs)
	}
	return []byte(prefix)
}

// calcSendPrivacyTxKey 计算以指定地址作为发送地址的交易信息索引
// addr为发送地址
// key为通过calcTxKey(heightstr)计算出来的值
func calcSendPrivacyTxKey(tokenname, addr, key string) []byte {
	return []byte(fmt.Sprintf("%s:%s-%s-%s", SendPrivacyTx, tokenname, addr, key))
}

// calcRecvPrivacyTxKey 计算以指定地址作为接收地址的交易信息索引
// addr为接收地址
// key为通过calcTxKey(heightstr)计算出来的值
func calcRecvPrivacyTxKey(tokenname, addr, key string) []byte {
	return []byte(fmt.Sprintf("%s:%s-%s-%s", RecvPrivacyTx, tokenname, addr, key))
}

// calcUTXOKey4TokenAddr 计算当前地址可用UTXO的Key健值
func calcUTXOKey4TokenAddr(token, addr, txhash string, index int) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s-%s-%d", AvailUTXOs, token, addr, txhash, index))
}

// calcKey4FTXOsInTx 交易构建以后,将可用UTXO冻结的健值
func calcKey4FTXOsInTx(token, addr, txhash string) []byte {
	return []byte(fmt.Sprintf("%s:%s-%s-%s", FrozenUTXOs, token, addr, txhash))
}

// calcRescanUtxosFlagKey 新账户导入时扫描区块上该地址相关的UTXO信息
func calcRescanUtxosFlagKey(addr string) []byte {
	return []byte(fmt.Sprintf("%s-%s", ReScanUtxosFlag, addr))
}

func calcScanPrivacyInputUTXOKey(txhash string, index int) []byte {
	return []byte(fmt.Sprintf("%s-%s-%d", ScanPrivacyInput, txhash, index))
}

func calcKey4STXOsInTx(txhash string) []byte {
	return []byte(fmt.Sprintf("%s:%s", STXOs4Tx, txhash))
}

// calcSTXOTokenAddrTxKey 计算当前地址已花费的UTXO
func calcSTXOTokenAddrTxKey(token, addr, txhash string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s-%s", PrivacySTXO, token, addr, txhash))
}

func calcSTXOPrefix4Addr(token, addr string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s", PrivacySTXO, token, addr))
}

// calcRevertSendTxKey 交易因为区块回退而将已经花费的UTXO移动到冻结UTXO队列的健值
func calcRevertSendTxKey(tokenname, addr, txhash string) []byte {
	return []byte(fmt.Sprintf("%s:%s-%s-%s", RevertSendtx, tokenname, addr, txhash))
}

//通过height*100000+index 查询Tx交易信息
//key:Tx:height*100000+index
func calcTxKey(key string) []byte {
	return []byte(fmt.Sprintf("Tx:%s", key))
}
