// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package extension

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// pubsubTracer pubsub 系统内部事件跟踪器
type pubsubTracer struct {
	blankTracer
	subLock     sync.RWMutex
	msgLock     sync.Mutex
	host        host.Host
	subscribers map[string]SubCallBack
	pendMsgList list.List
}

func newPubsubTracer(ctx context.Context, host host.Host) *pubsubTracer {
	pt := &pubsubTracer{host: host}
	pt.subscribers = make(map[string]SubCallBack)
	go pt.handlePendMsg(ctx)
	return pt
}

var _ pubsub.RawTracer = (*pubsubTracer)(nil)

func (pt *pubsubTracer) handlePendMsg(ctx context.Context) {

	ticker := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			msgs := pt.copyPendMsg()
			for _, msg := range msgs {
				cb := pt.getSubCallBack(*msg.Topic)
				cb(*msg.Topic, msg)
			}
		}

	}
}

func (pt *pubsubTracer) addSubscriber(topic string, cb SubCallBack) {
	pt.subLock.Lock()
	pt.subscribers[topic] = cb
	pt.subLock.Unlock()
}

func (pt *pubsubTracer) getSubCallBack(topic string) SubCallBack {
	pt.subLock.RLock()
	defer pt.subLock.RUnlock()
	return pt.subscribers[topic]
}

func (pt *pubsubTracer) copyPendMsg() []*pubsub.Message {
	pt.msgLock.Lock()
	defer pt.msgLock.Unlock()
	if pt.pendMsgList.Len() <= 0 {
		return nil
	}
	msgs := make([]*pubsub.Message, 0, pt.pendMsgList.Len())
	for it := pt.pendMsgList.Front(); it != nil; it = it.Next() {
		msgs = append(msgs, it.Value.(*pubsub.Message))
	}
	pt.pendMsgList.Init()
	return msgs
}

// 监听subscribe数据丢弃事件, 实现重发机制
func (pt *pubsubTracer) UndeliverableMessage(msg *pubsub.Message) {
	if pt.host.ID() == msg.ReceivedFrom {
		return
	}
	pt.msgLock.Lock()
	defer pt.msgLock.Unlock()
	pt.pendMsgList.PushBack(msg)
}

type blankTracer struct{}

func (*blankTracer) UndeliverableMessage(msg *pubsub.Message)         {}
func (*blankTracer) DeliverMessage(msg *pubsub.Message)               {}
func (*blankTracer) RejectMessage(msg *pubsub.Message, reason string) {}
func (*blankTracer) ValidateMessage(msg *pubsub.Message)              {}
func (*blankTracer) AddPeer(p peer.ID, proto protocol.ID)             {}
func (*blankTracer) RemovePeer(p peer.ID)                             {}
func (*blankTracer) Join(topic string)                                {}
func (*blankTracer) Leave(topic string)                               {}
func (*blankTracer) Graft(p peer.ID, topic string)                    {}
func (*blankTracer) Prune(p peer.ID, topic string)                    {}
func (*blankTracer) DuplicateMessage(msg *pubsub.Message)             {}
func (*blankTracer) RecvRPC(rpc *pubsub.RPC)                          {}
func (*blankTracer) SendRPC(rpc *pubsub.RPC, p peer.ID)               {}
func (*blankTracer) DropRPC(rpc *pubsub.RPC, p peer.ID)               {}
func (*blankTracer) ThrottlePeer(p peer.ID)                           {}
