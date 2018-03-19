package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"

	"code.aliyun.com/chain33/chain33/executor/drivers/blacklist"
	"google.golang.org/grpc"
)

const fee = 1e6
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var conn *grpc.ClientConn
var c types.GrpcserviceClient
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
	c = types.NewGrpcserviceClient(conn)
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func main() {
	common.SetLogLevel("eror")
	if len(os.Args) == 1 || os.Args[1] == "-h" {
		LoadHelp()
		return
	}
	createConn(os.Args[1])
	argsWithoutProg := os.Args[2:]
	switch argsWithoutProg[0] {
	case "-h": //使用帮助
		LoadHelp()
	case "transferperf": //performance in transfer
		if len(argsWithoutProg) != 6 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		//TransferPerf(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4], argsWithoutProg[5])
	case "sendtoaddress": //发送到地址
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		SendToAddress(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4])
	case "submitRecord": //发送到地址
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		submitRecord(argsWithoutProg[1])
	case  "queryRecord":
		if len(argsWithoutProg) !=3{
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		queryRecord(argsWithoutProg[1],argsWithoutProg[2])
	case "queryRecordByName":
		if len(argsWithoutProg) !=3{
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		queryRecordByName(argsWithoutProg[1],argsWithoutProg[2])
	case "createOrg": //发送到地址
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		CreateOrg(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	case "queryOrg": //发送到地址
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
	case "deleteRecord":
		if len(argsWithoutProg) != 4{
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		deleteRecord(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
		//NormGet(argsWithoutProg[1])
	}
}

func LoadHelp() {
	fmt.Println("Available Commands:")
	fmt.Println("[ip] transferperf [from, to, amount, txNum, duration]            : 转账性能测试")
	fmt.Println("[ip] sendtoaddress [from, to, amount, note]                      : 发送交易到地址")
	fmt.Println("[ip] normperf [privkey, size, num, duration]                     : 常规写数据性能测试")
	fmt.Println("[ip] noneput [privkey, key, value]                               : 常规写数据")
	fmt.Println("[ip] normget [key]                                               : 常规读数据")
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

//func createOrg(privkey string, size string, num string, duration string) {
//	var key string
//	var value string
//	sizeInt, err := strconv.Atoi(size)
//	if err != nil {
//		fmt.Fprintln(os.Stderr, err)
//		return
//	}
//	numInt, err := strconv.Atoi(num)
//	if err != nil {
//		fmt.Fprintln(os.Stderr, err)
//		return
//	}
//	durInt, err := strconv.Atoi(duration)
//	if err != nil {
//		fmt.Fprintln(os.Stderr, err)
//		return
//	}
//	ch := make(chan struct{}, numInt)
//	for i := 0; i < numInt; i++ {
//		go func() {
//			txs := 0
//			for {
//				key = RandStringBytes(10)
//				value = RandStringBytes(sizeInt)
//				NonePut(privkey, key, value)
//				txs++
//				if durInt != 0 && txs == durInt {
//					break
//				}
//				time.Sleep(time.Second)
//			}
//			ch <- struct{}{}
//		}()
//	}
//	for j := 0; j < numInt; j++ {
//		<-ch
//	}
//}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func CreateOrg(privkey string, key string, value string) {
	fmt.Println(key, "=", value)

	org := &blacklist.Org{
	}
	org.OrgId="33"
	org.OrgName="fuzamei"
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Or{org},FuncName:blacklist.CreateOrg}
	//nput := &types.NormAction_Nput{&types.NormPut{Key: key,Value: []byte(value)}}
	//action := &types.NormAction{Value: nput, Ty: types.NormActionPut}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: fee}
	tx.To = "user.blacklist"
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
func submitRecord(privkey string){
	rc := &blacklist.Record{
	}
	rc.OrgId="33"
	rc.ClientId="one"
	rc.ClientName="dirk"
	rc.Searchable=true
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Rc{rc},FuncName:blacklist.SubmitRecord}
	//nput := &types.NormAction_Nput{&types.NormPut{Key: key,Value: []byte(value)}}
	//action := &types.NormAction{Value: nput, Ty: types.NormActionPut}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: fee}
	tx.To = "user.blacklist"
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
func queryRecord(privKey string ,recordId string) {
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.QueryRecord
	qb := &blacklist.QueryRecordParam{}
	qb.ByClientId=recordId
	query := &blacklist.Query{&blacklist.Query_QueryRecord{qb},privKey}
	req.Payload = []byte(query.String())

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
	value := string(reply.Msg[:])
	fmt.Println("GetValue =", value)
}
func deleteRecord(privkey string ,orgId string,recordId string) {
	rc := &blacklist.Record{
	}
	rc.OrgId=orgId
	rc.RecordId=recordId
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Rc{rc},FuncName:blacklist.DeleteRecord}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: fee}
	tx.To = "user.blacklist"
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
func queryRecordByName(privKey string,name string) {
	var req types.Query
	qb := &blacklist.QueryRecordParam{}
	qb.ByClientName=name
	query := &blacklist.Query{&blacklist.Query_QueryRecord{qb},privKey}
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.QueryRecordByName
	req.Payload = []byte(query.String())

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
	value := string(reply.Msg[:])
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
