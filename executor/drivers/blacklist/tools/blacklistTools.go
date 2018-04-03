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

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	blacklist "gitlab.33.cn/chain33/chain33/executor/drivers/blacklist/types"
	"gitlab.33.cn/chain33/chain33/types"

	//"gitlab.33.cn/chain33/chain33/executor/drivers/blacklist"
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
	//case "transferperf": //performance in transfer
	//	if len(argsWithoutProg) != 6 {
	//		fmt.Print(errors.New("参数错误").Error())
	//		return
	//	}
	//TransferPerf(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4], argsWithoutProg[5])
	//case "sendtoaddress": //发送到地址
	//	if len(argsWithoutProg) != 5 {
	//		fmt.Print(errors.New("参数错误").Error())
	//		return
	//	}
	//	SendToAddress(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4])
	case "submitRecord": //发送到地址
		if len(argsWithoutProg) != 6 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		submitRecord(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4], argsWithoutProg[5])
	case "queryRecord":
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		queryRecord(argsWithoutProg[1], argsWithoutProg[2])
	case "queryRecordByName":
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		queryRecordByName(argsWithoutProg[1], argsWithoutProg[2])
	case "createOrg": //发送到地址
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		CreateOrg(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	case "queryOrg": //发送到地址
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		QueryOrg(argsWithoutProg[1], argsWithoutProg[2])
	case "deleteRecord":
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		deleteRecord(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
		//NormGet(argsWithoutProg[1])
	case "login":
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		Login(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	case "registerUser":
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		RegisterUser(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4])
	case "queryTransaction":
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		QueryTransaction(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4])
	case "transfer":
		if len(argsWithoutProg) != 7 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		Transfer(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4], argsWithoutProg[5], argsWithoutProg[6])
	case "modifyUserPwd":
		if len(argsWithoutProg) != 5 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		ModifyUserPwd(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4])
	}
}

func LoadHelp() {
	fmt.Println("Available Commands:")
	fmt.Println("[ip] submitRecord [privkey ,orgId ,clientId ,clientName,recordId ]  : 发布黑名单记录")
	fmt.Println("[ip] queryRecord [privKey，recordId]                       : 查询黑名单记录")
	fmt.Println("[ip] queryRecordByName [privKey, name]                     : 根据名字查询黑名单记录")
	fmt.Println("[ip] createOrg [privkey, orgId, orgName]                   : 注册机构")
	fmt.Println("[ip] queryOrg [privkey, orgId]                             : 查询机构信息")
	fmt.Println("[ip] deleteRecord [privkey, orgId，recordId]               : 删除黑名单记录")
	fmt.Println("[ip] login [privkey, userName，passWord]                   : 用户登陆")
	fmt.Println("[ip] registerUser [privekey, userId,userName，passWord]    : 用户注册")
	fmt.Println("[ip] queryTransaction [privKey, txId，fromAddr，toAddr]    : 查询交易信息")
	fmt.Println("[ip] transfer [privkey ,txId ,from ,to ,docType ,credit ]  : 机构交易信息")
	fmt.Println("[ip] modifyUserPwd [privKey ,userId ,userName ,passWord ]  : 确认用户信息")
	//fmt.Println("[ip] transfer [privkey ,txId ,from ,to ,docType ,credit ]  : 机构交易信息")

}

