package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	rlog "gitlab.33.cn/chain33/chain33/common/log"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/norm/types"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

const fee = 1e6
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var conn *grpc.ClientConn
var c types.Chain33Client
var r *rand.Rand

func createConn(ip string) {
	var err error
	url := ip + ":8802"
	fmt.Println("grpc url:", url)
	conn, err = grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	c = types.NewChain33Client(conn)
	r = rand.New(rand.NewSource(types.Now().UnixNano()))
}

func main() {
	rlog.SetLogLevel("eror")
	if len(os.Args) == 1 || os.Args[1] == "-h" {
		LoadHelp()
		return
	}
	createConn(os.Args[1])
	argsWithoutProg := os.Args[2:]
	switch argsWithoutProg[0] {
	case "-h": //使用帮助
		LoadHelp()
	case "transferperf":
		if len(argsWithoutProg) != 6 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		TransferPerf(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4], argsWithoutProg[5])
	case "sendtoaddress":
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SendToAddress(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4])
	case "normperf":
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		NormPerf(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4])
	case "normput":
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		NormPut(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	case "normget":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		NormGet(argsWithoutProg[1])
		//zzh
	case "normreadperf":
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		NormReadPerf(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	}
}

func LoadHelp() {
	fmt.Println("Available Commands:")
	fmt.Println("[ip] transferperf [from, to, amount, txNum, duration]            : 转账性能测试")
	fmt.Println("[ip] sendtoaddress [from, to, amount, note]                      : 发送交易到地址")
	fmt.Println("[ip] normperf [size, num, interval, duration]                    : 常规写数据性能测试")
	fmt.Println("[ip] normput [privkey, key, value]                               : 常规写数据")
	fmt.Println("[ip] normget [key]                                               : 常规读数据")
	fmt.Println("[ip] normreadperf [num, interval, duration]                      : 常规读数据性能测试")
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
	tx := &types.ReqWalletSendToAddress{From: from, To: to, Amount: amountInt64 * 1e4, Note: note}

	reply, err := c.SendToAddress(context.Background(), tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	data, err := json.MarshalIndent(reply, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func NormPerf(size string, num string, interval string, duration string) {
	var key string
	var value string
	var numThread int
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
	intervalInt, err := strconv.Atoi(interval)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	durInt, err := strconv.Atoi(duration)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if numInt < 10 {
		numThread = 1
	} else if numInt > 100 {
		numThread = 10
	} else {
		numThread = numInt / 10
	}
	maxTxPerAcc := 50
	ch := make(chan struct{}, numThread)
	for i := 0; i < numThread; i++ {
		go func() {
			var result int64 = 0
			totalCount := 0
			txCount := 0
			_, priv := genaddress()
			for sec := 0; durInt == 0 || sec < durInt; {
				start := time.Now()
				for txs := 0; txs < numInt/numThread; txs++ {
					if txCount >= maxTxPerAcc {
						_, priv = genaddress()
						txCount = 0
					}
					key = RandStringBytes(20)
					value = RandStringBytes(sizeInt)
					NormPut(common.ToHex(priv.Bytes()), key, value)
					txCount++
					totalCount++
				}
				end := time.Now()
				result += end.Sub(start).Nanoseconds() / 1000000
				time.Sleep(time.Second * time.Duration(intervalInt))
				sec += intervalInt
			}
			fmt.Println("perform put ", totalCount, " times, cost time [ms]:", result)
			ch <- struct{}{}
		}()
	}
	for j := 0; j < numThread; j++ {
		<-ch
	}
}

//zzh
func NormReadPerf(num string, interval string, duration string) {
	var numThread int
	numInt, err := strconv.Atoi(num)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	intervalInt, err := strconv.Atoi(interval)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	durInt, err := strconv.Atoi(duration)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if numInt < 10 {
		numThread = 1
	} else if numInt > 100 {
		numThread = 10
	} else {
		numThread = numInt / 10
	}

	ch := make(chan struct{}, numThread)
	for i := 0; i < numThread; i++ {
		go func() {
			f, err := os.Open("normperf.log")
			if err != nil {
				panic("open file failed.")
				return
			}
			buf := bufio.NewReader(f)
			cnt := 0
			var result, totalTime int64
			result = 0
			totalTime = 0
			totalCount := 0
			for sec := 0; durInt == 0 || sec < durInt; {
				start := time.Now()
				for txs := 0; txs < numInt/numThread; txs++ {
					line, err := buf.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							f, err := os.Open("normperf.log")
							if err != nil {
								panic("open file failed.")
								return
							}
							buf = bufio.NewReader(f)
						}
						continue
					}
					line = strings.Replace(line, " ", "", -1)
					index := strings.IndexAny(line, "=")
					if index > 0 {
						prefix := []byte(line)[0:index]
						NormGet(string(prefix))
						cnt++
						totalCount++
					} else {
						continue
					}
				}
				end := time.Now()
				result += end.Sub(start).Nanoseconds() / 1000000
				if cnt > 100 {
					fmt.Println("normal get ", cnt, "times, cost time [ms]:", result)
					totalTime += result
					cnt = 0
					result = 0
				}
				time.Sleep(time.Second * time.Duration(intervalInt))
				sec += intervalInt
			}
			totalTime += result
			fmt.Println("perform total get ", totalCount, " times, cost total time [ms]:", totalTime)
			ch <- struct{}{}
		}()
	}
	for j := 0; j < numThread; j++ {
		<-ch
	}
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	rand.Seed(types.Now().UnixNano())
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func NormPut(privkey string, key string, value string) {
	fmt.Println(key, "=", value)
	nput := &pty.NormAction_Nput{&pty.NormPut{Key: key, Value: []byte(value)}}
	action := &pty.NormAction{Value: nput, Ty: pty.NormActionPut}
	tx := &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(action), Fee: fee}
	tx.To = address.ExecAddress("norm")
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, getprivkey(privkey))

	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
}

func NormGet(key string) {
	in := &pty.NormGetKey{Key: key}
	data, err := proto.Marshal(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	var req types.ChainExecutor
	req.Driver = "norm"
	req.FuncName = "NormGet"
	req.Param = data

	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
	//the first two byte is not valid
	//QueryChain() need to change
	//value := string(reply.Msg[2:])
	//fmt.Println("GetValue =", value)
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
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

func genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := address.PubKeyToAddress(privto.PubKey().Bytes())
	fmt.Println("addr:", addrto.String())
	return addrto.String(), privto
}
