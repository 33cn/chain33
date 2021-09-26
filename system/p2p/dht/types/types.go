// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types 外部公用类型
package types

// P2PSubConfig p2p 子配置
type P2PSubConfig struct {
	// P2P服务监听端口号
	Port int32 `protobuf:"varint,1,opt,name=port" json:"port,omitempty"`
	// 手动配置节点
	Seeds []string `protobuf:"bytes,2,rep,name=seeds" json:"seeds,omitempty"`
	//交易开始采用哈希广播的ttl
	LightTxTTL int32 `protobuf:"varint,3,opt,name=lightTxTTL" json:"lightTxTTL,omitempty"`
	// 最大传播ttl, ttl达到该值将停止继续向外发送
	MaxTTL int32 `protobuf:"varint,4,opt,name=maxTTL" json:"maxTTL,omitempty"`
	// p2p网络频道,用于区分主网/测试网/其他网络

	Channel int32 `protobuf:"varint,5,opt,name=channel" json:"channel,omitempty"`
	//区块轻广播的最低区块大小,单位KB, 大于该值时区块内交易采用短哈希广播
	MinLtBlockSize int32 `protobuf:"varint,6,opt,name=minLtBlockSize" json:"minLtBlockSize,omitempty"`
	//最大dht连接数
	MaxConnectNum int32 `protobuf:"varint,7,opt,name=maxConnectNum" json:"maxConnectNum,omitempty"`
	//引导节点配置
	BootStraps []string `protobuf:"bytes,8,rep,name=bootStraps" json:"bootStraps,omitempty"`
	//轻广播本地区块缓存大小, 单位M
	LtBlockCacheSize int32 `protobuf:"varint,9,opt,name=ltBlockCacheSize" json:"ltBlockCacheSize,omitempty"`

	//中继传输主动建立连接，中继服务端可以选配，启动中继服务之后，NAT后面的节点可以通过中继节点广播自己的节点地址信息

	RelayHop            bool   `protobuf:"varint,10,opt,name=relayHop" json:"relayHop,omitempty"`
	DisableFindLANPeers bool   `protobuf:"varint,11,opt,name=disableFindLANPeers" json:"disableFindLANPeers,omitempty"`
	DHTDataPath         string `protobuf:"bytes,12,opt,name=DHTDataPath" json:"DHTDataPath,omitempty"`
	DHTDataCache        int32  `protobuf:"varint,13,opt,name=DHTDataCache" json:"DHTDataCache,omitempty"`
	// 分片数据保存比例
	Percentage int `protobuf:"varint,14,opt,name=percentage" json:"percentage,omitempty"`
	//是否开启全节点模式
	IsFullNode bool `protobuf:"varint,15,opt,name=isFullNode" json:"isFullNode,omitempty"`
	//老版本最大广播节点数
	MaxBroadcastPeers int `protobuf:"varint,16,opt,name=maxBroadcastPeers" json:"maxBroadcastPeers,omitempty"`
	//是否启用中继功能，如果自己身NAT后面的节点，RelayEnable=true，则仍有可能被其他节点连接。
	RelayEnable bool `protobuf:"varint,18,opt,name=relayEnable" json:"relayEnable,omitempty"`
	//指定中继节点作为
	RelayNodeAddr []string `protobuf:"varint,19,opt,name=relayNodeAddr" json:"relayNodeAddr,omitempty"`
	// 不启动分片功能，默认启动
	DisableShard bool `protobuf:"varint,120,opt,name=disableShard" json:"disableShard,omitempty"`
	//特定场景下的p2p白名单，只连接配置的节点,联盟链使用
	WhitePeerList []string `protobuf:"bytes,21,rep,name=whitePeerList" json:"whitePeerList,omitempty"`
	// 扩展路由表最小节点数
	MinExtendRoutingTableSize int `protobuf:"varint,22,opt,name=minExtendRoutingTableSize" json:"minExtendRoutingTableSize,omitempty"`
	// 扩展路由表最大节点数
	MaxExtendRoutingTableSize int `protobuf:"varint,23,opt,name=n=maxExtendRoutingTableSize" json:"maxExtendRoutingTableSize,omitempty"`

	//配置libp2p库代码日志等级， 支持DEBUG, INFO, WARN, ERROR，默认ERROR
	Libp2pLogLevel string `json:"libp2pLogLevel,omitempty"`

	// pubsub配置
	PubSub PubSubConfig `json:"pubsub,omitempty"`
	//启动私有网络，只有相同配置的节点才能连接，多用于联盟链需求，创建方式 hex.Encode([32]byte),32字节的十六进制编码字符串
	Psk string `json:"psk"`
	Trace traceConfig `json:"trace,omitempty"`

}

//授权给某个IP远程访问本节点相关数据，授权的接口功能为只读模式
type traceConfig struct{
	ListenAddr string `json:"listenAddr,omitempty"`
}

// PubSubConfig pubsub config
type PubSubConfig struct {

	// pub sub消息是否需要签名和验签
	DisablePubSubMsgSign bool `json:"disablePubSubMsgSign,omitempty"`

	// set the buffer size for outbound messages to a peer, Defaults to 128
	// We start dropping messages to a peer if the outbound queue if full
	PeerOutboundQueueSize int `json:"peerOutboundQueueSize,omitempty"`

	// sets the buffer of validate queue. Defaults to 128.
	// When queue is full, validation is throttled and new messages are dropped.
	ValidateQueueSize int `json:"validateQueueSize,omitempty"`

	// This should generally be enabled in bootstrappers and
	// well connected/trusted nodes used for bootstrapping.
	EnablePeerExchange bool `json:"enablePeerExchange,omitempty"`

	// GossipSubDlo sets the lower bound on the number of peers we keep in a GossipSub topic mesh.
	// If we have fewer than GossipSubDlo peers, we will attempt to graft some more into the mesh at
	// the next heartbeat.
	GossipSubDlo int `json:"gossipSubDlo,omitempty"`

	// GossipSubD sets the optimal degree for a GossipSub topic mesh. For example, if GossipSubD == 6,
	// each peer will want to have about six peers in their mesh for each topic they're subscribed to.
	// GossipSubD should be set somewhere between GossipSubDlo and GossipSubDhi.
	GossipSubD int `json:"gossipSubD,omitempty"`

	// GossipSubDhi sets the upper bound on the number of peers we keep in a GossipSub topic mesh.
	// If we have more than GossipSubDhi peers, we will select some to prune from the mesh at the next heartbeat.
	GossipSubDhi int `json:"gossipSubDhi,omitempty"`

	// GossipSubHeartbeatInterval controls the time between heartbeats. ms
	GossipSubHeartbeatInterval int `json:"gossipSubHeartbeatInterval,omitempty"`

	// GossipSubHistoryGossip controls how many cached message ids we will advertise in
	// IHAVE gossip messages. When asked for our seen message IDs, we will return
	// only those from the most recent GossipSubHistoryGossip heartbeats. The slack between
	// GossipSubHistoryGossip and GossipSubHistoryLength allows us to avoid advertising messages
	// that will be expired by the time they're requested.
	//
	// GossipSubHistoryGossip must be less than or equal to GossipSubHistoryLength to
	// avoid a runtime panic.
	GossipSubHistoryGossip int `json:"gossipSubHistoryGossip,omitempty"`

	// GossipSubHistoryLength controls the size of the message cache used for gossip.
	// The message cache will remember messages for GossipSubHistoryLength heartbeats.
	GossipSubHistoryLength int `json:"gossipSubHistoryLength,omitempty"`
}
