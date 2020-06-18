// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package broadcast

import (
	"encoding/hex"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func Test_sendBlock(t *testing.T) {

	proto := newTestProtocol()
	_, ok := proto.handleSend(&types.P2PBlock{Block: &types.Block{}}, testPid, testAddr)
	assert.True(t, ok)
	_, ok = proto.handleSend(&types.P2PBlock{Block: &types.Block{}}, testPid, testAddr)
	assert.False(t, ok)
	proto.p2pCfg.MinLtBlockSize = 0
	data, ok := proto.handleSend(&types.P2PBlock{Block: testBlock}, "newpid", testAddr)
	ltBlock := data.Value.(*types.BroadCastData_LtBlock).LtBlock
	assert.True(t, ok)
	assert.True(t, ltBlock.MinerTx == minerTx)
	assert.Equal(t, 3, len(ltBlock.STxHashes))
}

func Test_recvBlock(t *testing.T) {
	q := queue.New("test")
	go q.Start()
	defer q.Close()
	proto := newTestProtocolWithQueue(q)
	recvData := &types.BroadCastData{}
	block := &types.BroadCastData_Block{Block: &types.P2PBlock{}}
	recvData.Value = block
	err := proto.handleReceive(recvData, testPid, testAddr)
	assert.Equal(t, types.ErrInvalidParam, err)
	block.Block = &types.P2PBlock{Block: testBlock}

	newCli := q.Client()
	newCli.Sub("blockchain")
	err = proto.handleReceive(recvData, testPid, testAddr)
	assert.Nil(t, err)
	err = proto.handleReceive(recvData, testPid, testAddr)
	assert.Nil(t, err)
	blockHash := hex.EncodeToString(testBlock.Hash(proto.ChainCfg))
	assert.True(t, proto.blockSendFilter.Contains(blockHash))
	assert.True(t, proto.blockFilter.Contains(blockHash))

	msg := <-newCli.Recv()
	assert.Equal(t, types.EventBroadcastAddBlock, int(msg.Ty))
	blc, ok := msg.Data.(*types.BlockPid)
	assert.True(t, ok)
	assert.Equal(t, testPid.Pretty(), blc.Pid)
	assert.Equal(t, blockHash, hex.EncodeToString(blc.Block.Hash(proto.ChainCfg)))
}

func startHandleMempool(q queue.Queue, txList *[]*types.Transaction) chan struct{} {
	client := q.Client()
	client.Sub("mempool")
	done := make(chan struct{})
	go func() {
		close(done)
		for msg := range client.Recv() {
			msg.Reply(client.NewMessage("p2p", types.EventTxListByHash, &types.ReplyTxList{Txs: *txList}))
		}
	}()
	return done
}

func handleTestMsgReply(cli queue.Client, ty int64, reply interface{}, once bool) chan struct{} {
	done := make(chan struct{})
	go func() {
		close(done)
		for msg := range cli.Recv() {
			if msg.Ty == ty {
				msg.Reply(cli.NewMessage("p2p", ty, reply))
				if once {
					return
				}
			}
		}
	}()
	return done
}

func Test_recvLtBlock(t *testing.T) {

	q := queue.New("test")
	proto := newTestProtocolWithQueue(q)
	proto.p2pCfg.MinLtBlockSize = 0
	defer q.Close()
	memTxList := []*types.Transaction{tx, tx1, tx2}
	done := startHandleMempool(q, &memTxList)
	<-done
	block := &types.Block{TxHash: []byte("test"), Txs: txList, Height: 10}
	temppid, _ := peer.Decode("16Uiu2HAkudcLD1rRPmLL6PYqT3frNFUhTt764yWx1pfrVW8eWUeY")
	sendData, _ := proto.handleSend(&types.P2PBlock{Block: block}, temppid, "tempaddr")

	// block tx hash校验不过
	err := proto.handleReceive(sendData, testPid, testAddr)
	assert.Nil(t, err)
	blockHash := hex.EncodeToString(block.Hash(proto.ChainCfg))
	assert.True(t, proto.blockSendFilter.Contains(blockHash))
	//缺失超过1/3的交易数量
	memTxList = []*types.Transaction{tx, nil, nil}
	err = proto.handleReceive(sendData, testPid, testAddr)
	assert.Nil(t, err)
	//交易组正确流程测试
	txGroup, _ := types.CreateTxGroup([]*types.Transaction{tx1, tx2}, proto.ChainCfg.GetMinTxFeeRate())
	gtx := txGroup.Tx()
	memTxList = []*types.Transaction{tx, gtx}
	block.TxHash = merkle.CalcMerkleRoot(proto.ChainCfg, block.GetHeight(), block.Txs)
	newCli := q.Client()
	newCli.Sub("blockchain")
	sendData, _ = proto.handleSend(&types.P2PBlock{Block: block}, testPid, testAddr)
	err = proto.handleReceive(sendData, testPid, testAddr)
	assert.Nil(t, err)
	msg := <-newCli.Recv()
	assert.Equal(t, types.EventBroadcastAddBlock, int(msg.Ty))
	blc, ok := msg.Data.(*types.BlockPid)
	assert.True(t, ok)
	assert.Equal(t, testPid.Pretty(), blc.Pid)
	assert.Equal(t, block.Hash(proto.ChainCfg), blc.Block.Hash(proto.ChainCfg))
}
