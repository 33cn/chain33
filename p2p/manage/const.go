// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package manage

const (
	// GossipTypeName 底层基于非结构化网络, 采用gossip协议广播
	GossipTypeName = "gossip"
	// DHTTypeName 底层基于libp2p框架, dht结构化网络
	DHTTypeName    = "dht"
	defaultP2PType = GossipTypeName
)
