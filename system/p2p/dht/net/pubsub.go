package net

import (
	"context"
	"errors"
	"fmt"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/libp2p/go-libp2p-pubsub"
	"sync"
)

type TopicMap map[string]*topicinfo

type topicinfo struct {
	pubtopic *pubsub.Topic
	sub      *pubsub.Subscription
	ctx      context.Context
	cancel   context.CancelFunc
	topic    string
}

type subinfo struct {
	sub    *pubsub.Subscription
	ctx    context.Context
	cancel context.CancelFunc
	topic  string
}
type PubSub struct {
	ps         *pubsub.PubSub
	topics     TopicMap
	topicMutex sync.Mutex
	ctx        context.Context
}

type SubMsg struct {
	Data  []byte
	Topic string
	From  string
}

func NewPubSub(ctx context.Context, host host.Host) (*PubSub, error) {
	p := &PubSub{
		ps:     nil,
		topics: make(TopicMap, 0),
	}
	//选择使用GossipSub
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, err
	}

	p.ps = ps
	p.ctx = ctx
	p.topics = make(TopicMap)
	return p, nil
}

func (p *PubSub) GetTopics() []string {
	return p.ps.GetTopics()
}

func (p *PubSub) HasTopic(topic string) bool {
	_, ok := p.topics[topic]
	return ok
}

//加入topic&subTopic
func (p *PubSub) JoinTopicAndSubTopic(topic string, mchan chan interface{}, opts ...pubsub.TopicOpt) error {
	Topic, err := p.ps.Join(topic, opts...)
	if err != nil {
		return err
	}

	subscription, err := Topic.Subscribe()
	if err != nil {
		return err
	}
	//p.topics = append(p.topics, Topic)
	ctx, cancel := context.WithCancel(p.ctx)

	p.topicMutex.Lock()
	p.topics[topic] = &topicinfo{
		pubtopic: Topic,
		ctx:      ctx,
		topic:    topic,
		cancel:   cancel,
		sub:      subscription,
	}
	p.topicMutex.Unlock()

	return nil
}

//Publish 发布消息
func (p *PubSub) Publish(topic string, msg []byte) error {
	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
	t, ok := p.topics[topic]
	if !ok {
		log.Error("publish", "no this topic", topic)
		return errors.New(fmt.Sprintf("no this topic:%v", topic))
	}

	err := t.pubtopic.Publish(t.ctx, msg)
	if err != nil {
		log.Error("publish", "err", err)
		return err
	}
	return nil
}

func (p *PubSub) SubTopic(msg chan interface{}) {
	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()

	for _, info := range p.topics {

		go func(info *topicinfo) {
			for {
				topic := info.sub.Topic()
				got, err := info.sub.Next(info.ctx)
				if err != nil {
					log.Error("SubMsg", "topic msg err", err, "topic", topic)
					if err == p.ctx.Err() {
						return
					}
				}
				log.Info("SubMsg", "readData", string(got.GetData()), "msgID")
				var data SubMsg
				data.Data = got.GetData()
				data.Topic = topic
				data.From = got.GetFrom().String()
				msg <- data
			}
		}(info)
	}
}

func (p *PubSub) RemoveTopic(topic string) {

	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()

	info, ok := p.topics[topic]
	if ok {
		log.Info("RemoveTopic", topic)
		info.cancel()
		info.sub.Cancel()
		err := info.pubtopic.Close()
		if err != nil {
			log.Error("RemoveTopic", "topic", err)
		}

	}

	delete(p.topics, topic)

}

func (p *PubSub) FetchTopicPeers(topic string) []peer.ID {
	p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
	topicobj, ok := p.topics[topic]
	if ok {
		return topicobj.pubtopic.ListPeers()
	}
	return nil
}

func (p *PubSub) FetchTopics() []string {
	return p.ps.GetTopics()
	/*p.topicMutex.Lock()
	defer p.topicMutex.Unlock()
	var topics []string
	for topic := range p.topics {
		topics = append(topics, topic)
	}
	return topics*/
}

func (p *PubSub) TopicNum() int {
	return len(p.ps.GetTopics())
	/*
		p.topicMutex.Lock()
		defer p.topicMutex.Unlock()
		return len(p.topics)
	*/
}
