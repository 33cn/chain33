package types

import (
	"encoding/json"
	"github.com/inconshreveable/log15"

	"errors"
)

//interface for testCase
type CaseFunc interface {
	GetID() string
	GetCmd() string
	GetDep() []string
	GetRepeat() int
	GetBaseCase() *BaseCase
	//一个用例的输入依赖于另一个用例输出，设置依赖的输出数据
	SetDependData(interface{})
	//执行用例命令，并返回用例打包结构
	SendCommand(packId string) (PackFunc, error)
}

//interface for check testCase result
type PackFunc interface {
	GetPackID() string
	SetPackID(id string)
	GetBaseCase() *BaseCase
	GetTxHash() string
	GetTxReceipt() string
	GetBasePack() *BaseCasePack
	SetLogger(fLog log15.Logger, tLog log15.Logger)
	GetCheckHandlerMap() CheckHandlerMap
	GetDependData() interface{}
	CheckResult(CheckHandlerMap) (bool, bool)
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
	TCase      CaseFunc
	CheckTimes int
	TxHash     string
	TxReceipt  string
	PackID     string
	FLog       log15.Logger
	TLog       log15.Logger
}


//default send command implementation
func DefaultSend(testCase CaseFunc, testPack PackFunc, packID string) (PackFunc, error) {


	baseCase := testCase.GetBaseCase()
	txHash, bSuccess := SendTxCommand(baseCase.Command)
	if !bSuccess {
		return nil, errors.New(txHash)
	}
	pack := testPack.GetBasePack()
	pack.TxHash = txHash
	pack.TCase = testCase

	pack.PackID = packID
	pack.CheckTimes = 0
	return testPack, nil
}


//interface CaseFunc implementing by BaseCase
func (t *BaseCase) SendCommand(packID string) (PackFunc, error) {
	return nil, nil
}

func (t *BaseCase) GetID() string {

	return t.ID
}

func (t *BaseCase) GetCmd() string {

	return t.Command
}

func (t *BaseCase) GetDep() []string {

	return t.Dep
}

func (t *BaseCase) GetRepeat() int {

	return t.Repeat
}

func (t *BaseCase) GetBaseCase() *BaseCase {

	return t
}

func (t *BaseCase) SetDependData(interface{}) {

}

//interface PackFunc implementing by BaseCasePack

func (pack *BaseCasePack) GetPackID() string {

	return pack.PackID
}

func (pack *BaseCasePack) SetPackID(id string) {

	pack.PackID = id
}

func (pack *BaseCasePack) GetBaseCase() *BaseCase {

	return pack.TCase.GetBaseCase()
}

func (pack *BaseCasePack) GetTxHash() string {

	return pack.TxHash
}


func (pack *BaseCasePack) GetTxReceipt() string {

	return pack.TxReceipt
}

func (pack *BaseCasePack) SetLogger(fLog log15.Logger, tLog log15.Logger) {

	pack.FLog = fLog
	pack.TLog = tLog
}

func (pack *BaseCasePack) GetBasePack() *BaseCasePack {

	return pack
}

func (pack *BaseCasePack) GetDependData() interface{} {

	return nil
}

func (pack *BaseCasePack) GetCheckHandlerMap() CheckHandlerMap {

	return make(map[string]CheckHandlerFunc, 1)
}

func (pack *BaseCasePack) CheckResult(handlerMap CheckHandlerMap) (bCheck bool, bSuccess bool) {

	bCheck = false
	bSuccess = false

	tCase := pack.TCase.GetBaseCase()
	txInfo, bReady := GetTxInfo(pack.TxHash)

	if !bReady && (txInfo != "tx not exist\n" || pack.CheckTimes >= CheckTimeout) {

		pack.TxReceipt = txInfo
		pack.FLog.Error("CheckTimeout", "TestID", pack.PackID, "ErrInfo", txInfo)
		pack.TxReceipt = txInfo
		return true, false
	}

	if bReady {

		bCheck = true
		var tyname string
		var jsonMap map[string]interface{}
		pack.TxReceipt = txInfo
		pack.FLog.Info("TxReceiptJson", "TestID", pack.PackID)
		//hack, for pretty json log
		pack.FLog.Info("PrettyJsonLogFormat", "TxReceipt", []byte(txInfo))
		err := json.Unmarshal([]byte(txInfo), &jsonMap)
		if err != nil {

			pack.FLog.Error("UnMarshalFailed", "TestID", pack.PackID, "jsonStr", txInfo, "ErrInfo", err.Error())
			return true, false
		}

		tyname, bSuccess = GetTxRecpTyname(jsonMap)
		pack.FLog.Info("CheckItemResult", "TestID", pack.PackID, "RecpTyname", tyname)

		if !bSuccess {

			logArr := jsonMap["receipt"].(map[string]interface{})["logs"].([]interface{})
			logErrInfo := ""
			for _, log := range logArr {

				logMap := log.(map[string]interface{})

				if logMap["tyName"].(string) == "LogErr" {

					logErrInfo = logMap["log"].(string)
					break
				}
			}
			pack.FLog.Error("ExecPack", "TestID", pack.PackID,
				"LogErrInfo", logErrInfo)

		} else {

			for _, item := range tCase.CheckItem {

				checkHandler, ok := handlerMap[item]
				if ok {

					itemRes := checkHandler(jsonMap)
					bSuccess = bSuccess && itemRes
					pack.FLog.Info("CheckItemResult", "TestID", pack.PackID, "Item", item, "Passed", itemRes)
				}
			}
		}
	}

	pack.CheckTimes++
	return bCheck, bSuccess
}
