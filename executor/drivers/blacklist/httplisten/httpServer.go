package httplisten

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	blacklist "gitlab.33.cn/chain33/chain33/executor/drivers/blacklist/types"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
	"os"
	"strings"
)

var conn *grpc.ClientConn
var rd *rand.Rand
var c types.GrpcserviceClient

var addrexec *account.Address

var addr string
var privGenesis, privkey crypto.PrivKey

const fee = 1e6

func Init() {
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
func httpListen() {
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
func responseMsg(w http.ResponseWriter, out interface{}) {

	b, err := json.Marshal(out)
	if err != nil {
		fmt.Println("Marsha Json fail:" + err.Error())
		return
	}
	w.Write(b)
}
func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(400)
		return
	}
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.FuncName_LoginCheck
	user := &blacklist.User{}
	user.UserName = r.Header.Get("userName")
	user.PassWord = r.Header.Get("passWord")
	query := &blacklist.Query{&blacklist.Query_LoginCheck{user}, ""}
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
	responseMsg(w, value)
}
func registerUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(400)
		return
	}
	//判断用户是否已经注册过了
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.FuncName_CheckKeyIsExsit
	key := &blacklist.QueryKeyParam{Key: r.Header.Get("userName")}
	query := &blacklist.Query{&blacklist.Query_QueryKey{key}, ""}
	req.Payload = types.Encode(query)

	respon, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !respon.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(respon.GetMsg())))
		return
	}

	value := strings.Replace(string(respon.GetMsg()[2:]),"","",-1)
	fmt.Println("Value =", value)
	fmt.Println("GetValue =", strings.TrimSpace(value))
	if strings.TrimSpace(value) != blacklist.FAIL {
		w.Header().Add("error", "the userName have been registered!")
		w.WriteHeader(403)
	}
	//如果没有注册，则进行注册
	user := &blacklist.User{}
	user.UserId = r.Header.Get("userId")
	user.UserName = r.Header.Get("userName")
	user.PassWord = r.Header.Get("passWord")
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_User{user}, FuncName: blacklist.FuncName_RegisterUser}
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
func createOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(400)
		return
	}
	org := &blacklist.Org{}
	org.OrgId = r.Header.Get("orgId")
	org.OrgName = r.Header.Get("orgName")
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Or{org}, FuncName: blacklist.FuncName_CreateOrg}
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
func queryOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(400)
		return
	}
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.FuncName_QueryOrgById
	qb := &blacklist.QueryOrgParam{}
	qb.OrgId = r.Header.Get("orgId")
	fmt.Println(qb.OrgId)
	query := &blacklist.Query{&blacklist.Query_QueryOrg{qb}, r.Header.Get("privateKey")}
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
	responseMsg(w, value)
}
func submitRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(400)
		return
	}
	rc := &blacklist.Record{}
	//rc.RecordId =r.Header.Get("recordId")
	//rc.OrgId = r.Header.Get("orgId")
	//rc.ClientId = r.Header.Get("clientId")
	//rc.ClientName = r.Header.Get("clientName")
	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println("body==============",body)
	err := json.Unmarshal(body, rc)
	fmt.Println("err==============",err)
	if err != nil {
		w.WriteHeader(403)
		return
	}
	rc.Searchable = true
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Rc{rc}, FuncName: blacklist.FuncName_SubmitRecord}
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
func queryRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(400)
		return
	}
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.FuncName_QueryRecordById
	qb := &blacklist.QueryRecordParam{}
	qb.ByRecordId = r.Header.Get("recordId")
	query := &blacklist.Query{&blacklist.Query_QueryRecord{qb}, r.Header.Get("privateKey")}
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
	responseMsg(w, value)
}
func transfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(400)
		return
	}
	tr := &blacklist.Transaction{}
	tr.TxId = r.Header.Get("txId")
	tr.From = r.Header.Get("from")
	tr.To = r.Header.Get("to")
	tr.DocType = r.Header.Get("docType")
	tr.Credit, _ = strconv.ParseInt(r.Header.Get("credit"), 10, 64)
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Tr{tr}, FuncName: blacklist.FuncName_Transfer}
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
func queryTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(400)
		return
	}
	qt := &blacklist.QueryTransactionParam{}
	qt.ByTxId = r.Header.Get("txId")
	qt.ByFromAddr = r.Header.Get("fromAddr")
	qt.ByToAddr = r.Header.Get("toAddr")
	query := &blacklist.Query{&blacklist.Query_QueryTransaction{qt}, r.Header.Get("privateKey")}
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
		w.WriteHeader(400)
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
	responseMsg(w, value)
}
func deleteRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(400)
		return
	}
	rc := &blacklist.Record{}
	rc.OrgId = r.Header.Get("orgId")
	rc.RecordId = r.Header.Get("recordId")
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_Rc{rc}, FuncName: blacklist.FuncName_DeleteRecord}
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
	w.WriteHeader(http.StatusAccepted)
}
func modifyUserPwd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(400)
		return
	}
	//oldPwd校验
	var req types.Query
	req.Execer = []byte("user.blacklist")
	req.FuncName = blacklist.FuncName_LoginCheck
	user := &blacklist.User{}
	user.UserName = r.Header.Get("userName")
	user.PassWord = r.Header.Get("oldPwd")
	query := &blacklist.Query{&blacklist.Query_LoginCheck{user}, ""}
	req.Payload = types.Encode(query)

	respon, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !respon.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(respon.GetMsg())))
		return
	}
	//msg := string(respon.GetMsg()[2:])
	//fmt.Println("msg =",msg)
	value := strings.Replace(string(respon.GetMsg()[2:]),"","",-1)

	fmt.Println("GetValue =",strings.TrimSpace(value))

	if strings.TrimSpace(value) != blacklist.SUCESS {
		w.WriteHeader(403)
	}
	//设置新的密码
	user.PassWord = r.Header.Get("newPwd")
	action := &blacklist.BlackAction{Value: &blacklist.BlackAction_User{user}, FuncName: blacklist.FuncName_ModifyUserPwd}
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
func resetUserPwd(w http.ResponseWriter, r *http.Request) {
	//TODO:待实现
}
