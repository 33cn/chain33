package peer

import (
	"testing"
	"time"

	l "github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/stretchr/testify/require"
)

func init() {
	l.SetLogLevel("err")
}

// 测试订阅topic
func testHandleSubTopicEvent(p *Protocol, msg *queue.Message) {
	p.handleEventSubTopic(msg)
}

// 测试删除topic
func testHandleRemoveTopicEvent(p *Protocol, msg *queue.Message) {
	p.handleEventRemoveTopic(msg)
}

// 测试获取topicList
func testHandleGetTopicsEvent(p *Protocol, msg *queue.Message) {
	p.handleEventGetTopics(msg)
}

// 测试pubmsg
func testHandlerPubMsg(p *Protocol, msg *queue.Message) {
	p.handleEventPubMsg(msg)
}

func testSubTopic(t *testing.T, p *Protocol) {

	msgs := make([]*queue.Message, 0)
	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "blockchain",
		Topic:  "bzTest",
	}))

	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "blockchain",
		Topic:  "bzTest",
	}))

	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "mempool",
		Topic:  "bzTest",
	}))

	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "rpc",
		Topic:  "rtopic",
	}))

	msgs = append(msgs, p.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "rpc",
		Topic:  "rtopic",
	}))

	//发送订阅请求
	for _, msg := range msgs {
		testHandleSubTopicEvent(p, msg)
		replyMsg, err := p.QueueClient.WaitTimeout(msg, time.Second*10)
		require.Nil(t, err)

		subReply, ok := replyMsg.GetData().(*types.Reply)
		if ok {
			t.Log("subReply status", subReply.IsOk)
			if subReply.IsOk {
				var reply types.SubTopicReply
				types.Decode(subReply.GetMsg(), &reply)
				t.Log("reply", reply.GetMsg())
			} else {
				//订阅失败
				t.Log("subfailed Reply ", string(subReply.GetMsg()))
			}
		}
	}
}

func testPushMsg(t *testing.T, protocol *Protocol) {
	pubTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventPubTopicMsg, &types.PublishTopicMsg{Topic: "bzTest", Msg: []byte("one two tree four")})
	testHandlerPubMsg(protocol, pubTopicMsg)
	resp, err := protocol.QueueClient.WaitTimeout(pubTopicMsg, time.Second*10)
	require.Nil(t, err)
	rpy := resp.GetData().(*types.Reply)
	t.Log("bzTest isok", rpy.IsOk, "msg", string(rpy.GetMsg()))
	require.True(t, rpy.IsOk)
	//换个不存在的topic测试
	pubTopicMsg = protocol.QueueClient.NewMessage("p2p", types.EventPubTopicMsg, &types.PublishTopicMsg{Topic: "bzTest2", Msg: []byte("one two tree four")})
	testHandlerPubMsg(protocol, pubTopicMsg)
	resp, err = protocol.QueueClient.WaitTimeout(pubTopicMsg, time.Second*10)
	require.Nil(t, err)
	rpy = resp.GetData().(*types.Reply)
	t.Log("bzTest2 isok", rpy.IsOk, "msg", string(rpy.GetMsg()))
	require.False(t, rpy.IsOk)
	errPubTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventPubTopicMsg, &types.FetchTopicList{})
	testHandlerPubMsg(protocol, errPubTopicMsg)

}

// 测试获取topicList
func testFetchTopics(t *testing.T, protocol *Protocol) []string {
	//获取topicList
	fetchTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventFetchTopics, &types.FetchTopicList{})
	testHandleGetTopicsEvent(protocol, fetchTopicMsg)
	resp, err := protocol.QueueClient.WaitTimeout(fetchTopicMsg, time.Second*10)
	require.Nil(t, err)
	//获取topicList
	require.True(t, resp.GetData().(*types.Reply).GetIsOk())
	var topiclist types.TopicList
	err = types.Decode(resp.GetData().(*types.Reply).GetMsg(), &topiclist)
	require.Nil(t, err)

	t.Log("topicList", topiclist.GetTopics())

	errFetchTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventFetchTopics, &types.PublishTopicMsg{})
	testHandleGetTopicsEvent(protocol, errFetchTopicMsg)
	return topiclist.GetTopics()
}

