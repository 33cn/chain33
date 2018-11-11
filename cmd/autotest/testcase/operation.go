package testcase

import (
	"container/list"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
)

type TestOperator struct {
	addDone     chan bool
	sendDone    chan bool
	checkDone   chan bool
	depEmpty    chan bool
	sendBuf     chan CaseFunc
	checkBuf    chan PackFunc
	addDepBuf   chan CaseFunc
	delDepBuf   chan PackFunc
	depCaseMap  map[string][]CaseFunc //key:TestID, val: array of testCases depending on the key
	depCountMap map[string]int        //key:TestID, val: dependency count

	fLog log15.Logger
	tLog log15.Logger

	totalCase int
	totalFail int
	failID    []string
}

func (tester *TestOperator) AddCaseArray(caseArrayList ...interface{}) {

	for i := range caseArrayList {

		caseArray := reflect.ValueOf(caseArrayList[i])

		if caseArray.Kind() != reflect.Slice {
			continue
		}

		for j := 0; j < caseArray.Len(); j++ {

			testCase := caseArray.Index(j).Addr().Interface().(CaseFunc)
			baseCase := testCase.getBaseCase()

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

func (tester *TestOperator) HandleDependency() {

	keepLoop := true
	addDoneFlag := false
	for keepLoop {
		select {

		case testCase := <-tester.addDepBuf:

			baseCase := testCase.getBaseCase()

			for _, depID := range baseCase.Dep {

				testArr := append(tester.depCaseMap[depID], testCase)
				tester.depCaseMap[depID] = testArr
			}

		case testPack := <-tester.delDepBuf:

			packID := testPack.getPackID()
			//取出依赖于该用例的所有用例
			caseArr, exist := tester.depCaseMap[packID]
			if exist {

				for i := range caseArr {

					baseCase := caseArr[i].getBaseCase()
					tester.depCountMap[baseCase.ID]--
					if tester.depCountMap[baseCase.ID] == 0 {

						caseArr[i].setDependData(testPack.getDependData())
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
						tester.tLog.Error("InvalidDependency", "TestIDs", tester.depCaseMap[depID][i].getID())
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
					testCase := e.Value.(CaseFunc)
					c.Remove(e)
					baseCase := testCase.getBaseCase()

					repeat := baseCase.Repeat
					if repeat <= 0 { //default val if empty in tomlFile
						repeat = 1
					}

					tester.totalCase += repeat
					packID := baseCase.ID

					for i := 1; i <= repeat; i++ {

						tester.fLog.Info("CommandExec", "TestID", packID, "Command", baseCase.Command)
						pack, err := testCase.doSendCommand(packID)

						if err != nil {

							if strings.Contains(packID, "fail") { //some logs

								tester.tLog.Info("TestCaseResult", "TestID", packID, "Result", "Succeed")

							} else {

								tester.totalFail++
								tester.failID = append(tester.failID, packID)
								tester.tLog.Error("TestCaseFailDetail", "TestID", packID, "Command", baseCase.Command, "Result", "")
								fmt.Println(err.Error())
							}
							tester.fLog.Info("CommandResult", "TestID", packID, "Result", err.Error())
							tester.delDepBuf <- &BaseCasePack{packID: packID}
							continue
						}

						pack.setLogger(tester.fLog, tester.tLog)
						tester.checkBuf <- pack
						tester.fLog.Info("CommandResult", "TestID", packID, "Result", pack.getTxHash())
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

		case <-time.After(time.Second): //do check operation with an dependent check list

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

						casePack := e.Value.(PackFunc)
						checkOver, bSuccess := casePack.doCheckResult(casePack.getCheckHandlerMap())
						n = e.Next()

						//have done checking
						if checkOver {

							c.Remove(e)
							//find if any case depend
							tester.delDepBuf <- casePack
							isFailCase := strings.Contains(casePack.getPackID(), "fail")

							if (bSuccess && !isFailCase) || (!bSuccess && isFailCase) { //some logs

								tester.tLog.Info("TestCaseResult", "TestID", casePack.getPackID(), "Result", "Succeed")

							} else {
								basePack := casePack.getBasePack()
								baseCase := basePack.tCase.getBaseCase()
								tester.totalFail++
								tester.failID = append(tester.failID, casePack.getPackID())
								tester.tLog.Error("TestCaseFailDetail", "TestID", casePack.getPackID(), "Command", baseCase.Command, "TxHash", basePack.txHash, "TxReceipt", "")
								fmt.Println(basePack.txReceipt)
							}
						}
					}

					if c.Len() > 0 {

						//tester.tLog.Info("CheckRoutineSleep", "SleepTime", CheckSleepTime*time.Second, "WaitCheckNum", c.Len())
						time.Sleep(CheckSleepTime * time.Second)
					}

				}

			}(checkList, checkWg)

			checkList = nil //always set nil for new list
		}
	}

	checkWg.Wait()
	tester.checkDone <- true
}

func (tester *TestOperator) WaitTest() {

	<-tester.checkDone
	if tester.totalFail > 0 {

		tester.tLog.Error("TestDone", "TotalCase", tester.totalCase, "TotalFail", tester.totalFail, "FailID", tester.failID)
		tester.fLog.Error("TestDone", "TotalCase", tester.totalCase, "TotalFail", tester.totalFail, "FailID", tester.failID)
	} else {

		tester.tLog.Info("TestDone", "TotalCase", tester.totalCase, "TotalFail", tester.totalFail)
		tester.fLog.Info("TestDone", "TotalCase", tester.totalCase, "TotalFail", tester.totalFail)
	}
}

func NewTestOperator(fileLog log15.Logger, stdLog log15.Logger) (tester *TestOperator) {

	tester = new(TestOperator)

	tester.addDone = make(chan bool, 1)
	tester.sendDone = make(chan bool, 1)
	tester.checkDone = make(chan bool, 1)
	tester.depEmpty = make(chan bool, 1)
	tester.addDepBuf = make(chan CaseFunc, 1)
	tester.delDepBuf = make(chan PackFunc, 1)
	tester.sendBuf = make(chan CaseFunc, 1)
	tester.checkBuf = make(chan PackFunc, 1)
	tester.depCaseMap = make(map[string][]CaseFunc)
	tester.depCountMap = make(map[string]int)
	tester.fLog = fileLog
	tester.tLog = stdLog
	tester.totalCase = 0
	tester.totalFail = 0
	return tester

}
