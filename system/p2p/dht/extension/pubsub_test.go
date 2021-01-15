package extension

import (
	"context"
	"fmt"
	"testing"

	core "github.com/libp2p/go-libp2p-core"
	"github.com/stretchr/testify/require"
)

func testMsg(topic string, msg SubMsg) {
	fmt.Println("testMsg", core.PeerID(msg.From).String(), "data", string(msg.Data))
}

func Test_pubsub(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hosts := getNetHosts(ctx, 2, t)
	connect(t, hosts[0], hosts[1])

	psub, err := NewPubSub(ctx, hosts[0])
	require.Nil(t, err)
	err = psub.JoinAndSubTopic("bztest", testMsg)
	require.Nil(t, err)

	err = psub.Publish("bztest", []byte("hello,world"))
	require.Nil(t, err)
	err = psub.Publish("bztest2", []byte("hello,world"))
	require.NotNil(t, err)
	topics := psub.GetTopics()
	require.Equal(t, 1, len(topics))
	require.Equal(t, "bztest", topics[0])
	require.False(t, psub.HasTopic("mytest"))
	peers := psub.FetchTopicPeers("mytest")
	t.Log(peers)
	require.Equal(t, 0, len(peers))
	peers = psub.FetchTopicPeers("bztest")
	t.Log(peers)
	psub.RemoveTopic("bztest")
	require.Equal(t, 0, psub.TopicNum())

}
