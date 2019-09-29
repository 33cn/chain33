// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"time"
)

// time limit for timeout
var (
	UpdateState                 = 2 * time.Second
	PingTimeout                 = 14 * time.Second
	DefaultSendTimeout          = 10 * time.Second
	DialTimeout                 = 5 * time.Second
	mapUpdateInterval           = 45 * time.Hour
	StreamPingTimeout           = 20 * time.Second
	MonitorPeerInfoInterval     = 10 * time.Second
	MonitorPeerNumInterval      = 30 * time.Second
	MonitorReBalanceInterval    = 15 * time.Minute
	GetAddrFromAddrBookInterval = 5 * time.Second
	GetAddrFromOnlineInterval   = 5 * time.Second
	GetAddrFromGitHubInterval   = 5 * time.Minute
	CheckActivePeersInterVal    = 5 * time.Second
	CheckBlackListInterVal      = 30 * time.Second
	CheckCfgSeedsInterVal       = 1 * time.Minute
)

const (
	msgTx           = 1
	msgBlock        = 2
	tryMapPortTimes = 20
	maxSamIPNum     = 20
)

const (
	//defalutNatPort  = 23802
	maxOutBoundNum  = 25
	stableBoundNum  = 15
	maxAttemps      = 5
	protocol        = "tcp"
	externalPortTag = "externalport"
)

const (
	nodeNetwork = 1
	nodeGetUTXO = 2
	nodeBloom   = 4
)

const (
	// Service service number
	Service int32 = nodeBloom + nodeNetwork + nodeGetUTXO
)

// leveldb 中p2p privkey,addrkey
const (
	addrkeyTag = "addrs"
	privKeyTag = "privkey"
)

//TTL
const (
	DefaultLtTxBroadCastTTL  = 3
	DefaultMaxTxBroadCastTTL = 25
)

// P2pCacheTxSize p2pcache size of transaction
const (
	PeerAddrCacheNum = 1000
	//接收的交易哈希过滤缓存设为mempool最大接收交易量
	TxRecvFilterCacheNum = 10240
	BlockFilterCacheNum  = 50
	//发送过滤主要用于发送时冗余检测, 发送完即可以被删除, 维护较小缓存数
	TxSendFilterCacheNum  = 500
	BlockCacheNum         = 10
	MaxBlockCacheByteSize = 100 * 1024 * 1024
)

// TestNetSeeds test seeds of net
var TestNetSeeds = []string{
	"47.97.223.101:13802",
}

// MainNetSeeds built-in list of seed
var MainNetSeeds = []string{
	"116.62.14.25:13802",
	"114.55.95.234:13802",
	"115.28.184.14:13802",
	"39.106.166.159:13802",
	"39.106.193.172:13802",
	"47.106.114.93:13802",
	"120.76.100.165:13802",
	"120.24.85.66:13802",
	"120.24.92.123:13802",
	"161.117.7.127:13802",
	"161.117.9.54:13802",
	"161.117.5.95:13802",
	"161.117.7.28:13802",
	"161.117.8.242:13802",
	"161.117.6.193:13802",
	"161.117.8.158:13802",
	"47.88.157.209:13802",
	"47.74.215.41:13802",
	"47.74.128.69:13802",
	"47.74.178.226:13802",
	"47.88.154.76:13802",
	"47.74.151.226:13802",
	"47.245.31.41:13802",
	"47.245.57.239:13802",
	"47.245.54.118:13802",
	"47.245.54.121:13802",
	"47.245.56.140:13802",
	"47.245.52.211:13802",
	"47.91.88.195:13802",
	"47.91.72.71:13802",
	"47.91.91.38:13802",
	"47.91.94.224:13802",
	"47.91.75.191:13802",
	"47.254.152.172:13802",
	"47.252.0.181:13802",
	"47.90.246.246:13802",
	"47.90.208.100:13802",
	"47.89.182.70:13802",
	"47.90.207.173:13802",
	"47.89.188.54:13802",
}
