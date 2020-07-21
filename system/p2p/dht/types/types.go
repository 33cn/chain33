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

	//中继传输主动建立连接，中继服务端可以选配
	/*

		    a config RelayDiscovery
		    b config RelayHop,RelayActive
		    c config nothing
			a->b a connect b
			a.DialPeer(b,c) will success
			if b just config RelayHop,a->b a connect b
			a.DialPeer(b,c) will failed
		    if  b just config RelayHop,a->b,b->c
			a.DialPeer(b,c) will success
	*/
	RelayActive bool `protobuf:"varint,10,opt,name=relayActive" json:"relayActive,omitempty"`
	//接受其他节点发过来的中继请求，中继服务端必须配置
	RelayHop bool `protobuf:"varint,11,opt,name=relayHop" json:"relayHop,omitempty"`
	//发现新的中继节点，中继客户端端必须配置
	RelayDiscovery      bool `protobuf:"varint,12,opt,name=relayDiscovery" json:"relayDiscovery,omitempty"`
	DisableFindLANPeers bool `protobuf:"varint,13,opt,name=disableFindLANPeers" json:"disableFindLANPeers,omitempty"`

	DHTDataDriver string `protobuf:"bytes,14,opt,name=DHTDataDriver" json:"DHTDataDriver,omitempty"`
	DHTDataPath   string `protobuf:"bytes,15,opt,name=DHTDataPath" json:"DHTDataPath,omitempty"`
	DHTDataCache  int32  `protobuf:"varint,16,opt,name=DHTDataCache" json:"DHTDataCache,omitempty"`

	//是否开启全节点模式
	IsFullNode bool `protobuf:"varint,17,opt,name=isFullNode" json:"isFullNode,omitempty"`
}
