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
	//"strings"
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

type Information struct {
	RecordId string
	value    string
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
		http.HandleFunc("/putInformation", putInformation)
		http.HandleFunc("/getInformation", getInformation)

		err := http.ListenAndServe(":"+strconv.Itoa(8081), nil)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func getInformation(w http.ResponseWriter, r *http.Request) {
	fmt.Println("getInformation")

	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(body))

	var req types.Query
	req.Execer = []byte("norm")
	req.FuncName = "NormGet"

	req.Payload = body
	fmt.Println("getInformation 1")
	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("getInformation 2")
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return
	}

	fmt.Println(reply.GetMsg())
	w.Write(reply.GetMsg())

}

func putInformation(w http.ResponseWriter, r *http.Request) {
	fmt.Println("putInformation")
	if r.Method != "POST" {
		return
	}

	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(body)

	var info Information
	err := json.Unmarshal(body, &info)
	if err != nil {
		fmt.Println(err)
		return
	}

	//rpc operation
	var testKey string
	//var testValue []byte

	fmt.Println(info)

	testKey = info.RecordId
	//testValue, _ = json.Marshal(info)

	vput := &types.NormAction_Nput{&types.NormPut{Key: testKey, Value: body}}
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
