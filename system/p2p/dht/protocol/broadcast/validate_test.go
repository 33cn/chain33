// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/33cn/chain33/client/mocks"
	"github.com/stretchr/testify/mock"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	ps "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/stretchr/testify/require"
)

func Test_handleBroadcastReply(t *testing.T) {

	val := newValidator(newTestPubSub())
	reply := &types.Reply{IsOk: true}
	val.handleBroadcastReply(reply, &broadcastMsg{})
	require.Equal(t, 0, len(val.deniedPeers))
	reply.IsOk = false
	reply.Msg = []byte(types.ErrMemFull.Error())
	val.handleBroadcastReply(reply, nil)
	require.Equal(t, 0, len(val.deniedPeers))
	msg := &broadcastMsg{msg: &queue.Message{Ty: types.EventTx}, publisher: "testpid1"}
	reply.Msg = []byte(types.ErrNoBalance.Error())
	val.handleBroadcastReply(reply, msg)
	require.Equal(t, 1, len(val.deniedPeers))
	info, ok := val.deniedPeers["testpid1"]
	require.True(t, ok)
	require.Equal(t, 1, info.count)
}

func Test_deniedPeerMethod(t *testing.T) {

	val := newValidator(newTestPubSub())
	val.addDeniedPeer("testpid1", 0)
	val.addDeniedPeer("testpid2", 10)
	require.False(t, val.isDeniedPeer("testpid1"))
	require.True(t, val.isDeniedPeer("testpid2"))
	require.Equal(t, 2, len(val.deniedPeers))
	val.recoverDeniedPeers()
	require.Equal(t, 2, len(val.deniedPeers))
	val.reduceDeniedCount("testpid1")
	val.recoverDeniedPeers()
	require.Equal(t, 1, len(val.deniedPeers))
	info, _ := val.deniedPeers["testpid2"]
	currFreeTimestamp := info.freeTimestamp
	val.addDeniedPeer("testpid2", 1)
	val.addDeniedPeer("testpid2", 1)
	require.Equal(t, 3, info.count)
	require.Equal(t, currFreeTimestamp+4+8, info.freeTimestamp)
}

func Test_broadcastMsg(t *testing.T) {
	val := newValidator(newTestPubSub())
	q := queue.New("test")
	defer q.Close()
	val.QueueClient = q.Client()
	val.Ctx = context.Background()
	msg := &broadcastMsg{msg: queue.NewMessage(0, "p2p", types.EventTx, nil), publisher: "testpid1"}
	msg.msg.Reply(&queue.Message{Data: &types.Reply{Msg: []byte(types.ErrNoBalance.Error())}})
	val.addBroadcastMsg(msg)
	require.Equal(t, 1, val.msgList.Len())
	val.copyMsgList()
	require.Equal(t, 0, val.msgList.Len())
	require.Equal(t, 1, len(val.msgBuf))
	val.addBroadcastMsg(msg)
	go val.manageDeniedPeer()
	testDone := make(chan struct{})
	go func() {
		for !val.isDeniedPeer("testpid1") {
			time.Sleep(time.Millisecond * 100)
		}
		testDone <- struct{}{}
	}()

	select {
	case <-testDone:
	case <-time.After(time.Second * 5):
		t.Error("test broadcast msg timeout")
	}
}

func Test_blockHeaderCache(t *testing.T) {

	val := newValidator(newTestPubSub())
	for i := 0; i < blkHeaderCacheSize*2; i++ {
		val.addBlockHeader(&types.Header{Height: int64(i * 2)})
	}
	require.Equal(t, blkHeaderCacheSize/2, len(val.blkHeaderCache))
}

func Test_validateBlock(t *testing.T) {
	val := newValidator(newTestPubSub())
	proto, cancel := newTestProtocol()
	defer cancel()
	val.broadcastProtocol = proto
	msg := &ps.Message{Message: &pubsub_pb.Message{From: []byte(val.Host.ID())}}
	require.Equal(t, ps.ValidationAccept, val.validateBlock(val.Ctx, val.Host.ID(), msg))
	val.addDeniedPeer("errpid", 10)
	msg = &ps.Message{Message: &pubsub_pb.Message{From: []byte("errpid")}}
	require.Equal(t, ps.ValidationReject, val.validateBlock(val.Ctx, "errpid", msg))
	msg = &ps.Message{Message: &pubsub_pb.Message{Data: []byte("errmsg")}}
	require.Equal(t, ps.ValidationReject, val.validateBlock(val.Ctx, "testpid", msg))

	testBlock := &types.Block{Height: 1}
	sendBuf := make([]byte, 0)
	msg.Data = val.encodeMsg(testBlock, &sendBuf)
	blockHash := hex.EncodeToString(testBlock.Hash(val.ChainCfg))
	val.blockFilter.AddWithCheckAtomic(blockHash, struct{}{})
	require.Equal(t, ps.ValidationIgnore, val.validateBlock(val.Ctx, "testpid", msg))
	val.blockFilter.Remove(blockHash)
	require.Equal(t, ps.ValidationAccept, val.validateBlock(val.Ctx, "testpid", msg))
}

