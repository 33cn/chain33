package privacy

import (
	"fmt"
)

const (
	PrivacyOutputKeyPrefix = "mavl-privacy-output-ahi"
	PrivacyInputKeyPrefix = "mavl-privacy-input"
)

func calcprivacyOutputKey(amount int64, txhash string, index int) (key []byte) {
	return []byte(fmt.Sprintf(PrivacyOutputKeyPrefix + "-%d-%s-%d", amount, txhash, index))
}
