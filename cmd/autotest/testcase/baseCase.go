package testcase

import (
	"encoding/json"

	"github.com/inconshreveable/log15"
)

//interface for testCase
type CaseFunc interface {
	getID() string
	getCmd() string
	getDep() []string
	getRepeat() int
	getBaseCase() *BaseCase
	setDependData(interface{})
	doSendCommand(id string) (PackFunc, error)
}

//interface for check testCase result
type PackFunc interface {
	getPackID() string
	getTxHash() string
	setLogger(fLog log15.Logger, tLog log15.Logger)
	getCheckHandlerMap() CheckHandlerMap
	getDependData() interface{}
	doCheckResult(CheckHandlerMap) (bool, bool)
}

//base test case
type BaseCase struct {
	ID        string   `toml:"id"`
	Command   string   `toml:"command"`
	Dep       []string `toml:"dep,omitempty"`
	CheckItem []string `toml:"checkItem,omitempty"`
	Repeat    int      `toml:"repeat,omitempty"`
}

//check item handler
type CheckHandlerFunc func(map[string]interface{}) bool
type CheckHandlerMap map[string]CheckHandlerFunc

//pack testCase with some check info
type BaseCasePack struct {
	tCase      CaseFunc
	checkTimes int
	txHash     string
	packID     string
	fLog       log15.Logger
	tLog       log15.Logger
}

//interface CaseFunc implementing by BaseCase

func (t *BaseCase) doSendCommand(id string) (PackFunc, error) {

	return nil, nil
}

func (t *BaseCase) getID() string {

	return t.ID
}

func (t *BaseCase) getCmd() string {

	return t.Command
}

func (t *BaseCase) getDep() []string {

	return t.Dep
}

func (t *BaseCase) getRepeat() int {

	return t.Repeat
}

func (t *BaseCase) getBaseCase() *BaseCase {

	return t
}

func (t *BaseCase) setDependData(interface{}) {

}

//interface PackFunc implementing by BaseCasePack

func (pack *BaseCasePack) getPackID() string {

	return pack.packID
}

func (pack *BaseCasePack) getTxHash() string {

	return pack.txHash
}

func (pack *BaseCasePack) setLogger(fLog log15.Logger, tLog log15.Logger) {

	pack.fLog = fLog
	pack.tLog = tLog
}

func (pack *BaseCasePack) getBasePack() *BaseCasePack {

	return pack
}

func (pack *BaseCasePack) getDependData() interface{} {

	return nil
}

func (pack *BaseCasePack) getCheckHandlerMap() CheckHandlerMap {

	return make(map[string]CheckHandlerFunc, 1)
}

func (pack *BaseCasePack) doCheckResult(handlerMap CheckHandlerMap) (bCheck bool, bSuccess bool) {

	bCheck = false
	bSuccess = false

	tCase := pack.tCase.getBaseCase()
	txInfo, bReady := getTxInfo(pack.txHash)

	if !bReady && (txInfo != "tx not exist\n" || pack.checkTimes >= CheckTimeout) {

		pack.fLog.Error("CheckTimeOut", "TestID", pack.packID, "ErrInfo", txInfo)
		return true, false
	}

	if bReady {

		bCheck = true
		var tyname string
		var jsonMap map[string]interface{}
		err := json.Unmarshal([]byte(txInfo), &jsonMap)

		if err != nil {

			pack.fLog.Error("UnMarshalFailed", "TestID", pack.packID, "ErrInfo", err.Error())
			return true, false
		}

		pack.fLog.Info("TxJsonInfo", "TestID", pack.packID)
		//tricky, for pretty json log
		pack.fLog.Info("PrettyJsonLogFormat", "TxJson", []byte(txInfo))
		tyname, bSuccess = getTxRecpTyname(jsonMap)
		pack.fLog.Info("CheckItemResult", "TestID", pack.packID, "tyname", tyname)

		if !bSuccess {

			logArr := jsonMap["receipt"].(map[string]interface{})["logs"].([]interface{})
			for _, log := range logArr {

				logMap := log.(map[string]interface{})

				if logMap["tyname"].(string) == "LogErr" {

					pack.fLog.Error("TxLogErr", "TestID", pack.packID,
						"ErrInfo", logMap["log"].(string))
					break
				}
			}
		} else {

			for _, item := range tCase.CheckItem {

				checkHandler, ok := handlerMap[item]
				if ok {

					itemRes := checkHandler(jsonMap)
					bSuccess = bSuccess && itemRes
					pack.fLog.Info("CheckItemResult", "TestID", pack.packID, "Item", item, "Passed", itemRes)
				}
			}
		}
	}

	pack.checkTimes++
	return bCheck, bSuccess
}
