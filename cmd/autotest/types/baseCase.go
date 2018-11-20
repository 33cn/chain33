// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"encoding/json"
	"errors"

	"github.com/33cn/chain33/common/log/log15"
	chain33Type "github.com/33cn/chain33/system/dapp/commands/types"
)

//interface for testCase
type CaseFunc interface {
	//获取用例id
	GetID() string
	//获取命令行
	GetCmd() string
	//获取依赖数组
	GetDep() []string
	//获取重复次数
	GetRepeat() int
	//直接获取基类类型指针，方便获取所有成员
	GetBaseCase() *BaseCase
	//一个用例的输入依赖于另一个用例输出，设置依赖的输出数据
	SetDependData(interface{})
	//执行用例命令，并返回用例打包结构
	SendCommand(packId string) (PackFunc, error)
}

//interface for check testCase result
type PackFunc interface {
	//获取id
	GetPackID() string
	//设置id
	SetPackID(id string)
	//获取用例的基类指针
	GetBaseCase() *BaseCase
	//获取交易哈希
	GetTxHash() string
	//获取交易回执，json字符串
	GetTxReceipt() string
	//获取基类打包类型指针，提供直接访问成员
	GetBasePack() *BaseCasePack
	//设置log15日志
	SetLogger(fLog log15.Logger, tLog log15.Logger)
	//获取check函数字典
	GetCheckHandlerMap() interface{}
	//获取依赖数据
	GetDependData() interface{}
	//执行结果check
	CheckResult(interface{}) (bool, bool)
}

//base test case
type BaseCase struct {
	ID        string   `toml:"id"`
	Command   string   `toml:"command"`
	Dep       []string `toml:"dep,omitempty"`
	CheckItem []string `toml:"checkItem,omitempty"`
	Repeat    int      `toml:"repeat,omitempty"`
	Fail      bool     `toml:"fail,omitempty"`
}

//check item handler
//适配autotest早期版本，handlerfunc的参数为json的map形式，后续统一使用chain33的TxDetailResult结构体结构
type CheckHandlerFuncDiscard func(map[string]interface{}) bool
type CheckHandlerMapDiscard map[string]CheckHandlerFuncDiscard

//建议使用
type CheckHandlerParamType *chain33Type.TxDetailResult
type CheckHandlerFunc func(CheckHandlerParamType) bool
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

//default send command implementation, only for transaction type case
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

func (pack *BaseCasePack) GetCheckHandlerMap() interface{} {

	//return make(map[string]CheckHandlerFunc, 1)
	return nil
}

func (pack *BaseCasePack) CheckResult(handlerMap interface{}) (bCheck bool, bSuccess bool) {

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
		var txRecp chain33Type.TxDetailResult
		pack.TxReceipt = txInfo
		pack.FLog.Info("TxReceiptJson", "TestID", pack.PackID)
		//hack, for pretty json log
		pack.FLog.Info("PrettyJsonLogFormat", "TxReceipt", []byte(txInfo))
		//适配前期的接口
		err := json.Unmarshal([]byte(txInfo), &jsonMap)
		err1 := json.Unmarshal([]byte(txInfo), &txRecp)

		if err != nil || err1 != nil {

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

			//为了兼容前期autotest的map形式交易回执
			if funcMap, ok := handlerMap.(CheckHandlerMapDiscard); ok {

				for _, item := range tCase.CheckItem {

					checkHandler, exist := funcMap[item]
					if exist {

						itemRes := checkHandler(jsonMap)
						bSuccess = bSuccess && itemRes
						pack.FLog.Info("CheckItemResult", "TestID", pack.PackID, "Item", item, "Passed", itemRes)
					}
				}

			} else if funcMap, ok := handlerMap.(CheckHandlerMap); ok { //采用结构体形式回执

				for _, item := range tCase.CheckItem {

					checkHandler, exist := funcMap[item]
					if exist {

						itemRes := checkHandler(&txRecp)
						bSuccess = bSuccess && itemRes
						pack.FLog.Info("CheckItemResult", "TestID", pack.PackID, "Item", item, "Passed", itemRes)
					}
				}
			}
		}
	}

	pack.CheckTimes++
	return bCheck, bSuccess
}
