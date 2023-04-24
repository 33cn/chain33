// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package extension

import (
	"context"
	"fmt"
	"testing"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/peer"
	blankhost "github.com/libp2p/go-libp2p/p2p/host/blank"
	"github.com/stretchr/testify/require"
)

type mockHost struct {
	blankhost.BlankHost
}

func (h *mockHost) ID() peer.ID {
	return "testid"
}

func Test_trace(t *testing.T) {

	topic := "testtopic"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pt := newPubsubTracer(ctx, &mockHost{})
	msgNum := 1000
	msgChan := make(chan SubMsg, msgNum)
	testDone := make(chan struct{})

	pt.addSubscriber(topic, func(tp string, msg SubMsg) {
		msgChan <- msg
	})

	go func() {
		for i := 0; i < msgNum; i++ {
			msg := <-msgChan
			require.Equal(t, topic, *msg.Topic)
			id := peer.ID(fmt.Sprintf("id%d", i))
			require.Equal(t, id, msg.ReceivedFrom)
		}
		testDone <- struct{}{}
	}()
	go func() {
		for i := 0; i < msgNum; i++ {
			msg := &pubsub.Message{Message: &pubsub_pb.Message{}}
			msg.Topic = &topic
			msg.ReceivedFrom = peer.ID(fmt.Sprintf("id%d", i))
			pt.UndeliverableMessage(msg)
		}
	}()

	select {
	case <-testDone:
	case <-time.After(time.Second * 5):
		t.Error("test trace timeout")
	}
}
