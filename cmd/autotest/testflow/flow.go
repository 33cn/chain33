// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package testflow test flow, Add=>HandleDepend=>Send=>Check
package testflow

import (
	"container/list"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/33cn/chain33/cmd/autotest/types"
	"github.com/33cn/chain33/common/log/log15"
)

//TestOperator 测试操作符
type TestOperator struct {
	addDone     chan bool
	sendDone    chan bool
	checkDone   chan bool
	depEmpty    chan bool
	sendBuf     chan types.CaseFunc
	checkBuf    chan types.PackFunc
	addDepBuf   chan types.CaseFunc
	delDepBuf   chan types.PackFunc
	depCaseMap  map[string][]types.CaseFunc //key:TestID, val: array of testCases depending on the key
	depCountMap map[string]int              //key:TestID, val: dependency count

	fLog log15.Logger
	tLog log15.Logger

	dapp      string
	totalCase int
	totalFail int
	failID    []string
}

//AddCaseArray 添加用例
func (tester *TestOperator) AddCaseArray(caseArrayList ...interface{}) {

	for i := range caseArrayList {

		caseArray := reflect.ValueOf(caseArrayList[i])

		if caseArray.Kind() != reflect.Slice {
			continue
		}

		for j := 0; j < caseArray.Len(); j++ {

			testCase := caseArray.Index(j).Addr().Interface().(types.CaseFunc)
			baseCase := testCase.GetBaseCase()

			if len(baseCase.Dep) > 0 {

				tester.addDepBuf <- testCase
			} else {

				tester.sendBuf <- testCase
			}

			tester.depCountMap[baseCase.ID] = len(baseCase.Dep)
		}
	}

	tester.addDone <- true

}

//HandleDependency 管理依赖
func (tester *TestOperator) HandleDependency() {

	keepLoop := true
	addDoneFlag := false
	for keepLoop {
		select {

		case testCase := <-tester.addDepBuf:

			baseCase := testCase.GetBaseCase()

			for _, depID := range baseCase.Dep {

				testArr := append(tester.depCaseMap[depID], testCase)
				tester.depCaseMap[depID] = testArr
			}

		case testPack := <-tester.delDepBuf:

			packID := testPack.GetPackID()
			//取出依赖于该用例的所有用例
			caseArr, exist := tester.depCaseMap[packID]
			if exist {

				for i := range caseArr {

					caseArr[i].SetDependData(testPack.GetDependData())
					baseCase := caseArr[i].GetBaseCase()
					tester.depCountMap[baseCase.ID]--
					if tester.depCountMap[baseCase.ID] == 0 {
						tester.sendBuf <- caseArr[i]
						delete(tester.depCountMap, baseCase.ID)
					}
				}
				delete(tester.depCaseMap, packID)
			}
		case <-tester.addDone:

			addDoneFlag = true
			//check dependency validity

			for depID := range tester.depCaseMap {

				_, exist := tester.depCountMap[depID]
				if !exist {
					//depending testCase not exist
					for i := range tester.depCaseMap[depID] {
						tester.tLog.Error("CheckCaseDependencyValid", "TestID", tester.depCaseMap[depID][i].GetID(), "DependTestID", depID, "Error", "DependCaseNotExist")
					}
					delete(tester.depCaseMap, depID)
				}
			}

		case <-time.After(time.Second):

			if addDoneFlag && len(tester.depCaseMap) == 0 {

				tester.depEmpty <- true
				keepLoop = false
				break
			}
		}
	}

	//each case will send to delDepBuf after checking
	for {
		<-tester.delDepBuf
	}
}

//RunSendFlow send
func (tester *TestOperator) RunSendFlow() {

	depEmpty := false
	keepLoop := true
	sendWg := &sync.WaitGroup{}
	sendList := (*list.List)(nil)

	for keepLoop {
		select {

		case testCase := <-tester.sendBuf:

			if sendList == nil {
				sendList = list.New()
			}
			sendList.PushBack(testCase)

		case <-tester.depEmpty:
			depEmpty = true

		case <-time.After(time.Second):

			if depEmpty {
				keepLoop = false
			}

			if sendList == nil {
				break
			}

			sendWg.Add(1)
			go func(c *list.List, wg *sync.WaitGroup) {

				defer wg.Done()
				var n *list.Element
				for e := c.Front(); e != nil; e = n {

					n = e.Next()
					testCase := e.Value.(types.CaseFunc)
					c.Remove(e)
					baseCase := testCase.GetBaseCase()

					repeat := baseCase.Repeat
					if repeat <= 0 { //default val if empty in tomlFile
						repeat = 1
					}

					tester.totalCase += repeat
					packID := baseCase.ID

					for i := 1; i <= repeat; i++ {

						tester.fLog.Info("CommandExec", "TestID", packID, "Command", baseCase.Command)
						pack, err := testCase.SendCommand(packID)

						if err != nil {

							if baseCase.Fail { //fail type case

								tester.tLog.Info("TestCaseResult", "TestID", packID, "Result", "Succeed")

							} else {

								tester.totalFail++
								tester.failID = append(tester.failID, packID)
								tester.tLog.Error("TestCaseResult", "TestID", packID, "Command", baseCase.Command, "Result", "")
								fmt.Println(err.Error())
							}
							tester.fLog.Info("CommandResult", "TestID", packID, "Result", err.Error())
							casePack := &types.BaseCasePack{}
							casePack.SetPackID(packID)
							tester.delDepBuf <- casePack

						} else {

							pack.SetLogger(tester.fLog, tester.tLog)
							tester.checkBuf <- pack
							tester.fLog.Info("CommandResult", "TestID", packID, "Result", pack.GetTxHash())
						}

						//distinguish with different packID, format: [TestID_RepeatOrder]
						packID = fmt.Sprintf("%s_%d", baseCase.ID, i)
					}
				}
			}(sendList, sendWg)

			sendList = nil
		}
	}

	sendWg.Wait()
	tester.sendDone <- true
}

