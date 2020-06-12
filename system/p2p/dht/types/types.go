// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package types 外部公用类型
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
	//区块轻广播的最低打包交易数, 大于该值时区块内交易采用短哈希广播
	MinLtBlockTxNum int32 `protobuf:"varint,12,opt,name=minLtBlockTxNum" json:"minLtBlockTxNum,omitempty"`
	//指定p2p类型, 支持gossip, dht
	MaxConnnectNum  int32 `protobuf:"varint,13,opt,name=maxConnnectNum" json:"maxConnnectNum,omitempty"`
	Relay_Active    bool  `protobuf:"varint,15,opt,name=relay_Active" json:"relay_Active,omitempty"`
	Relay_Hop       bool  `protobuf:"varint,16,opt,name=relay_Hop" json:"relay_Hop,omitempty"`
	Relay_Discovery bool  `protobuf:"varint,17,opt,name=relay_Discovery" json:"relay_Discovery,omitempty"`

	DHTDataDriver string `protobuf:"bytes,10,opt,name=DHTDataDriver" json:"DHTDataDriver,omitempty"`
	DHTDataPath   string `protobuf:"bytes,11,opt,name=DHTDataPath" json:"DHTDataPath,omitempty"`
	DHTDataCache  int32  `protobuf:"varint,12,opt,name=DHTDataCache" json:"DHTDataCache,omitempty"`
}
