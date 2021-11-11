// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package broadcast

import (
	"sync/atomic"
	"testing"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	tx      = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 4600, Expire: 2}
	tx1     = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 460000000, Expire: 0}
	tx2     = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 100, Expire: 1}
	minerTx = &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 14600, Expire: 200}
)

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

func Test_lightBroadcast(t *testing.T) {

	q := queue.New("test")
	proto := newTestProtocolWithQueue(q)
	defer proto.Ctx.Done()
	proto.cfg.MinLtBlockSize = 0
	proto.cfg.LtBlockPendTimeout = 1
	defer q.Close()
	memTxList := []*types.Transaction{tx, tx1, nil}
	done := startHandleMempool(q, &memTxList)
	<-done
	pid := proto.Host.ID()
	block := &types.Block{TxHash: []byte("test"), Txs: []*types.Transaction{minerTx, tx, tx1, tx2}, Height: 10}
	peerMsgCh := proto.ps.Sub(psBroadcast)
	proto.ltB.addLtBlock(proto.buildLtBlock(block), pid)
	// 缺少tx2, 组装失败, 等待1s超时
	msg := <-peerMsgCh
	require.Equal(t, msg.(psMsg).msg.(*types.PeerPubSubMsg).MsgID, blockReqMsgID)
	require.Equal(t, msg.(psMsg).topic, proto.getPeerTopic(pid))

	//交易组
	txGroup, _ := types.CreateTxGroup([]*types.Transaction{tx1, tx2}, proto.ChainCfg.GetMinTxFeeRate())
	gtx := txGroup.Tx()
	memTxList = []*types.Transaction{tx, gtx}
	blcCli := q.Client()
	blcCli.Sub("blockchain")
	proto.ltB.addLtBlock(proto.buildLtBlock(block), pid)
	// 组装成功, 发送到blockchain模块, 模拟blockchain接收数据
	msg1 := <-blcCli.Recv()
	require.Equal(t, types.EventBroadcastAddBlock, int(msg1.Ty))
	blc, ok := msg1.Data.(*types.BlockPid)
	require.True(t, ok)
	require.Equal(t, pid.Pretty(), blc.Pid)
}

func Test_blockRequest(t *testing.T) {

	q := queue.New("test")
	proto := newTestProtocolWithQueue(q)
	defer proto.Ctx.Done()
	api := &mocks.QueueProtocolAPI{}
	proto.API = api
	pid := proto.Host.ID()
	// blockchain获取失败, 返回
	var height int64 = 10
	atomic.StoreInt64(&proto.currHeight, height)
	api.On("GetBlocks", mock.Anything).Return(nil, types.ErrActionNotSupport).Once()
	proto.ltB.addBlockRequest(height, pid)
	require.Equal(t, 0, proto.ltB.blockRequestList.Len())
	testBlock := &types.Block{Txs: []*types.Transaction{minerTx}, Height: 10}
	details := &types.BlockDetails{
		Items: []*types.BlockDetail{
			{
				Block: testBlock,
			},
		},
	}
	// 本地高度未达到, 加入到等待队列
	atomic.StoreInt64(&proto.currHeight, height-1)
	api.On("GetBlocks", mock.Anything).Return(details, nil)
	proto.ltB.addBlockRequest(10, pid)
	require.Equal(t, 1, proto.ltB.blockRequestList.Len())

	// 正确获取流程
	peerMsgCh := proto.ps.Sub(psBroadcast)
	atomic.StoreInt64(&proto.currHeight, height)
	// 缺少tx2, 组装失败, 等待1s超时
	msg := <-peerMsgCh
	require.Equal(t, msg.(psMsg).msg.(*types.PeerPubSubMsg).MsgID, blockRespMsgID)
	require.Equal(t, msg.(psMsg).topic, proto.getPeerTopic(pid))
}
