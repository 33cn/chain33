package testcase

import (
	"container/list"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/inconshreveable/log15"
)

type TestOperator struct {
	addDone    chan bool
	sendDone   chan bool
	checkDone  chan bool
	depEmpty   chan bool
	sendBuf    chan CaseFunc
	checkBuf   chan PackFunc
	addDepBuf  chan CaseFunc
	delDepBuf  chan PackFunc
	depCaseMap map[string][]CaseFunc

	fLog log15.Logger
	tLog log15.Logger

	totalCase int
	totalFail int
	failID    []string
}

func (tester *TestOperator) HandleDependency() {

	keepLoop := true
	addDoneFlag := false
	for keepLoop {
		select {

		case testCase := <-tester.addDepBuf:

			depArr := append(tester.depCaseMap[testCase.getDep()], testCase)
			tester.depCaseMap[testCase.getDep()] = depArr

		case testPack := <-tester.delDepBuf:

			id := testPack.getPackID()
			testArr, isExist := tester.depCaseMap[id]
			if isExist {

				for _, testCase := range testArr {

					testCase.setDependData(testPack.getDependData())
					tester.sendBuf <- testCase
				}
				delete(tester.depCaseMap, id)
			}
		case <-tester.addDone:
			addDoneFlag = true

		case <-time.After(2 * time.Second):

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

	for keepLoop {
		select {

		case testCase := <-tester.sendBuf:

			sendWg.Add(1)
			go func(testCase CaseFunc, wg *sync.WaitGroup) {

				defer wg.Done()
				repeat := testCase.getRepeat()
				if repeat <= 0 { //default val if empty in tomlFile
					repeat = 1
				}

				tester.totalCase += repeat
				testID := testCase.getID()
				packID := testID

				for i := 1; i <= repeat; i++ {

					tester.fLog.Info("CommandExec", "TestID", packID, "CMD", testCase.getCmd())
					pack, err := testCase.doSendCommand(packID)

					if err != nil {
						tester.tLog.Error("TestCaseResult", "TestID", packID, "Failed", "ErrInfo", err.Error())
						tester.fLog.Info("CommandResult", "TestID", packID, "Result", err.Error())
						tester.delDepBuf <- &BaseCasePack{packID: packID}
						continue
					}

					pack.setLogger(tester.fLog, tester.tLog)
					tester.checkBuf <- pack
					tester.fLog.Info("CommandResult", "TestID", packID, "Result", pack.getTxHash())
					//distinguish with different packID, format: [TestID_RepeatOrder]
					packID = fmt.Sprintf("%s_%d", testID, i)
				}

			}(testCase, sendWg)

		case <-tester.depEmpty:
			depEmpty = true

		case <-time.After(2 * time.Second):

			if depEmpty {

				keepLoop = false
				break
			}
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

		case <-time.After(2 * time.Second): //do check operation with an dependent check list

			if checkList == nil {

				if sendDoneFlag {
					keepLoop = false //no more case from send flow
				}
				break
			}

			checkWg.Add(1)
			go func(c *list.List, wg *sync.WaitGroup, depBuf chan PackFunc) {

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
							depBuf <- casePack
							isFailCase := strings.Contains(casePack.getPackID(), "fail")

							if (bSuccess && !isFailCase) || (!bSuccess && isFailCase) { //some logs

								tester.tLog.Info("TestCaseResult", "TestID", casePack.getPackID(), "Result", "Succeed")

							} else {

								tester.totalFail++
								tester.failID = append(tester.failID, casePack.getPackID())
								tester.tLog.Error("TestCaseResult", "TestID", casePack.getPackID(), "Result", "Failed")
							}
						}
					}

					if c.Len() > 0 {

						tester.tLog.Info("CheckRoutineSleep", "SleepTime", CheckSleepTime*time.Second, "WaitCheckNum", c.Len())
						time.Sleep(CheckSleepTime * time.Second)
					}

				}

			}(checkList, checkWg, tester.delDepBuf)

			checkList = nil //always set nil for new list
		}
	}

	checkWg.Wait()
	tester.checkDone <- true
}

func (tester *TestOperator) AddCase(testCase CaseFunc) {

	if testCase.getDep() != "" {
		//save to depend map
		tester.addDepBuf <- testCase

	} else {

		tester.sendBuf <- testCase
	}

}

func (tester *TestOperator) WaitTest() {

	tester.addDone <- true
	<-tester.checkDone
	tester.tLog.Info("TestDone", "TotalCase", tester.totalCase, "TotalFail", tester.totalFail, "FailID", tester.failID)
	tester.fLog.Info("TestDone", "TotalCase", tester.totalCase, "TotalFail", tester.totalFail, "FailID", tester.failID)
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
	tester.fLog = fileLog
	tester.tLog = stdLog
	tester.totalCase = 0
	tester.totalFail = 0
	return tester

}
