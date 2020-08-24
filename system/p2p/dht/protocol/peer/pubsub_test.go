package peer

import (
	"time"

	"github.com/33cn/chain33/system/p2p/dht/net"

	l "github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/queue"

	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"

	"github.com/33cn/chain33/types"

	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	l.SetLogLevel("err")
}

func newTestpeerPubSubWithQueue(q queue.Queue) *peerPubSub {
	env := newTestEnv(q)
	protocol := &peerPubSub{}
	protocol.BaseProtocol = new(prototypes.BaseProtocol)
	prototypes.ClearEventHandler()
	protocol.InitProtocol(env)

	return protocol
}

func newTestPubProtocol(q queue.Queue) *peerPubSub {

	return newTestpeerPubSubWithQueue(q)
}

func TestPeerPubSubInitProtocol(t *testing.T) {
	q := queue.New("test")
	protocol := newTestPubProtocol(q)
	assert.NotNil(t, protocol)

}

//测试订阅topic
func testHandleSubTopicEvent(protocol *peerPubSub, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleSubTopic(msg)
}

//测试删除topic

func testHandleRemoveTopicEvent(protocol *peerPubSub, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleRemoveTopc(msg)
}

//测试获取topiclist

func testHandleGetTopicsEvent(protocol *peerPubSub, msg *queue.Message) {

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handleGetTopics(msg)

}

//测试pubmsg
func testHandlerPubMsg(protocol *peerPubSub, msg *queue.Message) {
	defer func() {
		if r := recover(); r != nil {
		}
	}()

	protocol.handlePubMsg(msg)
}
func testSubTopic(t *testing.T, protocol *peerPubSub) {

	msgs := make([]*queue.Message, 0)
	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "blockchain",
		Topic:  "bzTest",
	}))

	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "blockchain",
		Topic:  "bzTest",
	}))

	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "mempool",
		Topic:  "bzTest",
	}))

	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "rpc",
		Topic:  "rtopic",
	}))

	msgs = append(msgs, protocol.QueueClient.NewMessage("p2p", types.EventSubTopic, &types.SubTopic{
		Module: "rpc",
		Topic:  "rtopic",
	}))

	//发送订阅请求
	for _, msg := range msgs {
		testHandleSubTopicEvent(protocol, msg)
		replyMsg, err := protocol.GetQueueClient().Wait(msg)
		assert.Nil(t, err)

		subReply, ok := replyMsg.GetData().(*types.Reply)
		if ok {
			t.Log("subReply status", subReply.IsOk)
			if subReply.IsOk {
				var reply types.SubTopicReply
				types.Decode(subReply.GetMsg(), &reply)
				assert.NotNil(t, reply)
				t.Log("reply", reply.GetMsg())
			} else {
				//订阅失败
				t.Log("subfailed Reply ", string(subReply.GetMsg()))
			}

		}

	}

}

func testPushMsg(t *testing.T, protocol *peerPubSub) {
	pubTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventPubTopicMsg, &types.PublishTopicMsg{Topic: "bzTest", Msg: []byte("one two tree four")})
	testHandlerPubMsg(protocol, pubTopicMsg)
	resp, err := protocol.GetQueueClient().WaitTimeout(pubTopicMsg, time.Second*10)
	assert.Nil(t, err)
	rpy := resp.GetData().(*types.Reply)
	t.Log("bzTest isok", rpy.IsOk, "msg", string(rpy.GetMsg()))
	assert.True(t, rpy.IsOk)
	//换个不存在的topic测试
	pubTopicMsg = protocol.QueueClient.NewMessage("p2p", types.EventPubTopicMsg, &types.PublishTopicMsg{Topic: "bzTest2", Msg: []byte("one two tree four")})
	testHandlerPubMsg(protocol, pubTopicMsg)
	resp, err = protocol.GetQueueClient().WaitTimeout(pubTopicMsg, time.Second*10)
	assert.Nil(t, err)
	rpy = resp.GetData().(*types.Reply)
	t.Log("bzTest2 isok", rpy.IsOk, "msg", string(rpy.GetMsg()))
	assert.False(t, rpy.IsOk)
	errPubTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventPubTopicMsg, &types.FetchTopicList{})
	testHandlerPubMsg(protocol, errPubTopicMsg)

}

