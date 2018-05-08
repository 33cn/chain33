package tests

import (
	"strconv"
	"fmt"
	"encoding/hex"
)

func getBin(data string) (ret []byte) {
	ret,err := hex.DecodeString(data)
	if err != nil{
		fmt.Println(err)
	}
	return
}

func parseData(data map[string]interface{}) (cases []VMCase) {
	for k, v := range data {
		ut := VMCase{name: k}
		parseVMCase(v, &ut)
		cases = append(cases, ut)
	}
	return
}

func parseVMCase(data interface{}, ut *VMCase) {
	m := data.(map[string]interface{})
	for k, v := range m {
		switch k {
		case "env":
			ut.env = parseEnv(v)
		case "exec":
			ut.exec = parseExec(v)
		case "pre":
			ut.pre = parseAccount(v)
		case "post":
			ut.post = parseAccount(v)
		case "gas":
			ut.gas = toint64(unpre(v.(string)))
		case "logs":
			ut.logs = unpre(v.(string))
		case "out":
			ut.out = unpre(v.(string))
		case "err":
			ut.err = v.(string)
			if ut.err == "{{.Err}}" {
				ut.err = ""
			}
		default:
			//fmt.Println(k, "is of a type I don't know how to handle")
		}
	}
}

func parseEnv(data interface{}) EnvJson {
	m := data.(map[string]interface{})
	ut := EnvJson{}
	for k, v := range m {
		switch k {
		case "currentCoinbase":
			ut.currentCoinbase = unpre(v.(string))
		case "currentDifficulty":
			ut.currentDifficulty = toint64(unpre(v.(string)))
		case "currentGasLimit":
			ut.currentGasLimit = toint64(unpre(v.(string)))
		case "currentNumber":
			ut.currentNumber = toint64(unpre(v.(string)))
		case "currentTimestamp":
			ut.currentTimestamp = toint64(unpre(v.(string)))
		default:
			fmt.Println(k, "is of a type I don't know how to handle")
		}
	}
	return ut
}

func parseExec(data interface{}) ExecJson {
	m := data.(map[string]interface{})
	ut := ExecJson{}
	for k, v := range m {
		switch k {
		case "address":
			ut.address = unpre(v.(string))
		case "caller":
			ut.caller = unpre(v.(string))
		case "code":
			ut.code = unpre(v.(string))
		case "data":
			ut.data = unpre(v.(string))
		case "gas":
			ut.gas = toint64(unpre(v.(string)))
		case "gasPrice":
			ut.gasPrice = toint64(unpre(v.(string)))
		case "origin":
			ut.origin = unpre(v.(string))
		case "value":
			ut.value = toint64(unpre(v.(string)))/100000000
		default:
			fmt.Println(k, "is of a type I don't know how to handle")
		}
	}
	return ut
}

func parseAccount(data interface{}) map[string]AccountJson {
	ret := make(map[string]AccountJson)
	m := data.(map[string]interface{})
	for k, v := range m {
		ret[unpre(k)] = parseAccount2(v)
	}
	return ret
}

func parseAccount2(data interface{}) AccountJson {
	m := data.(map[string]interface{})
	ut := AccountJson{}
	for k, v := range m {
		switch k {
		case "balance":
			ut.balance = toint64(unpre(v.(string)))/100000000
		case "code":
			ut.code = unpre(v.(string))
		case "nonce":
			ut.nonce = toint64(unpre(v.(string)))
		case "storage":
			ut.storage = parseStorage(v)
		default:
			fmt.Println(k, "is of a type I don't know how to handle")
		}
	}
	return ut
}

func parseStorage(data interface{}) map[string]string {
	ret := make(map[string]string)
	m := data.(map[string]interface{})
	for k, v := range m {
		ret[unpre(k)] = unpre(v.(string))
	}
	return ret
}

func toint64(data string) int64 {
	if len(data) == 0 {
		return 0
	}

	val, err := strconv.ParseInt(data, 16, 64)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	return val
}

// 去掉十六进制字符串前面的0x
func unpre(data string) string {
	if len(data) > 1 && data[:2] == "0x" {
		return data[2:]
	}
	return data
}