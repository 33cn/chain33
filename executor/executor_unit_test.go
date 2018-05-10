package executor

import (
	//"errors"
	"math/rand"
	"testing"
	"time"

	"fmt"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	//"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var randomU *rand.Rand

var (
	addr1 string
)

func init() {
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	randomU = rand.New(rand.NewSource(time.Now().UnixNano()))
	log.SetLogLevel("error")
}

func initUnitEnv() (queue.Queue, *Executor) {
	var q = queue.New("channel")
	cfg := config.InitCfg("../cmd/chain33/chain33.test.toml")
	exec := New(cfg.Exec)
	exec.SetQueueClient(q.Client())
	//types.SetMinFee(0)
	return q, exec
}

func genExecTxListMsg(client queue.Client, block *types.Block) queue.Message {
	list := &types.ExecTxList{zeroHash[:], block.Txs, block.BlockTime, block.Height}
	msg := client.NewMessage("execs", types.EventExecTxList, list)
	return msg
}

func genExecCheckTxMsg(client queue.Client, block *types.Block) queue.Message {
	list := &types.ExecTxList{zeroHash[:], block.Txs, block.BlockTime, block.Height}
	msg := client.NewMessage("execs", types.EventCheckTx, list)
	return msg
}

func genEventAddBlockMsg(client queue.Client) queue.Message {
	blockDetail := constructionBlockDetail(zeroHash[:], 1, 1)
	msg := client.NewMessage("execs", types.EventAddBlock, blockDetail)
	return msg
}

func genEventDelBlockMsg(client queue.Client) queue.Message {
	blockDetail := constructionBlockDetail(zeroHash[:], 1, 1)
	msg := client.NewMessage("execs", types.EventDelBlock, blockDetail)
	return msg
}

func constructionBlockDetail(parentHash []byte, height int64, txcount int) *types.BlockDetail {
	var blockdetail types.BlockDetail
	var block types.Block

	blockdetail.Receipts = make([]*types.ReceiptData, txcount)

	block.BlockTime = height
	block.Height = height
	block.Version = 100
	block.StateHash = common.Hash{}.Bytes()
	block.ParentHash = parentHash
	txs := genTxs(int64(txcount))
	block.Txs = txs
	block.TxHash = merkle.CalcMerkleRoot(block.Txs)

	for j := 0; j < txcount; j++ {
		ReceiptData := types.ReceiptData{Ty: 2}
		blockdetail.Receipts[j] = &ReceiptData
	}
	blockdetail.Block = &block
	return &blockdetail
}

func genEventBlockChainQueryMsg(client queue.Client, param []byte) queue.Message {
	blockChainQue := &types.BlockChainQuery{"coins", "GetTxsByAddr", zeroHash[:], param}
	msg := client.NewMessage("execs", types.EventBlockChainQuery, blockChainQue)
	return msg
}

func TestQueueClient(t *testing.T) {
	q, _ := initUnitEnv()
	storeProcess(q)
	blockchainProcess(q)

	var txNum int64 = 5
	var msgs []queue.Message
	var msg queue.Message

	// 1、测试 EventExecTxList 消息
	block := createBlock(txNum)
	addr1 = account.PubKeyToAddress(block.Txs[0].GetSignature().GetPubkey()).String() //将获取随机生成交易地址
	msg = genExecTxListMsg(q.Client(), block)                                         //生成消息
	msgs = append(msgs, msg)

	// 2、测试 EventAddBlock 消息
	msg = genEventAddBlockMsg(q.Client()) //生成消息
	msgs = append(msgs, msg)

	// 3、测试 EventDelBlock 消息
	msg = genEventDelBlockMsg(q.Client()) //生成消息
	msgs = append(msgs, msg)

	// 4、测试 EventCheckTx 消息
	msg = genExecCheckTxMsg(q.Client(), block) //生成消息
	msgs = append(msgs, msg)

	// 5、测试 EventBlockChainQuery 消息
	var reqAddr types.ReqAddr
	reqAddr.Addr, _ = genaddress()
	reqAddr.Flag = 0
	reqAddr.Count = 10
	reqAddr.Direction = 0
	reqAddr.Height = -1
	reqAddr.Index = 0

	msg = genEventBlockChainQueryMsg(q.Client(), types.Encode(&reqAddr)) //生成消息
	msgs = append(msgs, msg)

	go func() {
		for _, msga := range msgs {
			q.Client().Send(msga, true)
			_, err := q.Client().Wait(msga)
			if err == nil || err == types.ErrNotFound {
				t.Logf("%v,%v", msga, err)
			} else {
				t.Error(err)
			}
		}
		q.Close()
	}()
	q.Start()
}

func storeProcess(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub("store")
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventStoreGet:
				datas := msg.GetData().(*types.StoreGet)
				fmt.Println("EventStoreGet data = %v", datas.StateHash)
				values := make([][]byte, 2)
				account := &types.Account{
					Balance: 1000 * 1e8,
					Addr:    addr1,
				}
				value := types.Encode(account)
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
				header := &types.Header{StateHash: []byte("111111111111111111111")}
				msg.Reply(client.NewMessage("", types.EventHeader, header))
			case types.EventLocalGet:
				fmt.Println("EventLocalGet rsp")
				msg.Reply(client.NewMessage("", types.EventLocalReplyValue, &types.LocalReplyValue{}))
			case types.EventLocalList:
				fmt.Println("EventLocalList rsp")
				//value := []byte{1,2,3,4,5,6}
				//values := make([][]byte,2)
				//values = append(values[:0],value)
				var values [][]byte
				msg.Reply(client.NewMessage("", types.EventLocalReplyValue, &types.LocalReplyValue{Values: values}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}
