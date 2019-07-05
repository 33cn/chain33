package p2p

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

type versionData struct {
	rawData interface{}
	version int32
}

func Test_processP2P(t *testing.T) {

	q := queue.New("channel")
	go q.Start()
	p2p := newP2p(12345, "testProcessP2p", q)
	defer freeP2p(p2p)
	defer q.Close()
	node := p2p.node
	client := p2p.client
	pid := "testPid"
	sendChan := make(chan interface{}, 1)
	recvChan := make(chan *types.BroadCastData, 1)
	testDone := make(chan struct{})

	payload := []byte("testpayload")
	minerTx := &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 14600, Expire: 200}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 4600, Expire: 2}
	tx1 := &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 460000000, Expire: 0}
	tx2 := &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 100, Expire: 1}
	txGroup, _ := types.CreateTxGroup([]*types.Transaction{tx1, tx2})
	gtx := txGroup.Tx()
	txList := append([]*types.Transaction{}, minerTx, tx, tx1, tx2)
	memTxList := append([]*types.Transaction{}, tx, gtx)

	block := &types.Block{
		TxHash: []byte("123"),
		Height: 10,
		Txs:    txList,
	}
	txHash := hex.EncodeToString(tx.Hash())
	blockHash := hex.EncodeToString(block.Hash())
	rootHash := merkle.CalcMerkleRoot(txList)

	//mempool handler
	go func() {
		client := q.Client()
		client.Sub("mempool")
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventTxListByHash:
				query := msg.Data.(*types.ReqTxHashList)
				var txs []*types.Transaction
				if !query.IsShortHash {
					txs = memTxList[:1]
				} else {
					txs = memTxList
				}
				msg.Reply(client.NewMessage("p2p", types.EventTxListByHash, &types.ReplyTxList{Txs: txs}))
			}
		}
	}()

	//测试发送
	go func() {
		for data := range sendChan {
			verData, ok := data.(*versionData)
			assert.True(t, ok)
			sendData, doSend := node.processSendP2P(verData.rawData, verData.version, "testIP:port")
			txHashFilter.regRData.Remove(txHash)
			blockHashFilter.regRData.Remove(blockHash)
			assert.True(t, doSend, "sendData:", verData.rawData)
			recvChan <- sendData
		}
	}()
	//测试接收
	go func() {
		for data := range recvChan {
			txHashFilter.regRData.Remove(txHash)
			blockHashFilter.regRData.Remove(blockHash)
			handled := node.processRecvP2P(data, pid, node.pubToPeer, "testIP:port")
			assert.True(t, handled)
		}
	}()

	go func() {
		p2pChan := node.pubsub.Sub("tx")
		for data := range p2pChan {
			if p2pTx, ok := data.(*types.P2PTx); ok {
				sendChan <- &versionData{rawData: p2pTx, version: lightBroadCastVersion}
			}
		}
	}()

	//data test
	go func() {
		subChan := node.pubsub.Sub(pid)
		//normal
		sendChan <- &versionData{rawData: &types.P2PTx{Tx: tx, Route: &types.P2PRoute{}}, version: lightBroadCastVersion - 1}
		assert.Nil(t, client.Send(client.NewMessage("p2p", types.EventTxBroadcast, tx), false))
		sendChan <- &versionData{rawData: &types.P2PBlock{Block: block}, version: lightBroadCastVersion - 1}
		//light broadcast
		txHashFilter.Add(hex.EncodeToString(tx1.Hash()), &types.P2PRoute{TTL: DefaultLtTxBroadCastTTL})
		_ = client.Send(client.NewMessage("p2p", types.EventTxBroadcast, tx1), false)
		sendChan <- &versionData{rawData: &types.P2PTx{Tx: tx, Route: &types.P2PRoute{TTL: DefaultLtTxBroadCastTTL}}, version: lightBroadCastVersion}
		<-subChan //query tx
		sendChan <- &versionData{rawData: &types.P2PBlock{Block: block}, version: lightBroadCastVersion}
		<-subChan //query block
		for !ltBlockCache.contains(blockHash) {
		}
		ltBlock := ltBlockCache.get(blockHash).(*types.Block)
		assert.True(t, bytes.Equal(rootHash, merkle.CalcMerkleRoot(ltBlock.Txs)))

		//query tx
		sendChan <- &versionData{rawData: &types.P2PQueryData{Value: &types.P2PQueryData_TxReq{TxReq: &types.P2PTxReq{TxHash: tx.Hash()}}}}
		_, ok := (<-subChan).(*types.P2PTx)
		assert.True(t, ok)
		sendChan <- &versionData{rawData: &types.P2PQueryData{Value: &types.P2PQueryData_BlockTxReq{BlockTxReq: &types.P2PBlockTxReq{
			BlockHash: blockHash,
			TxIndices: []int32{1, 2},
		}}}}
		rep, ok := (<-subChan).(*types.P2PBlockTxReply)
		assert.True(t, ok)
		assert.Equal(t, 2, int(rep.TxIndices[1]))
		sendChan <- &versionData{rawData: &types.P2PQueryData{Value: &types.P2PQueryData_BlockTxReq{BlockTxReq: &types.P2PBlockTxReq{
			BlockHash: blockHash,
			TxIndices: nil,
		}}}}
		rep, ok = (<-subChan).(*types.P2PBlockTxReply)
		assert.True(t, ok)
		assert.Nil(t, rep.TxIndices)

		//query reply
		sendChan <- &versionData{rawData: &types.P2PBlockTxReply{
			BlockHash: blockHash,
			TxIndices: []int32{1},
			Txs:       txList[1:2],
		}}
		<-subChan
		assert.True(t, ltBlockCache.contains(blockHash))

		ltBlock.TxHash = rootHash
		sendChan <- &versionData{rawData: &types.P2PBlockTxReply{
			BlockHash: blockHash,
			Txs:       txList[0:],
		}}
		for ltBlockCache.contains(blockHash) {
		}
		//max ttl
		node.nodeInfo.cfg.MaxTTL = 0
		_, doSend := node.processSendP2P(&types.P2PTx{Tx: tx, Route: &types.P2PRoute{TTL: 1}}, lightBroadCastVersion, "testIP:port")
		assert.False(t, doSend)
		close(testDone)
	}()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-testDone:
			return
		case <-ticker.C:
			t.Error("TestP2PProcessTimeout")
			return
		}
	}
}
