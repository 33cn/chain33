package executor

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/common"
)

const (
	PrivacyOutputKeyPrefix  = "mavl-privacy-UTXO-tahi"
	PrivacyKeyImagePrefix   = "mavl-privacy-UTXO-keyimage"
	PrivacyUTXOKEYPrefix    = "LODB-privacy-UTXO-tahhi"
	PrivacyAmountTypePrefix = "LODB-privacy-UTXO-atype"
	PrivacyTokenTypesPrefix = "LODB-privacy-UTXO-token"
	KeyImageSpentAlready    = 0x01
	Invalid_index           = -1
)

//该key对应的是types.KeyOutput
//该kv会在store中设置
func CalcPrivacyOutputKey(token string, amount int64, txhash string, outindex int) (key []byte) {
	return []byte(fmt.Sprintf(PrivacyOutputKeyPrefix+"-%s-%d-%s-%d", token, amount, txhash, outindex))
}

func calcPrivacyKeyImageKey(token string, keyimage []byte) []byte {
	return []byte(fmt.Sprintf(PrivacyKeyImagePrefix+"-%s-%s", token, common.ToHex(keyimage)))
}

//在本地数据库中设置一条可以找到对应amount的对应的utxo的global index
func CalcPrivacyUTXOkeyHeight(token string, amount, height int64, txhash string, txindex, outindex int) (key []byte) {
	return []byte(fmt.Sprintf(PrivacyUTXOKEYPrefix+"-%s-%d-%d-%s-%d-%d", token, amount, height, txhash, txindex, outindex))
}

func CalcPrivacyUTXOkeyHeightPrefix(token string, amount int64) (key []byte) {
	return []byte(fmt.Sprintf(PrivacyUTXOKEYPrefix+"-%s-%d-", token, amount))
}

//设置当前系统存在的token的amount的类型，如存在1,3,5,100...等等的类型,
func CalcprivacyKeyTokenAmountType(token string) (key []byte) {
	return []byte(fmt.Sprintf(PrivacyAmountTypePrefix+"-%s-", token))
}

func CalcprivacyKeyTokenTypes() (key []byte) {
	return []byte(PrivacyTokenTypesPrefix)
}
