// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"encoding/hex"
	"runtime"

	"github.com/33cn/chain33/types"
	"github.com/golang/snappy"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"

	net "github.com/33cn/chain33/system/p2p/dht/extension"
)

const (
	//监听发布给本节点的信息, 以pid作为topic
	psBroadcast          = "ps-init"
	psPeerMsgTopicPrefix = "peermsg/"
	psTxTopic            = "tx/v1.0.0"
	psBatchTxTopic       = "batchtx/v1.0"
	psBlockTopic         = "block/v1.0.0"
	psLtBlockTopic       = "ltblk/v1.0"
	blkHeaderCacheSize   = 128
)

// 基于libp2p pubsub插件广播
type pubSub struct {
	*broadcastProtocol
	peerTopic string
	val       *validator
}

// new pub sub
func initPubSubBroadcast(b *broadcastProtocol) *pubSub {
	p := &pubSub{broadcastProtocol: b}
	p.peerTopic = p.getPeerTopic(p.Host.ID())
	p.init()
	return p
}

// 广播入口函数，处理相关初始化
func (p *pubSub) init() {

	incoming := make(chan net.SubMsg, 1024) //sub 接收通道, 订阅外部广播消息
	outgoing := p.ps.Sub(psBroadcast)       //publish 发送通道, 订阅内部广播消息

	psTopics := []string{psTxTopic, psBatchTxTopic, psBlockTopic, psLtBlockTopic, p.peerTopic}

	for _, topic := range psTopics {
		// pub sub topic注册
		err := p.Pubsub.JoinAndSubTopic(topic, p.callback(incoming))
		if err != nil {
			log.Error("pubsub init", "topic", topic, "join topic err", err)
			return
		}
	}

	if !p.cfg.DisableValidation {
		p.val = initValidator(p)
		p.Pubsub.RegisterTopicValidator(psBlockTopic, p.val.validateBlock, pubsub.WithValidatorInline(true))
		p.Pubsub.RegisterTopicValidator(psTxTopic, p.val.validateTx, pubsub.WithValidatorInline(true))
		p.Pubsub.RegisterTopicValidator(psLtBlockTopic, p.val.validatePeer, pubsub.WithValidatorInline(true))
		p.Pubsub.RegisterTopicValidator(psBatchTxTopic, p.val.validateBatchTx, pubsub.WithValidatorInline(true))
	}

	//使用多个协程并发处理，提高效率
	concurrency := runtime.NumCPU() * 2
	for i := 0; i < concurrency; i++ {
		go p.handlePubMsg(outgoing)
		go p.handleSubMsg(incoming)
	}
}

// 处理广播消息发布
func (p *pubSub) handlePubMsg(out chan interface{}) {

	defer p.ps.Unsub(out)
	buf := make([]byte, 0)
	var err error
	for {
		select {
		case data, ok := <-out: //发送广播交易
			if !ok {
				return
			}
			psMsg := data.(publishMsg)
			raw := p.encodeMsg(psMsg.msg, &buf)
			if err != nil {
				log.Error("handlePubMsg", "topic", psMsg.topic, "err", err)
				break
			}

			err = p.Pubsub.Publish(psMsg.topic, raw)
			if err != nil {
				log.Error("handlePubMsg", "topic", psMsg.topic, "publish err", err)
			}

		case <-p.Ctx.Done():
			return
		}
	}
}

// 处理广播消息订阅
func (p *pubSub) handleSubMsg(in chan net.SubMsg) {

	buf := make([]byte, 0)
	var err error
	var msg types.Message
	for {
		select {
		case data, ok := <-in: //接收广播交易
			if !ok {
				return
			}
			topic := *data.Topic
			// 交易在pubsub内部验证时已经发送至mempool, 此处直接忽略
			if topic == psTxTopic || topic == psBatchTxTopic {
				break
			}
			msg = p.newMsg(topic)
			err = p.decodeMsg(data.Data, &buf, msg)
			if err != nil {
				log.Error("handleSubMsg", "topic", topic, "decodeMsg err", err)
				break
			}
			p.handleBroadcastReceive(subscribeMsg{
				topic:       topic,
				value:       msg,
				receiveFrom: data.ReceivedFrom,
				publisher:   peer.ID(data.From),
			})

		case <-p.Ctx.Done():
			return
		}
	}

}

// 统一处理哈希计算
func (p *pubSub) getMsgHash(topic string, msg types.Message) string {
	if topic == psTxTopic {
		return hex.EncodeToString(msg.(*types.Transaction).Hash())
	} else if topic == psBlockTopic {
		return hex.EncodeToString(msg.(*types.Block).Hash(p.ChainCfg))
	}
	return ""
}

// 构造接收消息对象s
func (p *pubSub) newMsg(topic string) types.Message {
	switch topic {
	case psTxTopic:
		return &types.Transaction{}
	case psBatchTxTopic:
		return &types.Transactions{}
	case psBlockTopic:
		return &types.Block{}
	case psLtBlockTopic:
		return &types.LightBlock{}
	default:
		return &types.PeerPubSubMsg{}
	}
}

// 生成订阅消息回调
func (p *pubSub) callback(out chan<- net.SubMsg) net.SubCallBack {
	return func(topic string, msg net.SubMsg) {
		out <- msg
	}
}

// 数据压缩后发送， 内部对相关数组进行重复利用
func (p *pubSub) encodeMsg(msg types.Message, pbuf *[]byte) []byte {
	buf := *pbuf
	buf = buf[:cap(buf)]
	raw := types.Encode(msg)
	buf = snappy.Encode(buf, raw)
	*pbuf = buf
	// 复用raw数组作为压缩数据返回， 需要比较容量是否够大
	if cap(raw) >= len(buf) {
		raw = raw[:len(buf)]
	} else {
		raw = make([]byte, len(buf))
	}
	copy(raw, buf)
	return raw
}

// 接收数据并解压缩
func (p *pubSub) decodeMsg(raw []byte, reuseBuf *[]byte, msg types.Message) error {

	var err error
	var buf []byte
	if reuseBuf == nil {
		reuseBuf = &buf
	} else {
		buf = *reuseBuf
	}

	buf = buf[:cap(buf)]
	buf, err = snappy.Decode(buf, raw)
	if err != nil {
		log.Error("pubSub decodeMsg", "snappy decode err", err)
		return errSnappyDecode
	}
	//重复利用解码buf
	*reuseBuf = buf
	err = types.Decode(buf, msg)
	if err != nil {
		log.Error("pubSub decodeMsg", "pb decode err", err)
		return types.ErrDecode
	}

	return nil
}
