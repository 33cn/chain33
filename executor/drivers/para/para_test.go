package para

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

var r *rand.Rand
var defaultkey string = "parakey"
var defaultvalue string = "default"

//var defaultvalue []byte = []byte{'d', 'e', 'f', 'a', 'u', 'l', 't'}
var c types.GrpcserviceClient
var conn *grpc.ClientConn
var (
	addrexec    string
	addr        string
	privGenesis crypto.PrivKey
)
var doQuery bool = false //query on para chain
var doPut bool = false   //put on main chain

var txNum int = 10

func init() {
	if doQuery != true && doPut != true {
		return
	}
	fmt.Println("para test init")
	var err error
	conn, err = grpc.Dial("localhost:8902", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	c = types.NewGrpcserviceClient(conn)
	addrexec = address.ExecAddress("user.para")
	privGenesis = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
}

func TestParaPut(t *testing.T) {
	if doPut {
		err := ParaPut()
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func TestParaQuery(t *testing.T) {
	if doQuery {
		var req types.Query
		req.Execer = []byte("user.para")
		req.FuncName = "ParaGet"
		for i := 0; i < txNum; i++ {
			req.Payload = []byte(defaultkey + strconv.Itoa(i+1))

			reply, err := c.QueryChain(context.Background(), &req)
			if err != nil {
				fmt.Println(err)
				return
			}
			if !reply.IsOk {
				fmt.Println("Query reply err")
				return
			}
			//the first two byte is not valid
			//QueryChain() need to change
			value := string(reply.Msg[2:])
			fmt.Println("GetValue =", value)
			time.Sleep(time.Second)
		}
	}
}

func ParaPut() error {
	for i := 0; i < txNum; i++ {
		parakey := defaultkey + strconv.Itoa(i+1)
		paraval := []byte(defaultvalue + strconv.Itoa(i+1))
		input := &types.ParaAction_Put{Put: &types.ParaPut{Key: parakey, Value: paraval}}
		fmt.Println(input)
		transfer := &types.ParaAction{Value: input, Ty: types.ParaActionPut}
		tx := &types.Transaction{Execer: []byte("user.para"), Payload: types.Encode(transfer), Fee: 0, To: addrexec}
		tx.Nonce = r.Int63()

		//tx.SetExpire(time.Second * 3600)
		tx.Sign(types.SECP256K1, privGenesis)
		reply, err := c.SendTransaction(context.Background(), tx)
		if err != nil {
			return err
		}
		if !reply.IsOk {
			fmt.Println("err = ", reply.GetMsg())
			return errors.New(string(reply.GetMsg()))
		}
		time.Sleep(5 * time.Second)
	}

	return nil
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
