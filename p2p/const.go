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

var (
	// LocalAddr local address
	LocalAddr string
	//defaultPort = 13802
)

const (
	defalutNatPort  = 23802
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

// P2pCacheTxSize p2pcache size of transaction
const (
	P2pCacheTxSize = 10240
)

// TestNetSeeds test seeds of net
var TestNetSeeds = []string{
	"114.55.101.159:13802",
	"47.104.125.151:13802",
	"47.104.125.97:13802",
	"47.104.125.177:13802",
}

// MainNetSeeds built-in list of seed
var MainNetSeeds = []string{
	"39.107.234.240:13802",
	"39.105.88.66:13802",
	"39.105.87.114:13802",
	"39.105.85.247:13802",
	"39.105.87.106:13802",
	"39.105.76.78:13802",
	"39.105.82.4:13802",
	"39.105.43.225:13802",
	"39.107.239.248:13802",
	"39.105.83.33:13802",
	"120.27.234.254:13802",
	"116.62.169.41:13802",
	"47.97.169.229:13802",
	"47.98.253.181:13802",
	"47.98.252.73:13802",
	"47.98.253.127:13802",
	"47.98.251.119:13802",
	"120.27.230.87:13802",
	"47.98.59.24:13802",
	"47.98.247.98:13802",
	"39.108.133.129:13802",
	"120.79.150.175:13802",
	"39.108.97.52:13802",
	"39.108.208.73:13802",
	"120.78.154.251:13802",
	"120.79.134.73:13802",
	"120.79.174.247:13802",
	"120.79.156.149:13802",
	"120.78.135.23:13802",
	"120.79.21.219:13802",
	"47.74.248.233:13802",
	"47.88.168.235:13802",
	"47.74.229.169:13802",
	"47.74.250.4:13802",
	"47.74.157.48:13802",
	"47.252.6.16:13802",
	"47.90.247.202:13802",
	"47.252.9.86:13802",
	"47.252.13.153:13802",
	"47.252.13.228:13802",
	"47.254.129.47:13802",
	"47.254.132.237:13802",
	"47.254.150.129:13802",
	"47.254.149.155:13802",
	"47.254.145.252:13802",
	"47.74.8.101:13802",
	"47.91.19.21:13802",
	"47.74.22.60:13802",
	"47.74.22.86:13802",
	"47.91.17.139:13802",
}
