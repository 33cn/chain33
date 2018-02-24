package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"code.aliyun.com/chain33/chain33/common"
	jsonrpc "code.aliyun.com/chain33/chain33/rpc"
	"code.aliyun.com/chain33/chain33/types"
)

const fee = 1e6

var rpc *jsonrpc.JsonClient

func init() {
	var err error
	rpc, err = jsonrpc.NewJsonClient("http://localhost:8801")
	if err != nil {
		panic(err)
	}
}

func main() {
	common.SetLogLevel("eror")
	if len(os.Args) == 1 {
		LoadHelp()
		return
	}
	argsWithoutProg := os.Args[1:]
	switch argsWithoutProg[0] {
	case "-h": //使用帮助
		LoadHelp()
	case "transferperf": //performance in transfer
		if len(argsWithoutProg) != 6 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		TransferPerf(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4], argsWithoutProg[5])
	case "sendtoaddress": //发送到地址
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SendToAddress(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4])
	case "normperf": //发送到地址
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		NormPerf(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	case "normput": //发送到地址
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		NormPut(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	case "normget": //发送到地址
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		NormGet(argsWithoutProg[1])
	}
}

func LoadHelp() {
	fmt.Println("Available Commands:")
	fmt.Println("transferperf [from, to, amount, txNum, duration]            : 转账性能测试")
	fmt.Println("sendtoaddress [from, to, amount, note]                      : 发送交易到地址")
	fmt.Println("normperf [privkey, num, duration]                           : 常规写数据性能测试")
	fmt.Println("normput [privkey, key, value]                               : 常规写数据")
	fmt.Println("normget [key]                                               : 常规读数据")
}

func TransferPerf(from string, to string, amount string, txNum string, duration string) {
	txNumInt, err := strconv.Atoi(txNum)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	durInt, err := strconv.Atoi(duration)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	ch := make(chan struct{}, txNumInt)
	for i := 0; i < txNumInt; i++ {
		go func() {
			txs := 0
			for {
				SendToAddress(from, to, amount, "test")
				txs++
				if durInt != 0 && txs == durInt {
					break
				}
				time.Sleep(time.Second)
			}
			ch <- struct{}{}
		}()
	}
	for j := 0; j < txNumInt; j++ {
		<-ch
	}
}

func SendToAddress(from string, to string, amount string, note string) {
	amountFloat64, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	amountInt64 := int64(amountFloat64 * 1e4)
	params := types.ReqWalletSendToAddress{From: from, To: to, Amount: amountInt64 * 1e4, Note: note}

	var res jsonrpc.ReplyHash
	err = rpc.Call("Chain33.SendToAddress", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func NormPerf(privkey string, key string, value string) {}

func NormPut(privkey string, key string, value string) {
	nput := &types.NormAction_Nput{&types.NormPut{Key: key, Value: value}}
	action := &types.NormAction{Value: nput, Ty: types.NormActionPut}
	tx := &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(action), Fee: fee}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey)

	var res jsonrpc.ReplyHash
	err = rpc.Call("Chain33.SendTransaction", tx, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func NormGet(key string) {
	var req types.Query
	req.Execer = []byte("norm")
	req.FuncName = "NormGet"
	req.Payload = []byte(testKey)

	var res jsonrpc.ReplyHash
	err = rpc.Call("Chain33.QueryChain", req, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}