// 测试推送订阅的消息内容
func testSendTopicData(t *testing.T, protocol *Protocol) {
	//发送收到的订阅消息,预期mempool,blockchain模块都会收到 hello,world 1
	//protocol.msgChan <- &types.TopicData{Topic: "bzTest", From: "123435555", Data: []byte("hello,world 1")}
	msg := &pubsub.Message{Message: &pubsub_pb.Message{Data: []byte("hello,world 1")}, ReceivedFrom: "123435555"}
	protocol.subCallBack("bzTest", msg)

}

// 测试删除某一个module的topic
func testRemoveModuleTopic(t *testing.T, p *Protocol, topic, module string) {
	//删除topic
	removetopic := p.QueueClient.NewMessage("p2p", types.EventRemoveTopic, &types.RemoveTopic{
		Topic:  topic,
		Module: module,
	})
	//预期只有mempool模块收到hello,world 2
	testHandleRemoveTopicEvent(p, removetopic) //删除blockchain的订阅消息
	msg := &pubsub.Message{Message: &pubsub_pb.Message{Data: []byte("hello,world 2")}, ReceivedFrom: "123435555"}
	p.subCallBack("bzTest", msg)
	//p.msgChan <- &types.TopicData{Topic: "bzTest", From: "123435555", Data: []byte("hello,world 2")}

	errRemovetopic := p.QueueClient.NewMessage("p2p", types.EventRemoveTopic, &types.FetchTopicList{})
	testHandleRemoveTopicEvent(p, errRemovetopic) //删除blockchain的订阅消息

	errRemovetopic = p.QueueClient.NewMessage("p2p", types.EventRemoveTopic, &types.RemoveTopic{Topic: "haha",
		Module: module})
	testHandleRemoveTopicEvent(p, errRemovetopic) //删除blockchain的订阅消息
}

func testBlockRecvSubData(t *testing.T, q queue.Queue) {
	client := q.Client()
	client.Sub(blockchain)
	go func() {
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventReceiveSubData:
				if req, ok := msg.GetData().(*types.TopicData); ok {
					t.Log("blockchain Recv from", req.GetFrom(), "topic:", req.GetTopic(), "data", string(req.GetData()))
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			}
		}
	}()
}

func testMempoolRecvSubData(t *testing.T, q queue.Queue) {
	client := q.Client()
	client.Sub(mempool)
	go func() {
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventReceiveSubData:
				if req, ok := msg.GetData().(*types.TopicData); ok {
					t.Log("mempool Recv", req.GetFrom(), "topic:", req.GetTopic(), "data", string(req.GetData()))
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}
			}
		}
	}()
}

func TestPubSub(t *testing.T) {
	q := queue.New("test")
	testBlockRecvSubData(t, q)
	testMempoolRecvSubData(t, q)
	protocol, cancel := initEnv(t, q)
	defer cancel()
	testSubTopic(t, protocol) //订阅topic

	topics := testFetchTopics(t, protocol) //获取topic list
	require.Equal(t, len(topics), 2)
	testSendTopicData(t, protocol) //通过chan推送接收到的消息

	testPushMsg(t, protocol)                                   //发布消息
	testRemoveModuleTopic(t, protocol, "bzTest", "blockchain") //删除某一个模块的topic
	topics = testFetchTopics(t, protocol)                      //获取topic list
	require.Equal(t, len(topics), 2)
	//--------
	testRemoveModuleTopic(t, protocol, "rtopic", "rpc") //删除某一个模块的topic
	topics = testFetchTopics(t, protocol)
	//t.Log("after Remove rtopic", topics)
	require.Equal(t, 1, len(topics))
	testRemoveModuleTopic(t, protocol, "bzTest", "mempool") //删除某一个模块的topic
	topics = testFetchTopics(t, protocol)
	//t.Log("after Remove bzTest", topics)
	require.Equal(t, 0, len(topics))

	time.Sleep(time.Second)
}
