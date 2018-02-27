package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	jsonrpc "code.aliyun.com/chain33/chain33/rpc"
	"code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

const fee = 1e6
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var rpc *jsonrpc.JsonClient
var r *rand.Rand

func init() {
	var err error
	rpc, err = jsonrpc.NewJsonClient("http://172.18.31.169:8801")
	if err != nil {
		panic(err)
	}
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
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
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		NormPerf(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4])
	case "normput": //发送到地址
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		NormPut(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	case "normget": //发送到地址
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		NormGet(argsWithoutProg[1], argsWithoutProg[2])
	}
}

func LoadHelp() {
	fmt.Println("Available Commands:")
	fmt.Println("transferperf [from, to, amount, txNum, duration]            : 转账性能测试")
	fmt.Println("sendtoaddress [from, to, amount, note]                      : 发送交易到地址")
	fmt.Println("normperf [privkey, size, num, duration]                     : 常规写数据性能测试")
	fmt.Println("normput [privkey, key, value]                               : 常规写数据")
	fmt.Println("normget [ip, key]                                           : 常规读数据")
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

func NormPerf(privkey string, size string, num string, duration string) {
	var key string
	var value string
	sizeInt, err := strconv.Atoi(size)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	numInt, err := strconv.Atoi(num)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	durInt, err := strconv.Atoi(duration)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	ch := make(chan struct{}, numInt)
	for i := 0; i < numInt; i++ {
		go func() {
			txs := 0
			for {
				key = RandStringBytes(10)
				value = RandStringBytes(sizeInt)
				fmt.Println(key, "=", value)
				NormPut(privkey, key, value)
				txs++
				if durInt != 0 && txs == durInt {
					break
				}
				time.Sleep(time.Second)
			}
			ch <- struct{}{}
		}()
	}
	for j := 0; j < numInt; j++ {
		<-ch
	}
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func NormPut(privkey string, key string, value string) {
	nput := &types.NormAction_Nput{&types.NormPut{Key: key, Value: []byte(value)}}
	action := &types.NormAction{Value: nput, Ty: types.NormActionPut}
	tx := &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(action), Fee: fee}
	//tx := &types.Transaction{Execer: []byte(key), Payload: []byte(value), Fee: fee, Expire: 0}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, getprivkey(privkey))
	params := jsonrpc.RawParm{Data: common.ToHex(types.Encode(tx))}

	var res string
	err := rpc.Call("Chain33.SendTransaction", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}

func NormGet(ip string, key string) {
	var req types.Query
	req.Execer = []byte("norm")
	req.FuncName = "NormGet"
	req.Payload = []byte(key)

	url := ip + ":8802"
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	c := types.NewGrpcserviceClient(conn)

	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
	value := strings.TrimSpace(string(reply.Msg))
	fmt.Println("GetValue =", value)
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}
