package norm

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
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

var addrexec *account.Address

var addr string
var privGenesis, privkey crypto.PrivKey

var testKey = "norm-key"
var testValue = "norm-value"

const fee = 1e6
const secretLen = 32
const defaultAmount = 1e10

func init() {
	var err error
	conn, err = grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	c = types.NewGrpcserviceClient(conn)
	addrexec = account.ExecAddress("norm")
}

func TestInitAccount(t *testing.T) {
	fmt.Println("TestInitAccount start")
	defer fmt.Println("TestInitAccount end\n")

	privGenesis = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
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
	//need balance to pass tx check in mempool
	err = sendtoaddress(c, privGenesis, addr, defaultAmount)
	if err != nil {
		fmt.Println(err)
		time.Sleep(time.Second)
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
}

func TestNormPut(t *testing.T) {
	fmt.Println("TestNormPut start")
	defer time.Sleep(time.Second)
	defer fmt.Println("TestNormPut end\n")

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
	value := strings.TrimSpace(string(reply.Msg))
	fmt.Println("GetValue =", value)
	if value != testValue {
		t.Error(err)
		return
	}
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

func sendtoaddress(c types.GrpcserviceClient, priv crypto.PrivKey, to string, amount int64) error {
	//defer conn.Close()
	//fmt.Println("sign key privkey: ", common.ToHex(priv.Bytes()))
	if amount > 0 {
		v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
		transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: fee, To: to}
		tx.Nonce = r.Int63()
		tx.Sign(types.SECP256K1, priv)
		// Contact the server and print out its response.
		reply, err := c.SendTransaction(context.Background(), tx)
		if err != nil {
			fmt.Println("err", err)
			return err
		}
		if !reply.IsOk {
			fmt.Println("err = ", reply.GetMsg())
			return errors.New(string(reply.GetMsg()))
		}
		return nil
	} else {
		v := &types.CoinsAction_Withdraw{&types.CoinsWithdraw{Amount: -amount}}
		withdraw := &types.CoinsAction{Value: v, Ty: types.CoinsActionWithdraw}
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(withdraw), Fee: fee, To: to}
		tx.Nonce = r.Int63()
		tx.Sign(types.SECP256K1, priv)
		// Contact the server and print out its response.
		reply, err := c.SendTransaction(context.Background(), tx)
		if err != nil {
			fmt.Println("err", err)
			return err
		}
		if !reply.IsOk {
			fmt.Println("err = ", reply.GetMsg())
			return errors.New(string(reply.GetMsg()))
		}
		return nil
	}
}
