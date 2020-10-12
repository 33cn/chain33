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
	/*
					比如 中继服务器R的地址：/ip4/116.63.171.186/tcp/13801/p2p/16Uiu2HAm1Qgogf1vHj7WV7d8YmVNPw6PzRuYEnJbQsMDmDWtLoEM
					NAT 后面的一个普通节点A是：/ip4/192.168.1.101/tcp/13801/p2p/16Uiu2HAkw6w2YVenbCLAXHv2XVo1kh945GVxQrpm5Y6z2kE3eFSg
				    第一步：A--->R  A连接到中继节点R
					第二步：A开始组装自己的中继地址：
				[/ip4/116.63.171.186/tcp/13801/p2p/16Uiu2HAm1Qgogf1vHj7WV7d8YmVNPw6PzRuYEnJbQsMDmDWtLoEM/p2p-circuit/ip4/192.168.1.101/tcp
			    /13801/p2p/16Uiu2HAkw6w2YVenbCLAXHv2XVo1kh945GVxQrpm5Y6z2kE3eFSg]
			        第三步：A广播这个拼接后的带有p2p-circuit的地址
			        第四步：网络中的节点不论是NAT前面的或者NAT后面的节点，如果想连接节点PID为16Uiu2HAkw6w2YVenbCLAXHv2XVo1kh945GVxQrpm5Y6z2kE3eFSg的A节点，
		                   只需要通过上述组装的带有p2p-circuit的地址就可以建立到与A的连接

	*/
	RelayHop            bool   `protobuf:"varint,10,opt,name=relayHop" json:"relayHop,omitempty"`
	DisableFindLANPeers bool   `protobuf:"varint,11,opt,name=disableFindLANPeers" json:"disableFindLANPeers,omitempty"`
	DHTDataDriver       string `protobuf:"bytes,12,opt,name=DHTDataDriver" json:"DHTDataDriver,omitempty"`
	DHTDataPath         string `protobuf:"bytes,13,opt,name=DHTDataPath" json:"DHTDataPath,omitempty"`
	DHTDataCache        int32  `protobuf:"varint,14,opt,name=DHTDataCache" json:"DHTDataCache,omitempty"`
	//是否开启全节点模式
	IsFullNode bool `protobuf:"varint,15,opt,name=isFullNode" json:"isFullNode,omitempty"`
	//老版本最大广播节点数
	MaxBroadcastPeers int `protobuf:"varint,16,opt,name=maxBroadcastPeers" json:"maxBroadcastPeers,omitempty"`
	//pub sub消息是否需要签名和验签
	DisablePubSubMsgSign bool `protobuf:"varint,17,opt,name=disablePubSubMsgSign" json:"disablePubSubMsgSign,omitempty"`
	//是否启用中继功能
	RelayEnable bool `protobuf:"varint,18,opt,name=relayEnable" json:"relayEnable,omitempty"`
	//指定中继节点作为
	RelayNodeAddr string `protobuf:"varint,19,opt,name=relayNodeAddr" json:"relayNodeAddr,omitempty"`
}
