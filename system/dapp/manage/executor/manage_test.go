package executor_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/limits"
	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/executor"
	"github.com/33cn/chain33/queue"
	_ "github.com/33cn/chain33/system"
	pty "github.com/33cn/chain33/system/dapp/manage/types"
	"github.com/33cn/chain33/types"
	"google.golang.org/grpc"
)

var (
	isManageTest       = false
	execNameMa         = "user.p.guodun.manage"
	feeForToken  int64 = 1e6
	ErrTest            = errors.New("ErrTest")

	addr1           string
	mainNetgrpcAddr = "localhost:8802"
	ParaNetgrpcAddr = "localhost:8902"
	mainClient      types.Chain33Client
	paraClient      types.Chain33Client

	random       *rand.Rand
	zeroHash     [32]byte
	cfg          *types.Config
	addr         string
	genkey       crypto.PrivKey
	privGenesis  crypto.PrivKey
	privkeySuper crypto.PrivKey
)

func init() {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	types.Init("local", nil)
	err := limits.SetLimits()
	if err != nil {
		panic(err)
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

	random = rand.New(rand.NewSource(types.Now().UnixNano()))
	genkey = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	log.SetLogLevel("error")
	privGenesis = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	privkeySuper = getprivkey("4a92f3700920dc422c8ba993020d26b54711ef9b3d74deab7c3df055218ded42")
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

func initUnitEnv() (queue.Queue, *executor.Executor) {
	var q = queue.New("channel")
	cfg, sub := types.InitCfg("../../../../cmd/chain33/chain33.test.toml")
	exec := executor.New(cfg.Exec, sub.Exec)
	exec.SetQueueClient(q.Client())
	return q, exec
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

func createTxEx(priv crypto.PrivKey, to string, amount int64, ty int32, execer string) *types.Transaction {
	var tx *types.Transaction
	switch execer {
	case "manage":
		v := &types.ModifyConfig{}
		modify := &pty.ManageAction{
			Ty:    ty,
			Value: &pty.ManageAction_Modify{Modify: v},
		}
		tx = &types.Transaction{Execer: []byte("manage"), Payload: types.Encode(modify)}
	default:
		return nil
	}
	tx.Nonce = random.Int63()
	//tx.To = address.ExecAddress(execer).String()
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func genTxsEx(n int64, ty int32, execer string) (txs []*types.Transaction) {
	_, priv := genaddress()
	to, _ := genaddress()
	for i := 0; i < int(n); i++ {
		txs = append(txs, createTxEx(priv, to, types.Coin*(n+1), ty, execer))
	}
	return txs
}

func createBlockEx(n int64, ty int32, execer string) *types.Block {
	newblock := &types.Block{}
	newblock.Height = -1
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = genTxsEx(n, ty, execer)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

func constructionBlockDetail(block *types.Block, height int64, txcount int) *types.BlockDetail {
	var blockdetail types.BlockDetail

	blockdetail.Receipts = make([]*types.ReceiptData, txcount)
	for j := 0; j < txcount; j++ {
		ReceiptData := types.ReceiptData{Ty: 2}
		blockdetail.Receipts[j] = &ReceiptData
	}
	blockdetail.Block = block
	return &blockdetail
}

func genExecTxListMsg(client queue.Client, block *types.Block) queue.Message {
	list := &types.ExecTxList{zeroHash[:], block.Txs, block.BlockTime, block.Height, 0, false}
	msg := client.NewMessage("execs", types.EventExecTxList, list)
	return msg
}

func genExecCheckTxMsg(client queue.Client, block *types.Block) queue.Message {
	list := &types.ExecTxList{zeroHash[:], block.Txs, block.BlockTime, block.Height, 0, false}
	msg := client.NewMessage("execs", types.EventCheckTx, list)
	return msg
}

func genEventAddBlockMsg(client queue.Client, block *types.Block) queue.Message {
	blockDetail := constructionBlockDetail(block, 1, 1)
	msg := client.NewMessage("execs", types.EventAddBlock, blockDetail)
	return msg
}

func genEventDelBlockMsg(client queue.Client, block *types.Block) queue.Message {
	blockDetail := constructionBlockDetail(block, 1, 1)
	msg := client.NewMessage("execs", types.EventDelBlock, blockDetail)
	return msg
}

//"coins", "GetTxsByAddr",
func genEventBlockChainQueryMsg(client queue.Client, param []byte, strDriver string, strFunName string) queue.Message {
	blockChainQue := &types.ChainExecutor{strDriver, strFunName, zeroHash[:], param, nil}
	msg := client.NewMessage("execs", types.EventBlockChainQuery, blockChainQue)
	return msg
}

func storeProcess(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub("store")
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventStoreGet:
				datas := msg.GetData().(*types.StoreGet)
				//fmt.Println("EventStoreGet Keys[0] = %s", string(datas.Keys[0]))

				var value []byte
				if strings.Contains(string(datas.Keys[0]), "this type ***") { //这种type结构体走该分支
					item := &types.ConfigItem{
						Key:  "111111111111111",
						Addr: "2222222222222",
						Value: &types.ConfigItem_Arr{
							Arr: &types.ArrayConfig{}},
					}
					value = types.Encode(item)
				} else {
					account := &types.Account{
						Balance: 1000 * 1e8,
						Addr:    addr1,
					}
					value = types.Encode(account)
				}
				values := make([][]byte, 2)
				values = append(values[:0], value)
				msg.Reply(client.NewMessage("", types.EventStoreGetReply, &types.StoreReplyValue{values}))
			case types.EventStoreGetTotalCoins:
				req := msg.GetData().(*types.IterateRangeByStateHash)
				resp := &types.ReplyGetTotalCoins{}
				resp.Count = req.Count
				msg.Reply(client.NewMessage("", types.EventGetTotalCoinsReply, resp))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func blockchainProcess(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub("blockchain")
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetLastHeader:
				header := &types.Header{StateHash: []byte("111111111111111111111"), Height: 1}
				msg.Reply(client.NewMessage("", types.EventHeader, header))
			case types.EventLocalGet:
				//fmt.Println("EventLocalGet rsp")
				msg.Reply(client.NewMessage("", types.EventLocalReplyValue, &types.LocalReplyValue{}))
			case types.EventLocalList:
				var values [][]byte
				msg.Reply(client.NewMessage("", types.EventLocalReplyValue, &types.LocalReplyValue{Values: values}))

			case types.EventLocalPrefixCount:
				msg.Reply(client.NewMessage("", types.EventLocalReplyValue, &types.Int64{Data: 0}))

			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func createBlockChainQueryRq(execer string, funcName string) proto.Message {
	switch execer {
	case "manage":
		{
			if funcName == "GetConfigItem" {
				in := &types.ReqString{Data: "this type ***"}
				return in
			}
		}
	default:
		return nil
	}

	return nil
}

func TestQueueClient(t *testing.T) {
	q, _ := initUnitEnv()
	storeProcess(q)
	blockchainProcess(q)

	var txNum int64 = 1
	var msgs []queue.Message
	var msg queue.Message
	var block *types.Block

	execTy := [][]string{
		{"manage", strconv.Itoa(pty.ManageActionModifyConfig)},
	}

	for _, str := range execTy {
		ty, _ := strconv.Atoi(str[1])
		block = createBlockEx(txNum, int32(ty), str[0])
		addr1 = block.Txs[0].From() //将获取随机生成交易地址

		// 1、测试 EventExecTxList 消息
		msg = genExecTxListMsg(q.Client(), block)
		msgs = append(msgs, msg)

		//2、测试 EventAddBlock 消息
		msg = genEventAddBlockMsg(q.Client(), block)
		msgs = append(msgs, msg)

		// 3、测试 EventDelBlock 消息
		msg = genEventDelBlockMsg(q.Client(), block)
		msgs = append(msgs, msg)

		// 4、测试 EventCheckTx 消息
		msg = genExecCheckTxMsg(q.Client(), block)
		msgs = append(msgs, msg)
	}

	// 5、测试 EventBlockChainQuery 消息
	var reqAddr types.ReqAddr
	reqAddr.Addr, _ = genaddress()
	reqAddr.Flag = 0
	reqAddr.Count = 10
	reqAddr.Direction = 0
	reqAddr.Height = -1
	reqAddr.Index = 0

	execFunName := [][]string{
		{"manage", "GetConfigItem"},
	}

	for _, str := range execFunName {
		t := createBlockChainQueryRq(str[0], str[1])
		if nil == t {
			continue
		}
		msg = genEventBlockChainQueryMsg(q.Client(), types.Encode(t), str[0], str[1]) //生成消息
		msgs = append(msgs, msg)
	}

	go func() {
		for _, msga := range msgs {
			q.Client().Send(msga, true)
			_, err := q.Client().Wait(msga)
			if err == nil || err == types.ErrNotFound || err == types.ErrEmpty {
				t.Logf("%v,%v", msga, err)
			} else {
				t.Error(err)
			}
		}
		q.Close()
	}()
	q.Start()
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

func TestManageForTokenBlackList(t *testing.T) {
	if !isManageTest {
		return
	}
	fmt.Println("TestManageForTokenBlackList start")
	defer fmt.Println("TestManageForTokenBlackList end")

	v := &types.ModifyConfig{Key: "token-blacklist", Op: "add", Value: "GDT", Addr: ""}
	modify := &pty.ManageAction{
		Ty:    pty.ManageActionModifyConfig,
		Value: &pty.ManageAction_Modify{Modify: v},
	}
	tx := &types.Transaction{
		Execer:  []byte(execNameMa),
		Payload: types.Encode(modify),
		Fee:     feeForToken,
		Nonce:   random.Int63(),
		To:      address.ExecAddress(execNameMa),
	}

	tx.Sign(types.SECP256K1, privkeySuper)

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

func TestManageForTokenFinisher(t *testing.T) {
	if !isManageTest {
		return
	}
	fmt.Println("TestManageForTokenFinisher start")
	defer fmt.Println("TestManageForTokenFinisher end")

	v := &types.ModifyConfig{Key: "token-finisher", Op: "add", Value: addr, Addr: ""}
	modify := &pty.ManageAction{
		Ty:    pty.ManageActionModifyConfig,
		Value: &pty.ManageAction_Modify{Modify: v},
	}
	tx := &types.Transaction{
		Execer:  []byte(execNameMa),
		Payload: types.Encode(modify),
		Fee:     feeForToken,
		Nonce:   random.Int63(),
		To:      address.ExecAddress(execNameMa),
	}

	tx.Sign(types.SECP256K1, privkeySuper)

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
