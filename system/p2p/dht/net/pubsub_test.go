package net

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testMsg(msg *SubMsg) {
	fmt.Println("testMsg", msg.From, "data", string(msg.Data))
}

func Test_pubsub(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := getNetHosts(ctx, 2, t)
	connect(t, hosts[0], hosts[1])
	psub, err := NewPubSub(ctx, hosts[0])
	assert.Nil(t, err)
	err = psub.JoinTopicAndSubTopic("bztest", testMsg)
	assert.Nil(t, err)
	err = psub.Publish("bztest", []byte("hello,world"))
	assert.Nil(t, err)

	err = psub.Publish("bztest2", []byte("hello,world"))
	assert.NotNil(t, err)
	topics := psub.GetTopics()
	assert.Equal(t, 1, len(topics))
	assert.Equal(t, "bztest", topics[0])
	assert.False(t, psub.HasTopic("mytest"))
	peers := psub.FetchTopicPeers("mytest")
	t.Log(peers)
	assert.Equal(t, 0, len(peers))
	peers = psub.FetchTopicPeers("bztest")
	t.Log(peers)

	psub.RemoveTopic("bztest")
	time.Sleep(time.Second * 2)
	assert.Equal(t, 0, psub.TopicNum())

}
