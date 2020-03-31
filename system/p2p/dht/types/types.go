// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package types 外部公用类型
package types

// P2PSubConfig p2p 子配置
type P2PSubConfig struct {
	// P2P服务监听端口号
	Port int32 `protobuf:"varint,1,opt,name=port" json:"port,omitempty"`
	// 种子节点，格式为ip:port，多个节点以逗号分隔，如seeds=
	Seeds []string `protobuf:"bytes,2,rep,name=seeds" json:"seeds,omitempty"`
	// 是否为种子节点
	IsSeed bool `protobuf:"varint,3,opt,name=isSeed" json:"isSeed,omitempty"`
	//固定连接节点，只连接配置项seeds中的节点
	FixedSeed bool `protobuf:"varint,4,opt,name=fixedSeed" json:"fixedSeed,omitempty"`
	// 是否使用内置的种子节点
	InnerSeedEnable bool `protobuf:"varint,5,opt,name=innerSeedEnable" json:"innerSeedEnable,omitempty"`
	// 是否使用Github获取种子节点
	UseGithub bool `protobuf:"varint,6,opt,name=useGithub" json:"useGithub,omitempty"`
	// 是否作为服务端，对外提供服务
	ServerStart bool `protobuf:"varint,7,opt,name=serverStart" json:"serverStart,omitempty"`
	// 最多的接入节点个数
	InnerBounds int32 `protobuf:"varint,8,opt,name=innerBounds" json:"innerBounds,omitempty"`
	//交易开始采用哈希广播的ttl
	LightTxTTL int32 `protobuf:"varint,9,opt,name=lightTxTTL" json:"lightTxTTL,omitempty"`
	// 最大传播ttl, ttl达到该值将停止继续向外发送
	MaxTTL int32 `protobuf:"varint,10,opt,name=maxTTL" json:"maxTTL,omitempty"`
	// p2p网络频道,用于区分主网/测试网/其他网络
	Channel int32 `protobuf:"varint,11,opt,name=channel" json:"channel,omitempty"`
	//区块轻广播的最低打包交易数, 大于该值时区块内交易采用短哈希广播
	MinLtBlockTxNum int32 `protobuf:"varint,12,opt,name=minLtBlockTxNum" json:"minLtBlockTxNum,omitempty"`
	//指定p2p类型, 支持gossip, dht
	MaxConnnectNum int32 `protobuf:"varint,13,opt,name=maxConnnectNum" json:"maxConnnectNum,omitempty"`
}
