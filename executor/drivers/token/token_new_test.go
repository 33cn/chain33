package token

import (
	"context"
	//crand "crypto/rand"
	//"encoding/json"
	"errors"
	"fmt"
	"math/rand"

	"strconv"
	//"strings"
	"testing"
	"time"

	//"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

var (
	isMainNetTest bool = true
	isParaNetTest bool = false
)

var (
	mainNetgrpcAddr = "localhost:8802"
	ParaNetgrpcAddr = "localhost:8902"
	mainClient      types.GrpcserviceClient
	paraClient      types.GrpcserviceClient
	r               *rand.Rand

	ErrTest = errors.New("ErrTest")

	addrexec    string
	addr        string
	privkey     crypto.PrivKey
	privGenesis crypto.PrivKey
)

const (
	defaultAmount = 1e10
	fee           = 1e6
)

//for token
var (
	tokenName         = "NEW"
	tokenSym          = "NEW"
	tokenIntro        = "newtoken"
	tokenPrice  int64 = 0
	tokenAmount int64 = 1000 * 1e4 * 1e4
	execName          = "user.p.guodun.token"
	feeForToken int64 = 1e6
	transToAddr       = "1NYxhca2zVMzxFqMRJdMcZfrSFnqbqotKe" //exec addr for convenience
	transAmount int64 = 100 * 1e4 * 1e4
)

//测试过程：
//1. 初始化账户，导入有钱的私钥，创建一个新账户，往这个新账户打钱（用来签名和扣手续费）
//2. 产生precreate的一种token
//3. finish这个token
//4. 向一个地址转账token
//5. 可选：在平行链上进行query

func init() {
	fmt.Println("Init start")
	defer fmt.Println("Init end")

	if !isMainNetTest && !isParaNetTest {
		return
	}

	conn, err := grpc.Dial(mainNetgrpcAddr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	mainClient = types.NewGrpcserviceClient(conn)

	conn, err = grpc.Dial(ParaNetgrpcAddr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	paraClient = types.NewGrpcserviceClient(conn)

	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	addrexec = address.ExecAddress("user.p.guodun.token")

	privGenesis = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
}

func TestInitAccount(t *testing.T) {
	if !isMainNetTest {
		return
	}
	fmt.Println("TestInitAccount start")
	defer fmt.Println("TestInitAccount end")

	addr, privkey = genaddress()
	label := strconv.Itoa(int(types.Now().UnixNano()))
	params := types.ReqWalletImportPrivKey{Privkey: common.ToHex(privkey.Bytes()), Label: label}

	unlock := types.WalletUnLock{Passwd: "fzm123", Timeout: 0, WalletOrTicket: false}
	_, err := mainClient.UnLock(context.Background(), &unlock)
	if err != nil {
		fmt.Println(err)
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	_, err = mainClient.ImportPrivKey(context.Background(), &params)
	if err != nil {
		fmt.Println(err)
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	txhash, err := sendtoaddress(mainClient, privGenesis, addr, defaultAmount)

	if err != nil {
		t.Error(err)
		return
	}
	if !waitTx(txhash) {
		t.Error(ErrTest)
		return
	}
	time.Sleep(5 * time.Second)
	return
}

func TestPrecreate(t *testing.T) {
	if !isMainNetTest {
		return
	}
	fmt.Println("TestPrecreate start")
	defer fmt.Println("TestPrecreate end")

	v := &types.TokenPreCreate{
		Name:         tokenName,
		Symbol:       tokenSym,
		Introduction: tokenIntro,
		Total:        tokenAmount,
		Price:        tokenPrice,
		Owner:        addr,
	}
	precreate := &types.TokenAction{
		Ty:    types.TokenActionPreCreate,
		Value: &types.TokenAction_Tokenprecreate{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(execName),
		Payload: types.Encode(precreate),
		Fee:     feeForToken,
		Nonce:   r.Int63(),
		To:      address.ExecAddress(execName),
	}
	tx.Sign(types.SECP256K1, privkey)

	reply, err := mainClient.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Println("err", err)
		t.Error(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		t.Error(ErrTest)
		return
	}

	if !waitTx(tx.Hash()) {
		t.Error(ErrTest)
		return
	}
	time.Sleep(5 * time.Second)
	return
}

func TestFinish(t *testing.T) {
	if !isMainNetTest {
		return
	}
	fmt.Println("TestFinish start")
	defer fmt.Println("TestFinish end")

	v := &types.TokenFinishCreate{Symbol: tokenSym, Owner: addr}
	finish := &types.TokenAction{
		Ty:    types.TokenActionFinishCreate,
		Value: &types.TokenAction_Tokenfinishcreate{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(execName),
		Payload: types.Encode(finish),
		Fee:     feeForToken,
		Nonce:   r.Int63(),
		To:      address.ExecAddress(execName),
	}
	tx.Sign(types.SECP256K1, privkey)

	reply, err := mainClient.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Println("err", err)
		t.Error(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		t.Error(ErrTest)
		return
	}

	if !waitTx(tx.Hash()) {
		t.Error(ErrTest)
		return
	}
	time.Sleep(5 * time.Second)
	return
}

func TestTransferToken(t *testing.T) {
	if !isMainNetTest {
		return
	}
	fmt.Println("TestTransferToken start")
	defer fmt.Println("TestTransferToken end")

	v := &types.TokenAction_Transfer{Transfer: &types.CoinsTransfer{Cointoken: tokenSym, Amount: transAmount, Note: ""}}
	transfer := &types.TokenAction{Value: v, Ty: types.ActionTransfer}

	tx := &types.Transaction{Execer: []byte(execName), Payload: types.Encode(transfer), Fee: fee, To: transToAddr}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey)

	reply, err := mainClient.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Println("err", err)
		t.Error(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		t.Error(ErrTest)
		return
	}

	if !waitTx(tx.Hash()) {
		t.Error(ErrTest)
		return
	}
	return

}

func TestQueryAsset(t *testing.T) {
	if !isParaNetTest {
		return
	}
	fmt.Println("TestQueryAsset start")
	defer fmt.Println("TestQueryAsset end")

	var req types.Query
	req.Execer = []byte(execName)
	req.FuncName = "GetAccountTokenAssets"

	var reqAsset types.ReqAccountTokenAssets
	reqAsset.Address = addr
	reqAsset.Execer = execName

	req.Payload = types.Encode(&reqAsset)

	reply, err := paraClient.QueryChain(context.Background(), &req)
	if err != nil {
		fmt.Println(err)
		t.Error(err)
		return
	}
	if !reply.IsOk {
		fmt.Println("Query reply err")
		t.Error(ErrTest)
		return
	}
	var res types.ReplyAccountTokenAssets
	err = types.Decode(reply.Msg, &res)
	if err != nil {
		t.Error(err)
		return
	}
	for _, ta := range res.TokenAssets {
		//balanceResult := strconv.FormatFloat(float64(ta.Account.Balance)/float64(types.TokenPrecision), 'f', 4, 64)
		//frozenResult := strconv.FormatFloat(float64(ta.Account.Frozen)/float64(types.TokenPrecision), 'f', 4, 64)
		fmt.Println(ta.Symbol)
		fmt.Println(ta.Account.Addr)
		fmt.Println(ta.Account.Currency)
		fmt.Println(ta.Account.Balance)
		fmt.Println(ta.Account.Frozen)

	}

}

//***************************************************
//**************common actions for Test**************
//***************************************************
func sendtoaddress(c types.GrpcserviceClient, priv crypto.PrivKey, to string, amount int64) ([]byte, error) {
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: fee, To: to}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, priv)
	// Contact the server and print out its response.
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Println("err", err)
		return nil, err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return nil, errors.New(string(reply.GetMsg()))
	}
	return tx.Hash(), nil
}

func waitTx(hash []byte) bool {
	i := 0
	for {
		i++
		if i%100 == 0 {
			fmt.Println("wait transaction timeout")
			return false
		}

		var reqHash types.ReqHash
		reqHash.Hash = hash
		res, err := mainClient.QueryTransaction(context.Background(), &reqHash)
		if err != nil {
			time.Sleep(time.Second)
		}
		if res != nil {
			return true
		}
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
	addrto := address.PubKeyToAddress(privto.PubKey().Bytes())
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
