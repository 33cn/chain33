package solo

import (
	"strconv"
	"testing"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	transactions []*types.Transaction
	txSize       int = 10000
	endLoop      int = 10
)

// 执行： go test -cover
// 多次执行，需要删除之前的数据库文件，不然交易都是重复的
func TestSolo(t *testing.T) {
	q := queue.New("channel")

	chain := blockchain.New()
	chain.SetQueueClient(q.Client())

	con := NewSolo()
	con.SetQueueClient(q.Client())

	go sendReplyList(q)

	log.Info("start")
	q.Start()

}

// 向共识发送交易列表
func sendReplyList(q queue.Queue) {
	client := q.Client()
	client.Sub("mempool")
	var accountNum int
	for msg := range client.Recv() {
		if msg.Ty == types.EventTxList {
			accountNum++
			if accountNum < endLoop/2 {
				// 无交易
				msg.Reply(client.NewMessage("consensus", types.EventTxListReply,
					&types.ReplyTxList{}))
			} else {
				// 有交易
				createReplyList("test" + strconv.Itoa(accountNum))
				msg.Reply(client.NewMessage("consensus", types.EventTxListReply,
					&types.ReplyTxList{transactions}))
			}
			if accountNum == endLoop+1 {
				log.Info("Test finished!!")
				q.Close()
				break
			}

		}

	}
}

// 准备交易列表
func createReplyList(account string) {
	var result []*types.Transaction
	for j := 0; j < txSize; j++ {
		var b types.Transaction
		if j > 1000 && j%1000 == 0 {
			// 重复交易
			b = types.Transaction{[]byte("duplicate"), []byte("duplicate"), []byte("duplicate")}
		} else {
			b = types.Transaction{[]byte(account + "This is a payload" + strconv.Itoa(j)), []byte(account + "This is a account" + strconv.Itoa(j)), []byte(account + "This is a signature" + strconv.Itoa(j))}
		}

		result = append(result, &b)
	}
	transactions = result
}
