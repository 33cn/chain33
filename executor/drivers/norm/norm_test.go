package norm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

var conn *grpc.ClientConn
var r *rand.Rand
var c types.GrpcserviceClient
var ErrTest = errors.New("ErrTest")

var addr string
var privkey crypto.PrivKey

const fee = 1e6
const secretLen = 32

func init() {
	var err error
	conn, err = grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	c = types.NewGrpcserviceClient(conn)
}

func TestInitAccount(t *testing.T) {
	fmt.Println("TestInitAccount start")
	time.Sleep(5 * time.Second)
	defer fmt.Println("TestInitAccount end\n")

	addr, privkey = genaddress()
	label := strconv.Itoa(int(time.Now().UnixNano()))
	params := types.ReqWalletImportPrivKey{Privkey: common.ToHex(privkey.Bytes()), Label: label}
	_, err := c.ImportPrivKey(context.Background(), &params)
	if err != nil {
		fmt.Println(err)
		time.Sleep(time.Second)
		t.Error(err)
		return
	}
}

type TestOrg struct {
	OrgId   string `json:"orgId"`
	OrgName string `json:"orgName"`
}

var testKey string
var testValue []byte

func createKV() error {
	var err error
	testOrg := &TestOrg{OrgId: "33", OrgName: "org33"}
	testKey = testOrg.OrgId
	testValue, err = json.Marshal(testOrg)
	return err
}

func TestNormPut(t *testing.T) {
	fmt.Println("TestNormPut start")
	defer time.Sleep(10 * time.Second)
	defer fmt.Println("TestNormPut end\n")

	err := createKV()
	if err != nil {
		t.Error(err)
		return
	}
	vput := &types.NormAction_Nput{&types.NormPut{Key: testKey, Value: testValue}}
	transfer := &types.NormAction{Value: vput, Ty: types.NormActionPut}
	tx := &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(transfer), Fee: fee}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey)
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		t.Error(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		t.Error(errors.New(string(reply.GetMsg())))
		return
	}
}

func TestNormGet(t *testing.T) {
	fmt.Println("TestNormGet start")
	defer fmt.Println("TestNormGet end\n")

	var req types.Query
	req.Execer = []byte("norm")
	req.FuncName = "NormGet"
	req.Payload = []byte(testKey)
	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		t.Error(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		t.Error(errors.New(string(reply.GetMsg())))
		return
	}
	//the first two byte is not valid
	//QueryChain() need to change
	value := string(reply.Msg[2:])
	fmt.Println("GetValue =", value)

	var org TestOrg
	err = json.Unmarshal([]byte(value), &org)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println("GetOrg =", org)
}

func TestNormHas(t *testing.T) {
	fmt.Println("TestNormHas start")
	defer fmt.Println("TestNormHas end\n")

	has, err := Has([]byte(testKey))
	if err != nil {
		t.Error(err)
		return
	}
	if !has {
		t.Error(errors.New(testKey + " does exist"))
	}

	has, err = Has([]byte("nokey"))
	if err != nil {
		t.Error(err)
		return
	}
	if has {
		t.Error(errors.New(testKey + " does not exist"))
	}
}

func Has(key []byte) (bool, error) {
	var req types.Query
	req.Execer = []byte("norm")
	req.FuncName = "NormHas"
	req.Payload = []byte(key)
	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		return false, err
	}
	if !reply.IsOk {
		return false, errors.New(string(reply.GetMsg()))
	}
	value := string(reply.Msg[2:])
	fmt.Println("GetValue =", value)

	if value == "true" {
		return true, nil
	}
	return false, nil
}

func genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := account.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}