//测试获取topiclist
func testFetchTopics(t *testing.T, protocol *peerPubSub) []string {
	//获取topiclist
	fetchTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventFetchTopics, &types.FetchTopicList{})
	testHandleGetTopicsEvent(protocol, fetchTopicMsg)
	resp, err := protocol.GetQueueClient().WaitTimeout(fetchTopicMsg, time.Second*10)
	assert.Nil(t, err)
	//获取topiclist
	assert.True(t, resp.GetData().(*types.Reply).GetIsOk())
	var topiclist types.TopicList
	err = types.Decode(resp.GetData().(*types.Reply).GetMsg(), &topiclist)
	assert.Nil(t, err)

	t.Log("topiclist", topiclist.GetTopics())

	errFetchTopicMsg := protocol.QueueClient.NewMessage("p2p", types.EventFetchTopics, &types.PublishTopicMsg{})
	testHandleGetTopicsEvent(protocol, errFetchTopicMsg)
	return topiclist.GetTopics()
}

//测试推送订阅的消息内容

func testSendTopicData(t *testing.T, protocol *peerPubSub) {
	//发送收到的订阅消息,预期mempool,blockchain模块都会收到 hello,world 1
	//protocol.msgChan <- &types.TopicData{Topic: "bzTest", From: "123435555", Data: []byte("hello,world 1")}
	protocol.subCallBack(&net.SubMsg{Data: []byte("hello,world 1"), From: "123435555", Topic: "bzTest"})

}

//测试删除某一个module的topic
func testRemoveModuleTopic(t *testing.T, protocol *peerPubSub, topic, module string) {
	//删除topic
	removetopic := protocol.QueueClient.NewMessage("p2p", types.EventRemoveTopic, &types.RemoveTopic{
		Topic:  topic,
		Module: module,
	})
	//预期只有mempool模块收到hello,world 2
	testHandleRemoveTopicEvent(protocol, removetopic) //删除blockchain的订阅消息
	protocol.subCallBack(&net.SubMsg{Data: []byte("hello,world 2"), From: "123435555", Topic: "bzTest"})
	//protocol.msgChan <- &types.TopicData{Topic: "bzTest", From: "123435555", Data: []byte("hello,world 2")}

	errRemovetopic := protocol.QueueClient.NewMessage("p2p", types.EventRemoveTopic, &types.FetchTopicList{})
	testHandleRemoveTopicEvent(protocol, errRemovetopic) //删除blockchain的订阅消息

	errRemovetopic = protocol.QueueClient.NewMessage("p2p", types.EventRemoveTopic, &types.RemoveTopic{Topic: "haha",
		Module: module})
	testHandleRemoveTopicEvent(protocol, errRemovetopic) //删除blockchain的订阅消息

}

func testBlockRecvSubData(t *testing.T, q queue.Queue) {
	client := q.Client()
	client.Sub("blockchain")
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
	client.Sub("mempool")
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
	protocol := newTestPubProtocol(q)
	testSubTopic(t, protocol) //订阅topic

	topics := testFetchTopics(t, protocol) //获取topic list
	assert.Equal(t, len(topics), 2)
	testSendTopicData(t, protocol) //通过chan推送接收到的消息

	testPushMsg(t, protocol)                                   //发布消息
	testRemoveModuleTopic(t, protocol, "bzTest", "blockchain") //删除某一个模块的topic
	topics = testFetchTopics(t, protocol)                      //获取topic list
	assert.Equal(t, len(topics), 2)
	//--------
	testRemoveModuleTopic(t, protocol, "rtopic", "rpc") //删除某一个模块的topic
	topics = testFetchTopics(t, protocol)
	//t.Log("after Remove rtopic", topics)
	assert.Equal(t, 1, len(topics))
	testRemoveModuleTopic(t, protocol, "bzTest", "mempool") //删除某一个模块的topic
	topics = testFetchTopics(t, protocol)
	//t.Log("after Remove bzTest", topics)
	assert.Equal(t, 0, len(topics))

	time.Sleep(time.Second)
}
