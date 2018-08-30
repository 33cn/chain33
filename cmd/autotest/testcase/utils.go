package testcase

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const FloatDiff = 0.00001

var (
	CliCmd         string
	CheckSleepTime time.Duration
	CheckTimeout   int
)

func Init(cliCmd string, checkSleep int, checkTimeout int) {

	CliCmd = cliCmd
	CheckSleepTime = time.Duration(checkSleep)
	CheckTimeout = checkTimeout
}

//customize log15 log format
func AutoTestLogFormat() log15.Format {


	logfmt := log15.LogfmtFormat()

	return log15.FormatFunc(func(r *log15.Record) []byte {

		if r.Msg == "PrettyJsonLogFormat" && len(r.Ctx) == 4{

			b, ok := r.Ctx[3].([]byte)
			if ok {
				//return raw json data directly
				return b
			}
		}

		return logfmt.Format(r)
	})

}

//invoke chain33 client
func runChain33Cli(para []string) (string, error) {

	rawOut, err := exec.Command(CliCmd, para[0:]...).CombinedOutput()

	strOut := string(rawOut)

	return strOut, err
}

//according to the accuracy of coins balance
func isBalanceEqualFloat(f1 float64, f2 float64) bool {

	if (f2-f1 < FloatDiff) && (f1-f2 < FloatDiff) {

		return true
	} else {

		return false
	}
}

//excute
func sendTxCommand(cmd string) (string, bool) {

	output, err := runChain33Cli(strings.Fields(cmd))
	bSuccess := true
	if err != nil {
		bSuccess = false
		output = err.Error()
	} else {
		output = output[0 : len(output)-1]
	}

	return output, bSuccess
}

//get tx query -s txHash
func getTxRecpTyname(txInfo map[string]interface{}) (tyname string, bSuccess bool) {

	tyname = txInfo["receipt"].(map[string]interface{})["tyname"].(string)

	bSuccess = false

	if tyname == "ExecOk" {
		bSuccess = true
	}

	return tyname, bSuccess
}

//get tx receipt with tx hash code if exist
func getTxInfo(txHash string) (string, bool) {

	bReady := false
	txInfo, err := runChain33Cli(strings.Fields(fmt.Sprintf("tx query -s %s", txHash)))

	if err == nil && txInfo != "tx not exist\n" {

		bReady = true
	} else if err != nil {

		txInfo = err.Error()
	}

	return txInfo, bReady
}

//diff balance
func checkBalanceDeltaWithAddr(log map[string]interface{}, addr string, delta float64) bool {

	logAddr := log["current"].(map[string]interface{})["addr"].(string)
	prev, _ := strconv.ParseFloat(log["prev"].(map[string]interface{})["balance"].(string), 64)
	curr, _ := strconv.ParseFloat(log["current"].(map[string]interface{})["balance"].(string), 64)

	logDelta := curr - prev

	return (logAddr == addr) && (isBalanceEqualFloat(logDelta, delta))
}

func checkFrozenDeltaWithAddr(log map[string]interface{}, addr string, delta float64) bool {

	logAddr := log["current"].(map[string]interface{})["addr"].(string)
	prev, _ := strconv.ParseFloat(log["prev"].(map[string]interface{})["frozen"].(string), 64)
	curr, _ := strconv.ParseFloat(log["current"].(map[string]interface{})["frozen"].(string), 64)

	logDelta := curr - prev

	return (logAddr == addr) && (isBalanceEqualFloat(logDelta, delta))
}
