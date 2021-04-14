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
	// 分片数据备份节点数
	Backup int `protobuf:"varint,14,opt,name=backup" json:"backup,omitempty"`
	//是否开启全节点模式
	IsFullNode bool `protobuf:"varint,15,opt,name=isFullNode" json:"isFullNode,omitempty"`
	//老版本最大广播节点数
	MaxBroadcastPeers int `protobuf:"varint,16,opt,name=maxBroadcastPeers" json:"maxBroadcastPeers,omitempty"`
	//pub sub消息是否需要签名和验签
	DisablePubSubMsgSign bool `protobuf:"varint,17,opt,name=disablePubSubMsgSign" json:"disablePubSubMsgSign,omitempty"`
	//是否启用中继功能，如果自己身NAT后面的节点，RelayEnable=true，则仍有可能被其他节点连接。
	RelayEnable bool `protobuf:"varint,18,opt,name=relayEnable" json:"relayEnable,omitempty"`
	//指定中继节点作为
	RelayNodeAddr []string `protobuf:"varint,19,opt,name=relayNodeAddr" json:"relayNodeAddr,omitempty"`
	// 不启动分片功能，默认启动
	DisableShard bool `protobuf:"varint,120,opt,name=disableShard" json:"disableShard,omitempty"`
	//特定场景下的p2p白名单，只连接配置的节点,联盟链使用
	WhitePeerList []string `protobuf:"bytes,21,rep,name=whitePeerList" json:"whitePeerList,omitempty"`
}
