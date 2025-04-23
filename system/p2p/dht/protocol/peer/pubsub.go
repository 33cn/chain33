package peer

import (
	"fmt"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/extension"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

// 处理订阅topic的请求
func (p *Protocol) handleEventSubTopic(msg *queue.Message) {
	//先检查是否已经订阅相关topic
	//接收chain33其他模块发来的请求消息
	subtopic := msg.GetData().(*types.SubTopic)
	topic := subtopic.GetTopic()
	//check topic
	moduleName := subtopic.GetModule()
	//模块名，用来收到订阅的消息后转发给对应的模块名
	if !p.Pubsub.HasTopic(topic) {
		err := p.Pubsub.JoinAndSubTopic(topic, p.subCallBack) //订阅topic
		if err != nil {
			log.Error("peerPubSub", "err", err)
			msg.Reply(p.QueueClient.NewMessage("", types.EventSubTopic, &types.Reply{IsOk: false, Msg: []byte(err.Error())}))
			return
		}
	}

	var reply types.SubTopicReply
	reply.Status = true
	reply.Msg = fmt.Sprintf("subtopic %v success", topic)
	msg.Reply(p.QueueClient.NewMessage("", types.EventSubTopic, &types.Reply{IsOk: true, Msg: types.Encode(&reply)}))

	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()

	var modules map[string]bool
	v, ok := p.topicModule.Load(topic)
	if ok {
		modules = v.(map[string]bool)
	} else {
		modules = make(map[string]bool)
	}
	modules[moduleName] = true
	p.topicModule.Store(topic, modules)
	//接收订阅的消息
}

// 处理收到的数据
func (p *Protocol) subCallBack(topic string, msg extension.SubMsg) {
	p.topicMutex.RLock()
	defer p.topicMutex.RUnlock()
	// 忽略来自本节点的数据
	if peer.ID(msg.From) == p.Host.ID() {
		return
	}
	modules, ok := p.topicModule.Load(topic)
	if !ok {
		return
	}

	for moduleName := range modules.(map[string]bool) {
		newMsg := p.QueueClient.NewMessage(moduleName, types.EventReceiveSubData, &types.TopicData{Topic: topic, From: peer.ID(msg.From).String(), Data: msg.Data}) //加入到输出通道)
		_ = p.QueueClient.Send(newMsg, false)
	}
}

// 获取所有已经订阅的topic
func (p *Protocol) handleEventGetTopics(msg *queue.Message) {
	_, ok := msg.GetData().(*types.FetchTopicList)
	if !ok {
		msg.Reply(p.QueueClient.NewMessage("", types.EventFetchTopics, &types.Reply{IsOk: false, Msg: []byte("need *types.FetchTopicList")}))
		return
	}
	//获取topic列表
	topics := p.Pubsub.GetTopics()
	var reply types.TopicList
	reply.Topics = topics
	msg.Reply(p.QueueClient.NewMessage("", types.EventFetchTopics, &types.Reply{IsOk: true, Msg: types.Encode(&reply)}))
}

// 删除已经订阅的某一个topic
func (p *Protocol) handleEventRemoveTopic(msg *queue.Message) {
	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()

	v, ok := msg.GetData().(*types.RemoveTopic)
	if !ok {
		msg.Reply(p.QueueClient.NewMessage("", types.EventRemoveTopic, &types.Reply{IsOk: false, Msg: []byte("need *types.RemoveTopic")}))
		return
	}

	vmodules, ok := p.topicModule.Load(v.GetTopic())
	if !ok || len(vmodules.(map[string]bool)) == 0 {
		msg.Reply(p.QueueClient.NewMessage("", types.EventRemoveTopic, &types.Reply{IsOk: false, Msg: []byte("this module no sub this topic")}))
		return
	}
	modules := vmodules.(map[string]bool)
	delete(modules, v.GetModule()) //删除消息推送的module
	var reply types.RemoveTopicReply
	reply.Topic = v.GetTopic()
	reply.Status = true

	if len(modules) != 0 {
		msg.Reply(p.QueueClient.NewMessage("", types.EventRemoveTopic, &types.Reply{IsOk: true, Msg: types.Encode(&reply)}))
		return
	}

	p.Pubsub.RemoveTopic(v.GetTopic())
	msg.Reply(p.QueueClient.NewMessage("", types.EventRemoveTopic, &types.Reply{IsOk: true, Msg: types.Encode(&reply)}))
}

// 发布Topic消息
func (p *Protocol) handleEventPubMsg(msg *queue.Message) {
	v, ok := msg.GetData().(*types.PublishTopicMsg)
	if !ok {
		msg.Reply(p.QueueClient.NewMessage("", types.EventPubTopicMsg, &types.Reply{IsOk: false, Msg: []byte("need *types.PublishTopicMsg")}))
		return
	}
	var isOK = true
	replyInfo := "push success"
	err := p.Pubsub.Publish(v.GetTopic(), v.GetMsg())
	if err != nil {
		//publish msg failed
		isOK = false
		replyInfo = err.Error()
	}
	msg.Reply(p.QueueClient.NewMessage("", types.EventPubTopicMsg, &types.Reply{IsOk: isOK, Msg: []byte(replyInfo)}))
}
