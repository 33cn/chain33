package blacklist

import (
	"context"
	"errors"
	"fmt"
	mrand "math/rand"
	"time"
	//"crypto/rand"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	. "gitlab.33.cn/chain33/chain33/executor/drivers/blacklist/types"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
	"os"
)

var conn *grpc.ClientConn
var client types.GrpcserviceClient
var rand *mrand.Rand

func createConn(ip string) {
	var err error
	url := ip + ":8802"
	fmt.Println("grpc url:", url)
	blog.Info("grpc url:", url)
	conn, err = grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	client = types.NewGrpcserviceClient(conn)
	rand = mrand.New(mrand.NewSource(time.Now().UnixNano()))
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
func sendTransaction(privKey string, tr *Transaction) {
	createConn(ConnIp)
	action := &BlackAction{Value: &BlackAction_Tr{tr}, FuncName: FuncName_Transfer}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: Fee}
	tx.To = "user.blacklist"
	tx.Nonce = rand.Int63()
	tx.Sign(types.SECP256K1, getprivkey(privKey))

	reply, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
}
func modifyPwd(privKey string, tr *Transaction) {
	createConn(ConnIp)
	action := &BlackAction{Value: &BlackAction_Tr{tr}, FuncName: FuncName_Transfer}
	tx := &types.Transaction{Execer: []byte("user.blacklist"), Payload: types.Encode(action), Fee: Fee}
	tx.To = "user.blacklist"
	tx.Nonce = rand.Int63()
	tx.Sign(types.SECP256K1, getprivkey(privKey))

	reply, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if !reply.IsOk {
		fmt.Fprintln(os.Stderr, errors.New(string(reply.GetMsg())))
		return
	}
}
func generateAddr() string {
	rand := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	addr := fmt.Sprintf("%06v", rand.Uint64())
	fmt.Println(addr)
	return addr
}
func generateTxId() string {
	rand := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	addr := fmt.Sprintf("%06v", rand.Int31n(1000000))
	fmt.Println(addr)
	return addr
}
