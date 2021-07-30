package executor

/*
 * 用户合约存取kv数据时，key值前缀需要满足一定规范
 * 即key = keyPrefix + userKey
 * 需要字段前缀查询时，使用’-‘作为分割符号
 */

var (
	//keyPrefixStateDB state db key必须前缀
	keyPrefixStateDB = "mavl-none-"
	//keyPrefixLocalDB local db的key必须前缀
	//keyPrefixLocalDB = "LODB-none-"
)

// delay transaction hash key
func formatDelayTxKey(txHash []byte) []byte {
	return append([]byte(keyPrefixStateDB), txHash...)
}
