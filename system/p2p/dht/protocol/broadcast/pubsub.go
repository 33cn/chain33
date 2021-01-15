// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"encoding/hex"
	"runtime"
	"time"

	"github.com/33cn/chain33/p2p/utils"
	net "github.com/33cn/chain33/system/p2p/dht/extension"
	"github.com/33cn/chain33/types"
	"github.com/golang/snappy"
)

const (
	psTxTopic    = "tx/v1.0.0"
	psBlockTopic = "block/v1.0.0"
)

// 基于libp2p pubsub插件广播
type pubSub struct {
	*broadcastProtocol
}

// new pub sub
func newPubSub(b *broadcastProtocol) *pubSub {
	return &pubSub{b}
}

// 广播入口函数，处理相关初始化
func (p *pubSub) broadcast() {

	//TODO check net is sync

	txIncoming := make(chan net.SubMsg, 1024) //交易接收通道, 订阅外部广播消息
	txOutgoing := p.ps.Sub(psTxTopic)         //交易发送通道, 订阅内部广播消息
	//区块
	blockIncoming := make(chan net.SubMsg, 128)
	blockOutgoing := p.ps.Sub(psBlockTopic)

	// pub sub topic注册
	err := p.Pubsub.JoinAndSubTopic(psTxTopic, p.callback(txIncoming))
	if err != nil {
		log.Error("pubsub broadcast", "join tx topic err", err)
		return
	}
	err = p.Pubsub.JoinAndSubTopic(psBlockTopic, p.callback(blockIncoming))
	if err != nil {
		log.Error("pubsub broadcast", "join block topic err", err)
		return
	}

	// 不存在订阅topic的节点时，不开启广播，目前只在初始化时做判定
	for len(p.Pubsub.FetchTopicPeers(psTxTopic)) == 0 {
		time.Sleep(time.Second * 2)
		log.Warn("pub sub broadcast", "info", "no peers available")
	}

	//发送和接收用多个函数并发处理，提高效率
	//交易广播, 使用多个协程并发处理，提高效率
	cpu := runtime.NumCPU()
	for i := 0; i < cpu; i++ {
		go p.handlePubMsg(psTxTopic, txOutgoing)
		go p.handleSubMsg(psTxTopic, txIncoming, p.txFilter)
	}

	//区块广播
	go p.handlePubMsg(psBlockTopic, blockOutgoing)
	go p.handleSubMsg(psBlockTopic, blockIncoming, p.blockFilter)
}

// 处理广播消息发布
func (p *pubSub) handlePubMsg(topic string, out chan interface{}) {

	defer p.ps.Unsub(out)
	buf := make([]byte, 0)
	var err error
	for {
		select {
		case data, ok := <-out: //发送广播交易
			if !ok {
				return
			}
			msg := data.(types.Message)
			raw := p.encodeMsg(msg, &buf)
			if err != nil {
				log.Error("handlePubMsg", "topic", topic, "hash", p.getMsgHash(topic, msg), "err", err)
				break
			}

			err = p.Pubsub.Publish(topic, raw)
			if err != nil {
				log.Error("handlePubMsg", "topic", topic, "publish err", err)
			}

		case <-p.Ctx.Done():
			return
		}
	}
}

// 处理广播消息订阅
func (p *pubSub) handleSubMsg(topic string, in chan net.SubMsg, filter *utils.Filterdata) {

	buf := make([]byte, 0)
	var err error
	var msg types.Message
	for {
		select {
		case data, ok := <-in: //接收广播交易
			if !ok {
				return
			}
			msg = p.newMsg(topic)
			err = p.decodeMsg(data.Data, &buf, msg)
			if err != nil {
				log.Error("handleSubMsg", "topic", topic, "decodeMsg err", err)
				break
			}
			hash := p.getMsgHash(topic, msg)
			// 接收重复检测,
			if filter.Contains(hash) {
				break
			}

			//TODO 保存消息源节点，对错误消息需要拉黑处理
			filter.Add(hash, struct{}{})

			// 将接收的交易或区块 转发到内部对应模块
			if topic == psTxTopic {
				err = p.postMempool(hash, msg.(*types.Transaction))
			} else {
				err = p.postBlockChain(hash, data.ReceivedFrom.String(), msg.(*types.Block))
			}

			if err != nil {
				log.Error("handleSubMsg", "topic", topic, "hash", hash, "post msg err", err)
			}

		case <-p.Ctx.Done():
			return
		}
	}

}

// 统一处理哈希计算
func (p *pubSub) getMsgHash(topic string, msg types.Message) string {
	if topic == psTxTopic {
		return hex.EncodeToString(msg.(*types.Transaction).Hash())
	}
	return hex.EncodeToString(msg.(*types.Block).Hash(p.ChainCfg))
}

// 构造接收消息对象s
func (p *pubSub) newMsg(topic string) types.Message {
	if topic == psTxTopic {
		return &types.Transaction{}
	}
	return &types.Block{}
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
func (p *pubSub) decodeMsg(raw []byte, pbuf *[]byte, msg types.Message) error {

	var err error
	buf := *pbuf
	buf = buf[:cap(buf)]
	buf, err = snappy.Decode(buf, raw)
	if err != nil {
		log.Error("pubSub decodeMsg", "snappy decode err", err)
		return errSnappyDecode
	}
	//重复利用解码buf
	*pbuf = buf
	err = types.Decode(buf, msg)
	if err != nil {
		log.Error("pubSub decodeMsg", "pb decode err", err)
		return types.ErrDecode
	}

	return nil
}