func Test_validatePeer(t *testing.T) {
	val := newValidator(newTestPubSub())
	val.addDeniedPeer("errpid", 10)
	topic := "tx"
	msg := &ps.Message{Message: &pubsub_pb.Message{Topic: &topic, From: []byte("errpid")}}
	require.Equal(t, ps.ValidationReject, val.validatePeer(val.Ctx, "", msg))
	msg.From = []byte("normalPid")
	require.Equal(t, ps.ValidationAccept, val.validatePeer(val.Ctx, "", msg))
}

func Test_validateTx(t *testing.T) {

	val := newValidator(newTestPubSub())
	proto, cancel := newTestProtocol()
	defer cancel()
	val.broadcastProtocol = proto
	msg := &ps.Message{Message: &pubsub_pb.Message{From: []byte(val.Host.ID())}}
	require.Equal(t, ps.ValidationAccept, val.validateTx(val.Ctx, val.Host.ID(), msg))

	msg = &ps.Message{Message: &pubsub_pb.Message{Data: []byte("errmsg")}}
	require.Equal(t, ps.ValidationReject, val.validateTx(val.Ctx, "testpid", msg))

	tx := &types.Transaction{Execer: []byte("coins")}
	sendBuf := make([]byte, 0)
	msg.Data = val.encodeMsg(tx, &sendBuf)
	txHash := hex.EncodeToString(tx.Hash())
	val.txFilter.AddWithCheckAtomic(txHash, struct{}{})
	require.Equal(t, ps.ValidationIgnore, val.validateTx(val.Ctx, "testpid", msg))
	val.txFilter.Remove(txHash)

	mockAPI := new(mocks.QueueProtocolAPI)
	val.API = mockAPI
	mockAPI.On("SendTx", mock.Anything).Return(nil, types.ErrNotSync).Once()
	require.Equal(t, ps.ValidationIgnore, val.validateTx(val.Ctx, "testpid", msg))
	val.txFilter.Remove(txHash)

	mockAPI.On("SendTx", mock.Anything).Return(nil, nil).Once()
	require.Equal(t, ps.ValidationAccept, val.validateTx(val.Ctx, "testpid", msg))
}

func Test_validateBatchTx(t *testing.T) {

	val := newValidator(newTestPubSub())
	proto, cancel := newTestProtocol()
	defer cancel()
	val.broadcastProtocol = proto
	msg := &ps.Message{Message: &pubsub_pb.Message{From: []byte(val.Host.ID())}}
	require.Equal(t, ps.ValidationAccept, val.validateBatchTx(val.Ctx, val.Host.ID(), msg))

	msg = &ps.Message{Message: &pubsub_pb.Message{Data: []byte("errmsg")}}
	require.Equal(t, ps.ValidationReject, val.validateBatchTx(val.Ctx, "testpid", msg))

	tx := &types.Transaction{Execer: []byte("coins")}
	txs := &types.Transactions{Txs: []*types.Transaction{tx}}
	sendBuf := make([]byte, 0)
	msg.Data = val.encodeMsg(txs, &sendBuf)
	mockAPI := new(mocks.QueueProtocolAPI)
	val.API = mockAPI
	mockAPI.On("SendTx", mock.Anything).Return(nil, nil)
	require.Equal(t, ps.ValidationAccept, val.validateBatchTx(val.Ctx, "testpid", msg))

	txs.Txs = append(txs.Txs, &types.Transaction{Execer: []byte("coins2")})
	msg.Data = val.encodeMsg(txs, &sendBuf)
	require.Equal(t, ps.ValidationIgnore, val.validateBatchTx(val.Ctx, "testpid", msg))
}
