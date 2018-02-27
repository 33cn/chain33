package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

var conn *grpc.ClientConn
var rd *rand.Rand
var c types.GrpcserviceClient
var ErrTest = errors.New("ErrTest")

var addrexec *account.Address

var addr string
var privGenesis, privkey crypto.PrivKey

const fee = 1e6
const secretLen = 32
const defaultAmount = 1e11

type Record struct {
	RecordId string `json:"recordId"`
	DocType  string `json:"docType"`
}

type Org struct {
	OrgId     string `json:"orgId"`
	OrgCredit int64  `json:"orgCredit"`
}

type OrgId struct {
	OrgId string `json:"orgId"`
}

//parpare an account
func init() {
	var err error
	conn, err = grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	rd = rand.New(rand.NewSource(time.Now().UnixNano()))
	c = types.NewGrpcserviceClient(conn)
	addrexec = account.ExecAddress("norm")
	privGenesis = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
}

func main() {
	common.SetLogLevel("eror")
	paranum := len(os.Args)

	if paranum == 1 { //http mode

		http.HandleFunc("/putOrg", putOrgHttp)
		http.HandleFunc("/getOrg", getOrgHttp)
		//http.HandleFunc("/putBlackRecord", putBlackRecordHttp)
		//http.HandleFunc("/getBlackRecord", getBlackRecordHttp)

		err := http.ListenAndServe(":"+strconv.Itoa(8081), nil)
		if err != nil {
			fmt.Println(err)
		}
	} else if paranum > 1 { //cli mode

		argsWithoutProg := os.Args[1:]
		switch argsWithoutProg[0] {
		case "putBlackRecord":
			if len(argsWithoutProg) != 3 {
				fmt.Print(errors.New("参数错误").Error())
				return
			}
			putBlackRecord(argsWithoutProg[1], argsWithoutProg[2])
		case "getBlackRecord":
			if len(argsWithoutProg) != 2 {
				fmt.Print(errors.New("参数错误").Error())
				return
			}
			getBlackRecord(argsWithoutProg[1])
		case "putOrg":
			if len(argsWithoutProg) != 3 {
				fmt.Print(errors.New("参数错误").Error())
				return
			}
			putOrg(argsWithoutProg[1], argsWithoutProg[2])

		case "getOrg":
			if len(argsWithoutProg) != 2 {
				fmt.Print(errors.New("参数错误").Error())
				return
			}
			getOrg(argsWithoutProg[1])

		default:
			fmt.Print("指令错误")
		}
	}
}

func getOrgHttp(w http.ResponseWriter, r *http.Request) {
	fmt.Println("getOrgHttp")

	//if r.Method != "POST" {
	//return
	//}
	var orgId string

	orgId = r.Header.Get("orgId")

	fmt.Println(orgId)

	var req types.Query
	req.Execer = []byte("norm")
	req.FuncName = "NormGet"

	req.Payload = []byte("org" + orgId)

	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Println(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return
	}
	value := strings.TrimSpace(string(reply.Msg))

	var org Org
	err = json.Unmarshal([]byte(value), &org)
	if err != nil {
		err = json.Unmarshal([]byte(value[1:len(value)]), &org)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println("org =", org)
	OutputJson(w, &org)
}

func putOrgHttp(w http.ResponseWriter, r *http.Request) {
	fmt.Println("putOrgHttp")
	if r.Method != "POST" {
		return
	}

	body, _ := ioutil.ReadAll(r.Body)

	//fmt.Println(body)

	//body体转化为请求结构体
	var org Org
	err := json.Unmarshal(body, &org)
	if err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Println(org)

	//rpc operation
	var testKey string
	var testValue []byte

	testKey = "org" + org.OrgId
	testValue, _ = json.Marshal(org)
	//fmt.Println(testValue)

	vput := &types.NormAction_Nput{&types.NormPut{Key: testKey, Value: testValue}}
	transfer := &types.NormAction{Value: vput, Ty: types.NormActionPut}
	tx := &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: fee}
	tx.Nonce = rd.Int63()
	tx.Sign(types.SECP256K1, privGenesis)
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Println(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return
	}
	return
}

func putOrg(orgId string, credit string) {
	fmt.Println("putOrg")
	creditInt64, _ := strconv.ParseInt(credit, 10, 64)
	var testKey string
	var testValue []byte

	org := &Org{OrgId: orgId, OrgCredit: creditInt64}
	testKey = "org" + org.OrgId
	testValue, _ = json.Marshal(org)
	fmt.Println(testValue)

	vput := &types.NormAction_Nput{&types.NormPut{Key: testKey, Value: testValue}}
	transfer := &types.NormAction{Value: vput, Ty: types.NormActionPut}
	tx := &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: fee}
	tx.Nonce = rd.Int63()
	tx.Sign(types.SECP256K1, privGenesis)
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Println(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return
	}
}

func getOrg(orgId string) {
	fmt.Println("getOrg")

	var req types.Query
	req.Execer = []byte("norm")
	req.FuncName = "NormGet"
	req.Payload = []byte("org" + orgId)
	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Println("err")
		return
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return
	}
	value := strings.TrimSpace(string(reply.Msg))
	fmt.Println("GetValue =", value)

	var org Org
	err = json.Unmarshal([]byte(value), &org)
	if err != nil {
		err = json.Unmarshal([]byte(value[1:len(value)]), &org)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println("org =", org)

}

//put the things in []byte
func putBlackRecord(recordId string, docType string) {
	fmt.Println("putBlackRecord")

	var testKey string
	var testValue []byte

	record := &Record{RecordId: recordId, DocType: docType}
	testKey = "blackRecord" + record.RecordId
	testValue, _ = json.Marshal(record)
	fmt.Println(testValue)

	vput := &types.NormAction_Nput{&types.NormPut{Key: testKey, Value: testValue}}
	transfer := &types.NormAction{Value: vput, Ty: types.NormActionPut}
	tx := &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: fee}
	tx.Nonce = rd.Int63()
	tx.Sign(types.SECP256K1, privGenesis)
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Println(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return
	}
}

func getBlackRecord(recordId string) {
	fmt.Println("getBlackRecord")

	var req types.Query
	req.Execer = []byte("norm")
	req.FuncName = "NormGet"
	req.Payload = []byte("blackRecord" + recordId)
	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Println("err")
		return
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return
	}
	value := strings.TrimSpace(string(reply.Msg))
	fmt.Println("GetValue =", value)

	var record Record
	err = json.Unmarshal([]byte(value), &record)
	if err != nil {
		err = json.Unmarshal([]byte(value[1:len(value)]), &record)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	fmt.Println("record =", record)
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

func OutputJson(w http.ResponseWriter, out interface{}) {
	//out := &Rst{code, reason, data}
	b, err := json.Marshal(out)
	if err != nil {
		fmt.Println("OutputJson fail:" + err.Error())
		return
	}

	w.Write(b)
}
