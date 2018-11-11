// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testcase

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/33cn/chain33/common/log/log15"
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

		if r.Msg == "PrettyJsonLogFormat" && len(r.Ctx) == 4 {

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

func checkTxHashValid(txHash string) bool {

	return len(txHash) == 66 && strings.HasPrefix(txHash, "0x")
}

//excute
func sendTxCommand(cmd string) (string, bool) {

	output, err := runChain33Cli(strings.Fields(cmd))
	if err != nil {
		return err.Error(), false
	} else {
		output = output[0 : len(output)-1]
	}

	return output, checkTxHashValid(output)
}

//隐私交易执行回执哈希为json格式，需要解析
func sendPrivacyTxCommand(cmd string) (string, bool) {

	//construct tx by send command
	if strings.HasPrefix(cmd, "send") {
		return sendTxCommand(cmd)
	}

	output, err := runChain33Cli(strings.Fields(cmd))

	if err != nil {
		return err.Error(), false
	}

	if strings.Contains(output, "Err") {
		return output, false
	}

	var jsonMap map[string]interface{}
	err = json.Unmarshal([]byte(output), &jsonMap)
	if err != nil {
		return output, false
	}
	output = jsonMap["hash"].(string)

	return output, checkTxHashValid(output)
}

//get tx query -s txHash
func getTxRecpTyname(txInfo map[string]interface{}) (tyname string, bSuccess bool) {

	tyname = txInfo["receipt"].(map[string]interface{})["tyName"].(string)

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

	logDelta := (curr - prev) / 1e8

	return (logAddr == addr) && (isBalanceEqualFloat(logDelta, delta))
}

func checkFrozenDeltaWithAddr(log map[string]interface{}, addr string, delta float64) bool {

	logAddr := log["current"].(map[string]interface{})["addr"].(string)
	prev, _ := strconv.ParseFloat(log["prev"].(map[string]interface{})["frozen"].(string), 64)
	curr, _ := strconv.ParseFloat(log["current"].(map[string]interface{})["frozen"].(string), 64)

	logDelta := (curr - prev) / 1e8

	return (logAddr == addr) && (isBalanceEqualFloat(logDelta, delta))
}

//calculate total amount in tx in/out utxo set, key = ["keyinput" | "keyoutput"]
func calcTxUtxoAmount(log map[string]interface{}, key string) float64 {

	if log[key] == nil {
		return 0
	}

	utxoArr := log[key].([]interface{})
	var totalAmount float64 = 0.0

	for i := range utxoArr {

		temp, _ := strconv.ParseFloat(utxoArr[i].(map[string]interface{})["amount"].(string), 64)
		totalAmount += temp
	}

	return totalAmount / 1e8
}

//calculate available utxo with specific addr and txHash
func calcUtxoAvailAmount(addr string, txHash string) (float64, error) {

	outStr, err := runChain33Cli(strings.Fields(fmt.Sprintf("privacy showpai -d 1 -a %s", addr)))

	if err != nil {
		return 0, err
	}
	var jsonMap map[string]interface{}
	err = json.Unmarshal([]byte(outStr), &jsonMap)

	if err != nil {
		return 0, err
	}

	var totalAmount float64 = 0
	if jsonMap["AvailableDetail"] == nil {
		return 0, nil
	}

	availArr := jsonMap["AvailableDetail"].([]interface{})

	for i := range availArr {

		if availArr[i].(map[string]interface{})["Txhash"].(string) == txHash {

			temp, _ := strconv.ParseFloat(availArr[i].(map[string]interface{})["Amount"].(string), 64)
			totalAmount += temp
		}
	}

	return totalAmount, nil
}

//calculate spend utxo with specific addr and txHash
func calcUtxoSpendAmount(addr string, txHash string) (float64, error) {

	outStr, err := runChain33Cli(strings.Fields(fmt.Sprintf("privacy showpas -a %s", addr)))

	if strings.Contains(outStr, "Err") {
		return 0, errors.New(outStr)
	}

	idx := strings.Index(outStr, "\n") + 1

	if err != nil {
		return 0, err
	}
	var jsonArr []interface{}
	err = json.Unmarshal([]byte(outStr[idx:]), &jsonArr)

	if err != nil {
		return 0, err
	}

	var totalAmount float64 = 0

	for i := range jsonArr {

		if jsonArr[i].(map[string]interface{})["Txhash"].(string) == txHash[2:] {

			spendArr := jsonArr[i].(map[string]interface{})["Spend"].([]interface{})
			for i := range spendArr {

				temp, _ := strconv.ParseFloat(spendArr[i].(map[string]interface{})["Amount"].(string), 64)
				totalAmount += temp
			}

			break
		}
	}

	return totalAmount, nil
}
