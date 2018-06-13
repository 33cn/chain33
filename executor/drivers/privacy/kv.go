package privacy

import (
	"fmt"
	"strconv"
	"strings"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	PrivacyOutputKeyPrefix  = "mavl-privacy-UTXO-tahi"
	PrivacyKeyImagePrefix   = "mavl-privacy-UTXO-keyimage"
	PrivacyUTXOKEYPrefix    = "local-privacy-UTXO-tahhi"
	PrivacyAmountTypePrefix = "local-privacy-UTXO-atype"
	PrivacyTokenTypesPrefix = "local-privacy-UTXO-token"
	KeyImageSpentAlready    = 0x01
	Invalid_index           = -1
)

//该key对应的是types.KeyOutput
//该kv会在store中设置
func CalcPrivacyOutputKey(token string, amount int64, txhash string, index int) (key []byte) {
	return []byte(fmt.Sprintf(PrivacyOutputKeyPrefix+"-%s-%d-%s-%d", token, amount, txhash, index))
}

func calcPrivacyKeyImageKey(token string, keyimage []byte) []byte {
	return []byte(fmt.Sprintf(PrivacyKeyImagePrefix+"-%s-%s", token, common.ToHex(keyimage)))
}

//在本地数据库中设置一条可以找到对应amount的对应的utxo的global index
func CalcPrivacyUTXOkeyHeight(token string, amount, height int64, txhash string, txindex, outindex int) (key []byte) {
	return []byte(CalcPrivacyUTXOkeyHeightStr(token, amount, height, txhash, txindex, outindex))
}

func CalcPrivacyUTXOkeyHeightStr(token string, amount, height int64, txhash string, txindex, outindex int) (key string) {
	return fmt.Sprintf(PrivacyUTXOKEYPrefix+"-%s-%d-%010d-%s-%d-%d", token, amount, height, txhash, txindex, outindex)
}

func DecodeAmountFromKey(key []byte, token string) (int64, error) {
	Prefix := fmt.Sprintf(PrivacyUTXOKEYPrefix+"-%s-", token)
	if len(key) <= len(Prefix) {
		return 0, types.ErrWrongKey
	}

	Amountstr := string(key[len(Prefix):])
	index := strings.Index(Amountstr, "-")
	Amountstr = string(key[len(Prefix):index])
	return strconv.ParseInt(Amountstr, 10, 64)
}

func CalcPrivacyUTXOkeyHeightPrefix(token string, amount int64) (key []byte) {
	return []byte(fmt.Sprintf(PrivacyUTXOKEYPrefix+"-%s-%d", token, amount))
}

//设置当前系统存在的token的amount的类型，如存在1,3,5,100...等等的类型,
func CalcprivacyKeyTokenAmountType(token string) (key []byte) {
	return []byte(fmt.Sprintf(PrivacyAmountTypePrefix+"-%s", token))
}

func CalcprivacyKeyTokenTypes() (key []byte) {
	return []byte(PrivacyTokenTypesPrefix)
}