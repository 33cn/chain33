package testcase

import "errors"

type TokenPreCreateCase struct {
	BaseCase
	//From string `toml:"from"`
	//Amount string `toml:"amount"`
}

type TokenPreCreatePack struct {
	BaseCasePack
}

type TokenFinishCreateCase struct {
	BaseCase
	//From string `toml:"from"`
	//Amount string `toml:"amount"`
}

type TokenFinishCreatePack struct {
	BaseCasePack
}

func (testCase *TokenPreCreateCase) doSendCommand(packID string) (PackFunc, error) {

	txHash, bSuccess := sendTxCommand(testCase.Command)
	if !bSuccess {
		return nil, errors.New(txHash)
	}
	pack := TokenPreCreatePack{}
	pack.txHash = txHash
	pack.tCase = testCase

	pack.packID = packID
	pack.checkTimes = 0
	return &pack, nil
}

func (testCase *TokenFinishCreateCase) doSendCommand(packID string) (PackFunc, error) {

	txHash, bSuccess := sendTxCommand(testCase.Command)
	if !bSuccess {
		return nil, errors.New(txHash)
	}
	pack := TokenFinishCreatePack{}
	pack.txHash = txHash
	pack.tCase = testCase

	pack.packID = packID
	pack.checkTimes = 0
	return &pack, nil
}

func (pack *TokenPreCreatePack) getCheckHandlerMap() CheckHandlerMap {

	funcMap := make(map[string]CheckHandlerFunc, 2)
	return funcMap
}

func (pack *TokenFinishCreatePack) getCheckHandlerMap() CheckHandlerMap {

	funcMap := make(map[string]CheckHandlerFunc, 2)
	return funcMap
}