//RunCheckFlow check
func (tester *TestOperator) RunCheckFlow() {

	checkList := (*list.List)(nil)
	sendDoneFlag := false
	keepLoop := true
	checkWg := &sync.WaitGroup{}

	for keepLoop {

		select {

		case casePack := <-tester.checkBuf:

			if checkList == nil {
				checkList = list.New()
			}

			checkList.PushBack(casePack)

		case <-tester.sendDone:
			sendDoneFlag = true

		case <-time.After(time.Second): //do check operation with an independent check list

			if checkList == nil {

				if sendDoneFlag {
					keepLoop = false //no more case from send flow
				}
				break
			}

			checkWg.Add(1)
			go func(c *list.List, wg *sync.WaitGroup) {

				defer wg.Done()
				for c.Len() > 0 {

					var n *list.Element
					//traversing checkList and check the result

					for e := c.Front(); e != nil; e = n {

						casePack := e.Value.(types.PackFunc)
						checkOver, bSuccess := casePack.CheckResult(casePack.GetCheckHandlerMap())
						n = e.Next()

						//have done checking
						if checkOver {

							c.Remove(e)
							//find if any case depend
							tester.delDepBuf <- casePack
							isFailCase := casePack.GetBaseCase().Fail

							if (bSuccess && !isFailCase) || (!bSuccess && isFailCase) { //some logs

								tester.tLog.Info("TestCaseResult", "TestID", casePack.GetPackID(), "Result", "Succeed")

							} else {
								baseCase := casePack.GetBaseCase()
								tester.totalFail++
								tester.failID = append(tester.failID, casePack.GetPackID())
								tester.tLog.Error("TestCaseFailDetail", "TestID", casePack.GetPackID(), "Command", baseCase.Command, "TxHash", casePack.GetTxHash(), "TxReceipt", "")
								fmt.Println(casePack.GetTxReceipt())
							}
						}
					}

					if c.Len() > 0 {

						//tester.tLog.Info("CheckRoutineSleep", "SleepTime", CheckSleepTime*time.Second, "WaitCheckNum", c.Len())
						time.Sleep(time.Duration(checkSleepTime) * time.Second)
					}

				}

			}(checkList, checkWg)

			checkList = nil //always set nil for new list
		}
	}

	checkWg.Wait()
	tester.checkDone <- true
}

//WaitTest 等待测试
func (tester *TestOperator) WaitTest() *AutoTestResult {

	<-tester.checkDone
	return &AutoTestResult{
		dapp:       tester.dapp,
		totalCase:  tester.totalCase,
		failCase:   tester.totalFail,
		failCaseID: tester.failID,
	}
}

//NewTestOperator new
func NewTestOperator(stdLog log15.Logger, fileLog log15.Logger, dapp string) (tester *TestOperator) {

	tester = new(TestOperator)

	tester.addDone = make(chan bool, 1)
	tester.sendDone = make(chan bool, 1)
	tester.checkDone = make(chan bool, 1)
	tester.depEmpty = make(chan bool, 1)
	tester.addDepBuf = make(chan types.CaseFunc, 1)
	tester.delDepBuf = make(chan types.PackFunc, 1)
	tester.sendBuf = make(chan types.CaseFunc, 1)
	tester.checkBuf = make(chan types.PackFunc, 1)
	tester.depCaseMap = make(map[string][]types.CaseFunc)
	tester.depCountMap = make(map[string]int)
	tester.fLog = fileLog.New("module", dapp)
	tester.tLog = stdLog.New("module", dapp)
	tester.dapp = dapp
	tester.totalCase = 0
	tester.totalFail = 0
	return tester

}
