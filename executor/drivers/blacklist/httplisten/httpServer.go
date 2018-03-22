package httplisten

import (
	"net/http"
	"strconv"
	"fmt"
	"time"
	"math/rand"
	"context"
	"errors"
	"encoding/json"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
	blacklist "code.aliyun.com/chain33/chain33/executor/drivers/blacklist/types"
	"os"
)

var conn *grpc.ClientConn
var rd *rand.Rand
var c types.GrpcserviceClient

var addrexec *account.Address

var addr string
var privGenesis, privkey crypto.PrivKey

const fee = 1e6
func Init(){
	var err error
	conn, err = grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	rd = rand.New(rand.NewSource(time.Now().UnixNano()))
	c = types.NewGrpcserviceClient(conn)
	httpListen()
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
func httpListen(){
	http.HandleFunc("/login", login)
	http.HandleFunc("/registerUser", registerUser)
	http.HandleFunc("/createOrg", createOrg)
	http.HandleFunc("/queryOrg", queryOrg)
	http.HandleFunc("/submitRecord", submitRecord)
	http.HandleFunc("/queryRecord", queryRecord)
	http.HandleFunc("/transfer", transfer)
	http.HandleFunc("/queryTransaction", queryTransaction)
	http.HandleFunc("/deleteRecord", deleteRecord)
	http.HandleFunc("/modifyUserPwd", modifyUserPwd)
	http.HandleFunc("/resetUserPwd", resetUserPwd)
	//http.HandleFunc("/queryOrg", queryOrg)

	err := http.ListenAndServe(":"+strconv.Itoa(8081), nil)
	if err != nil {
		fmt.Println(err)
	}
}
func responseByJson(w http.ResponseWriter, out interface{}) {
	//out := &Rst{code, reason, data}
	b, err := json.Marshal(out)
	if err != nil {
		fmt.Println("Marsha Json fail:" + err.Error())
		return
	}

	w.Write(b)
}
func login(w http.ResponseWriter, r *http.Request)  {
	if r.Method !=http.MethodGet{
		return
	}
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.FuncName_LoginCheck
	user := &blacklist.User{}
    user.UserName=r.Header.Get("userName")
    user.PassWord=r.Header.Get("passWord")
	query := &blacklist.Query{&blacklist.Query_LoginCheck{user},""}
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
	value := string(reply.Msg[:])
	fmt.Println("GetValue =", value)
	//w.WriteHeader(200)
	responseByJson(w,value)
}
func registerUser(w http.ResponseWriter, r *http.Request)  {
	if r.Method !=http.MethodPost{
		return
	}
	//登陆失败判断用户不存在时注册用户
	user := &blacklist.User{
	}
	user.UserId=r.Header.Get("userId")
	user.UserName=r.Header.Get("userName")
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_User{user},FuncName:blacklist.FuncName_RegisterUser}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: fee}
	tx.To = "user.blacklist"
	tx.Nonce = rd.Int63()
	tx.Sign(types.SECP256K1, getprivkey(r.Header.Get("privateKey")))

	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
	w.WriteHeader(http.StatusCreated)
}
func createOrg(w http.ResponseWriter, r *http.Request)  {
	if r.Method !=http.MethodPost{
		return
	}
	org := &blacklist.Org{
	}
	org.OrgId=r.Header.Get("orgId")
	org.OrgName=r.Header.Get("orgName")
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Or{org},FuncName:blacklist.FuncName_CreateOrg}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: fee}
	tx.To = "user.blacklist"
	tx.Nonce = rd.Int63()
	tx.Sign(types.SECP256K1, getprivkey(r.Header.Get("privateKey")))

	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
	w.WriteHeader(http.StatusCreated)
}
func queryOrg(w http.ResponseWriter, r *http.Request)  {
	if r.Method !=http.MethodPost{
		return
	}
	var req types.Query
	req.Execer = []byte("uesr.blacklist")
	req.FuncName = blacklist.FuncName_QueryOrgById
	qb := &blacklist.QueryOrgParam{}
	qb.OrgId = r.Header.Get("orgId")
	query := &blacklist.Query{&blacklist.Query_QueryOrg{qb},r.Header.Get("privateKey")}
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
	value := string(reply.Msg[:])
	fmt.Println("GetValue =", value)
	w.WriteHeader(http.StatusCreated)
}
func submitRecord(w http.ResponseWriter, r *http.Request)  {
	if r.Method != http.MethodPost{
		return
	}
	rc := &blacklist.Record{
	}
	rc.OrgId=r.Header.Get("orgId")
	rc.ClientId=r.Header.Get("clientId")
	rc.ClientName=r.Header.Get("clientName")
	rc.Searchable=true
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Rc{rc},FuncName:blacklist.FuncName_SubmitRecord}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: fee}
	tx.To = "user.blacklist"
	tx.Nonce = rd.Int63()
	tx.Sign(types.SECP256K1, getprivkey(r.Header.Get("privateKey")))

	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
	w.WriteHeader(http.StatusCreated)
}
func queryRecord(w http.ResponseWriter, r *http.Request)  {
	if r.Method != http.MethodGet{
		return
	}
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.FuncName_QueryRecordById
	qb := &blacklist.QueryRecordParam{}
	qb.ByClientId=r.Header.Get("recordId")
	query := &blacklist.Query{&blacklist.Query_QueryRecord{qb},r.Header.Get("privateKey")}
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
	value := string(reply.Msg[:])
	fmt.Println("GetValue =", value)
	w.WriteHeader(http.StatusCreated)
}
func transfer(w http.ResponseWriter, r *http.Request)  {
	if r.Method != http.MethodPost{
		return
	}
	rc := &blacklist.Transaction{
	}
	rc.TxId=r.Header.Get("txId")
	rc.From=r.Header.Get("from")
	rc.To=r.Header.Get("to")
	rc.DocType=r.Header.Get("docType")
	rc.Credit,_=strconv.ParseInt(r.Header.Get("credit"),10,64)
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Tr{rc},FuncName:blacklist.FuncName_Transfer}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: fee}
	tx.To = "user.blacklist"
	tx.Nonce = rd.Int63()
	tx.Sign(types.SECP256K1, getprivkey(r.Header.Get("privateKey")))

	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
	w.WriteHeader(http.StatusCreated)
}
func queryTransaction(w http.ResponseWriter, r *http.Request)  {

}
func deleteRecord(w http.ResponseWriter, r *http.Request)  {
	if r.Method != http.MethodPost{
		return
	}
	rc := &blacklist.Record{
	}
	rc.OrgId=r.Header.Get("orgId")
	rc.RecordId=r.Header.Get("recordId")
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Rc{rc},FuncName:blacklist.FuncName_DeleteRecord}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: fee}
	tx.To = "user.blacklist"
	tx.Nonce = rd.Int63()
	tx.Sign(types.SECP256K1, getprivkey(r.Header.Get("privateKey")))

	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
	w.WriteHeader(http.StatusCreated)
}
func modifyUserPwd(w http.ResponseWriter, r *http.Request)  {

}
func resetUserPwd(w http.ResponseWriter, r *http.Request)  {

}