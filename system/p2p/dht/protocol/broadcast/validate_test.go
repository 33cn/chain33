// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	ps "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/stretchr/testify/require"
)

func Test_handleBroadcastReply(t *testing.T) {

	val := newValidator(newTestPubSub())
	reply := &types.Reply{IsOk: true}
	val.handleBroadcastReply(reply, nil)
	require.Equal(t, 0, len(val.deniedPeers))
	reply.IsOk = false
	reply.Msg = []byte(types.ErrMemFull.Error())
	val.handleBroadcastReply(reply, nil)
	require.Equal(t, 0, len(val.deniedPeers))
	msg := &broadcastMsg{msg: &queue.Message{Ty: types.EventTx}, publisher: "testpid1"}
	reply.Msg = []byte(types.ErrNoBalance.Error())
	val.handleBroadcastReply(reply, msg)
	require.Equal(t, 1, len(val.deniedPeers))
	endTime, ok := val.deniedPeers["testpid1"]
	require.True(t, ok)
	require.True(t, endTime <= types.Now().Unix()+errTxDenyTime)
}

func Test_deniedPeerMethod(t *testing.T) {

	val := newValidator(newTestPubSub())
	val.addDeniedPeer("testpid1", 0)
	val.addDeniedPeer("testpid2", 2)
	require.False(t, val.isDeniedPeer("testpid1"))
	require.True(t, val.isDeniedPeer("testpid2"))
	require.Equal(t, 2, len(val.deniedPeers))
	val.recoverDeniedPeers()
	require.Equal(t, 1, len(val.deniedPeers))
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

	require.Equal(t, ps.ValidationAccept, val.validateBlock(val.Ctx, val.Host.ID(), nil))
	val.addDeniedPeer("errpid", 10)
	require.Equal(t, ps.ValidationReject, val.validateBlock(val.Ctx, "errpid", nil))
	msg := &ps.Message{Message: &pubsub_pb.Message{Data: []byte("errmsg")}}
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
	msg := &ps.Message{Message: &pubsub_pb.Message{Topic: &topic}}
	require.Equal(t, ps.ValidationReject, val.validatePeer(val.Ctx, "errpid", msg))
	require.Equal(t, ps.ValidationAccept, val.validatePeer(val.Ctx, "normalpid", msg))
}