//func TransferPerf(from string, to string, amount string, txNum string, duration string) {
//	txNumInt, err := strconv.Atoi(txNum)
//	if err != nil {
//		fmt.Fprintln(os.Stderr, err)
//		return
//	}
//	durInt, err := strconv.Atoi(duration)
//	if err != nil {
//		fmt.Fprintln(os.Stderr, err)
//		return
//	}
//	ch := make(chan struct{}, txNumInt)
//	for i := 0; i < txNumInt; i++ {
//		go func() {
//			txs := 0
//			for {
//				SendToAddress(from, to, amount, "test")
//				txs++
//				if durInt != 0 && txs == durInt {
//					break
//				}
//				time.Sleep(time.Second)
//			}
//			ch <- struct{}{}
//		}()
//	}
//	for j := 0; j < txNumInt; j++ {
//		<-ch
//	}
//}

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
func Login(privkey string, userName string, passWord string) {
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.FuncName_LoginCheck
	user := &blacklist.User{}
	user.UserName = userName
	user.PassWord = passWord
	query := &blacklist.Query{&blacklist.Query_LoginCheck{user}, "privkey"}
	req.Payload = types.Encode(query)

	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
	value := string(reply.GetMsg())
	fmt.Println("GetValue =", value)
}
func RegisterUser(privekey string, userId string, userName string, passWord string) {
	//TODO:这里应该有个判断用户是否已经注册过了
	user := &blacklist.User{}
	user.UserId = userId
	user.UserName = userName
	user.PassWord = passWord
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_User{user}, FuncName: blacklist.FuncName_RegisterUser}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: fee}
	tx.To = "user.blacklist"
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, getprivkey(privekey))

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
func CreateOrg(privkey string, orgId string, orgName string) {

	org := &blacklist.Org{}
	org.OrgId = orgId
	org.OrgName = orgName
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Or{org}, FuncName: blacklist.FuncName_CreateOrg}
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
func QueryOrg(privkey string, orgId string) {
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.FuncName_QueryOrgById
	qb := &blacklist.QueryOrgParam{}
	qb.OrgId = orgId
	fmt.Println(qb.OrgId)
	query := &blacklist.Query{&blacklist.Query_QueryOrg{qb}, privkey}
	req.Payload = types.Encode(query)

	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
	value := reply.String()
	fmt.Println("GetValue =", value)
}
func submitRecord(privkey string, orgId string, clientId string, clientName string, recordId string) {
	rc := &blacklist.Record{}
	rc.RecordId = recordId
	rc.OrgId = orgId
	rc.ClientId = clientId
	rc.ClientName = clientName
	rc.Searchable = true
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Rc{rc}, FuncName: blacklist.FuncName_SubmitRecord}
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
func queryRecord(privKey string, recordId string) {
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.FuncName_QueryRecordById
	qb := &blacklist.QueryRecordParam{}
	qb.ByRecordId = recordId
	query := &blacklist.Query{&blacklist.Query_QueryRecord{qb}, privKey}
	req.Payload = types.Encode(query)

	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
	value := string(reply.Msg[:])
	fmt.Println("GetValue =", value)
}
func deleteRecord(privkey string, orgId string, recordId string) {
	rc := &blacklist.Record{}
	rc.OrgId = orgId
	rc.RecordId = recordId
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Rc{rc}, FuncName: blacklist.FuncName_DeleteRecord}
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
func Transfer(privkey string, txId string, from string, to string, docType string, credit string) {
	tr := &blacklist.Transaction{}
	tr.TxId = txId
	tr.From = from
	tr.To = to
	tr.DocType = docType
	tr.Credit, _ = strconv.ParseInt(credit, 10, 64)
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Tr{tr}, FuncName: blacklist.FuncName_Transfer}
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
func QueryTransaction(privKey string, txId string, fromAddr string, toAddr string) {
	qt := &blacklist.QueryTransactionParam{}
	qt.ByTxId = txId
	qt.ByFromAddr = fromAddr
	qt.ByToAddr = toAddr
	query := &blacklist.Query{&blacklist.Query_QueryTransaction{qt}, privKey}
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.Payload = types.Encode(query)
	if qt.ByTxId != "" {
		req.FuncName = blacklist.FuncName_QueryTxById
	} else if qt.ByFromAddr != "" {
		req.FuncName = blacklist.FuncName_QueryTxByFromAddr
	} else if qt.ByToAddr != "" {
		req.FuncName = blacklist.FuncName_QueryTxByToAddr
	} else {
		return
	}
	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
	value := string(reply.GetMsg())
	fmt.Println("GetValue =", value)
}
func ModifyUserPwd(privKey string, userId string, userName string, passWord string) {
	//TODO:这里应该有个oldPwd校验
	user := &blacklist.User{}
	user.UserId = userId
	user.UserName = userName
	user.PassWord = passWord
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_User{user}, FuncName: blacklist.FuncName_ModifyUserPwd}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: fee}
	tx.To = "user.blacklist"
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, getprivkey(privKey))

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
func queryRecordByName(privKey string, name string) {
	var req types.Query
	qb := &blacklist.QueryRecordParam{}
	qb.ByClientName = name
	query := &blacklist.Query{&blacklist.Query_QueryRecord{qb}, privKey}
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.FuncName_QueryRecordByName
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
