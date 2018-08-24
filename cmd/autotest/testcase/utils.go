package testcase

import (
	"os/exec"
	"strings"
	"fmt"
	"strconv"
	"encoding/json"
	"time"
)

const FloatDiff = 0.00001



var(

	CliCmd string
	CheckSleepTime time.Duration
	CheckTimeout int
)


func Init(cliCmd string, checkSleep int, checkTimeout int){

	CliCmd = cliCmd
	CheckSleepTime = time.Duration(checkSleep)
	CheckTimeout = checkTimeout
}


//invoke chain33 client
func runChain33Cli(para []string) (string, error){


	rawOut, err := exec.Command(CliCmd, para[0:len(para)]...).CombinedOutput()

	strOut := string(rawOut)

	return strOut, err
}




//according to the accuracy of coins balance
func isBalanceEqualFloat(f1 float64, f2 float64) bool {


	if (f2 - f1 < FloatDiff) && (f1 - f2 < FloatDiff) {

		return true
	}else{

		return false
	}
}


//excute
func sendTxCommand(cmd string) (string, bool){


	output, err := runChain33Cli(strings.Fields(cmd))
	bSuccess := true
	if err != nil {
		bSuccess = false
		output = err.Error()
	}else{
		output = output[0:len(output) - 1]
	}

	return output, bSuccess
}


//get tx query -s txHash
func getTxRecpTyname(txInfo  map[string]interface{}) (tyname string, bSuccess bool){


	tyname = txInfo["receipt"].(map[string]interface{})["tyname"].(string)

	bSuccess = false

	if tyname == "ExecOk"{
		bSuccess = true
	}

	return tyname, bSuccess
}


//get trade sell sellID
func getSellID(txInfo  map[string]interface{}) (tyname string){


	tyname = ""

	return tyname
}


//get tx receipt with tx hash code if exist
func getTxInfo(txHash string) (string, bool){

	bReady := false
	txInfo, err:= runChain33Cli(strings.Fields(fmt.Sprintf("tx query -s %s", txHash)))

	if err == nil && txInfo != "tx not exist\n" {

		bReady = true
	}else if err != nil {

		txInfo = err.Error()
	}

	return txInfo, bReady
}


//diff balance
func checkBalanceDeltaWithAddr(log map[string]interface{}, addr string, delta float64) bool{

	logAddr := log["current"].(map[string]interface{})["addr"].(string)
	prev, _ := strconv.ParseFloat(log["prev"].(map[string]interface{})["balance"].(string), 64)
	curr, _ := strconv.ParseFloat(log["current"].(map[string]interface{})["balance"].(string), 64)

	logDelta := curr - prev

	return (logAddr == addr) && (isBalanceEqualFloat(logDelta, delta))
}


func checkFrozenDeltaWithAddr(log map[string]interface{}, addr string, delta float64) bool{

	logAddr := log["current"].(map[string]interface{})["addr"].(string)
	prev, _ := strconv.ParseFloat(log["prev"].(map[string]interface{})["frozen"].(string), 64)
	curr, _ := strconv.ParseFloat(log["current"].(map[string]interface{})["frozen"].(string), 64)

	logDelta := curr - prev

	return (logAddr == addr) && (isBalanceEqualFloat(logDelta, delta))
}




func getAddrBalance(addr string) (string, bool){


	paraStr := make([]string, 6)[:0]

	output, err:= runChain33Cli(append(paraStr, "account", "balance", "-e", "coins", "-a", addr))

	if err != nil {

		return "Fail to get balance", false
	}

	var jsonMap map[string]interface{}

	if err := json.Unmarshal([]byte(output), &jsonMap); err != nil {

		return "Fail to get balance", false
	}

	balance := jsonMap["balance"].(string)

	//bal, _ := strconv.ParseFloat(balance, 32)

	return balance, true
}



func outputRes(testID string, status bool, desc string) {

	var stastr string

	if status {
		stastr = "Succeed!"
	}else{
		stastr = "Failed: "
	}

	desc = strings.Replace(desc, "\n", " ", -1)

	fmt.Printf("[TestBtySendResult] [id:%s]\t-----------\t[%s%s]\n", testID, stastr, desc)
}


/*
func errReport(testID string, info...string){

	totalErr++
	failTestID = append(failTestID, testID)
}


func OutputTestResult() {

	tLog.Info("TestResultSummary", "TotalCasesNum", totalCase, "TotalFailedNum", totalErr, "\nFailedTestID", failTestID)
	//fLog.Info("TestResultSummary", "TotalCasesNum", totalCase, "TotalFailedNum", totalErr, "\nFailedTestID", failTestID)
}*/