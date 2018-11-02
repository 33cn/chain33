package types

import (
	"strings"
)

//simple case just executes without checking, suitable for init situation

type SimpleCase struct {
	BaseCase
}

type SimplePack struct {
	BaseCasePack
}

func (testCase *SimpleCase) SendCommand(packID string) (PackFunc, error) {

	return DefaultSend(testCase, &SimplePack{}, packID)
}

//simple case needn't check
func (pack *SimplePack) CheckResult(handlerMap CheckHandlerMap) (bCheck bool, bSuccess bool) {

	bCheck = true
	bSuccess = true
	if strings.Contains(pack.TxHash, "Err") || strings.Contains(pack.TxHash, "connection refused") {

		bSuccess = false
	}

	return bCheck, bSuccess
}
