package executor

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
	tokenty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

var (
	isMainNetTest bool = false
	isParaNetTest bool = false
)

var (
	mainNetgrpcAddr = "localhost:8802"
	ParaNetgrpcAddr = "localhost:8902"
	mainClient      types.Chain33Client
	paraClient      types.Chain33Client
	r               *rand.Rand

	ErrTest = errors.New("ErrTest")

	addrexec     string
	addr         string
	privkey      crypto.PrivKey
	privGenesis  crypto.PrivKey
	privkeySuper crypto.PrivKey
)

const (
	//defaultAmount = 1e10
	fee = 1e6
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
	transToAddr       = "1NYxhca2zVMzxFqMRJdMcZfrSFnqbqotKe"
	transAmount int64 = 100 * 1e4 * 1e4
	walletPass        = "fzm123"
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
	mainClient = types.NewChain33Client(conn)

	conn, err = grpc.Dial(ParaNetgrpcAddr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	paraClient = types.NewChain33Client(conn)

	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	addrexec = address.ExecAddress("user.p.guodun.token")

	privGenesis = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	privkeySuper = getprivkey("4a92f3700920dc422c8ba993020d26b54711ef9b3d74deab7c3df055218ded42")
}

func TestInitAccount(t *testing.T) {
	if !isMainNetTest {
		return
	}
	fmt.Println("TestInitAccount start")
	defer fmt.Println("TestInitAccount end")

	//need update to fixed addr here
	//addr = ""
	//privkey = ""
	//addr, privkey = genaddress()
	label := strconv.Itoa(int(types.Now().UnixNano()))
	params := types.ReqWalletImportPrivkey{Privkey: common.ToHex(privkey.Bytes()), Label: label}

	unlock := types.WalletUnLock{Passwd: walletPass, Timeout: 0, WalletOrTicket: false}
	_, err := mainClient.UnLock(context.Background(), &unlock)
	if err != nil {
		fmt.Println(err)
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	_, err = mainClient.ImportPrivkey(context.Background(), &params)
	if err != nil && err != types.ErrPrivkeyExist {
		fmt.Println(err)
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	/*
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
	*/
}

func TestPrecreate(t *testing.T) {
	if !isMainNetTest {
		return
	}
	fmt.Println("TestPrecreate start")
	defer fmt.Println("TestPrecreate end")

	v := &tokenty.TokenPreCreate{
		Name:         tokenName,
		Symbol:       tokenSym,
		Introduction: tokenIntro,
		Total:        tokenAmount,
		Price:        tokenPrice,
		Owner:        addr,
	}
	precreate := &tokenty.TokenAction{
		Ty:    tokenty.TokenActionPreCreate,
		Value: &tokenty.TokenAction_TokenPreCreate{v},
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

}

func TestFinish(t *testing.T) {
	if !isMainNetTest {
		return
	}
	fmt.Println("TestFinish start")
	defer fmt.Println("TestFinish end")

	v := &tokenty.TokenFinishCreate{Symbol: tokenSym, Owner: addr}
	finish := &tokenty.TokenAction{
		Ty:    tokenty.TokenActionFinishCreate,
		Value: &tokenty.TokenAction_TokenFinishCreate{v},
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

}

func TestTransferToken(t *testing.T) {
	if !isMainNetTest {
		return
	}
	fmt.Println("TestTransferToken start")
	defer fmt.Println("TestTransferToken end")

	v := &tokenty.TokenAction_Transfer{Transfer: &types.AssetsTransfer{Cointoken: tokenSym, Amount: transAmount, Note: "", To: transToAddr}}
	transfer := &tokenty.TokenAction{Value: v, Ty: tokenty.ActionTransfer}

	tx := &types.Transaction{Execer: []byte(execName), Payload: types.Encode(transfer), Fee: fee, To: addrexec}
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

}

func TestQueryAsset(t *testing.T) {
	if !isParaNetTest {
		return
	}
	fmt.Println("TestQueryAsset start")
	defer fmt.Println("TestQueryAsset end")

	var req types.ChainExecutor
	req.Driver = execName
	req.FuncName = "GetAccountTokenAssets"

	var reqAsset tokenty.ReqAccountTokenAssets
	reqAsset.Address = addr
	reqAsset.Execer = execName

	req.Param = types.Encode(&reqAsset)

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
	var res tokenty.ReplyAccountTokenAssets
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
func sendtoaddress(c types.Chain33Client, priv crypto.PrivKey, to string, amount int64) ([]byte, error) {
	v := &cty.CoinsAction_Transfer{&types.AssetsTransfer{Amount: amount}}
	transfer := &cty.CoinsAction{Value: v, Ty: cty.CoinsActionTransfer}
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
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
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
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
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
