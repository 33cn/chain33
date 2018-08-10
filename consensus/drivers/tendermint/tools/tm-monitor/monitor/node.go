package monitor

import (
	"fmt"
	"time"

	"github.com/tendermint/tendermint/libs/log"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
)

const (
	BLOCKTIME_MAP_SIZE = 128
)

type MemStatus struct {
	All  uint64 `json:"all"`
	Used uint64 `json:"used"`
	Free uint64 `json:"free"`
	Self uint64 `json:"self"`
}

type Node struct {
	rpcAddr string

	//IsValidator bool          `json:"is_validator"` // validator or non-validator?
	//pubKey      crypto.PubKey `json:"pub_key"`

	Name          string `json:"name"`
	Online        bool   `json:"online"`
	Height        int64  `json:"height"`
	BlockTime     int64  `json:"blocktime"`
	LastBlockTime int64  `json:"lastblocktime"`
	StateHash     string `json:"stateHash"`
	TxCount       int64  `json:"txcount"`
	BlockLatency  int64  `json:"block_latency"`

	// 最近一段时间的区块打包时间
	CurrentBlockTime [BLOCKTIME_MAP_SIZE]int64
	index            int64

	// rpcClient is an client for making RPC calls to TM
	rpcClient *jsonrpc.JSONClient

	//blockCh        chan<- tmtypes.Header
	//blockLatencyCh chan<- float64
	//disconnectCh   chan<- bool

	checkIsValidatorInterval time.Duration

	MemInfo *MemStatus

	quit chan struct{}

	logger log.Logger
}

func NewNode(rpcAddr string, options ...func(*Node)) *Node {
	rpcClient, err := jsonrpc.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Println("New JsonClient", "err", err)
	}
	n := &Node{
		rpcAddr:                  rpcAddr,
		rpcClient:                rpcClient,
		Name:                     rpcAddr,
		Online:                   false,
		BlockTime:                0,
		StateHash:                "",
		TxCount:                  0,
		quit:                     make(chan struct{}),
		checkIsValidatorInterval: 5 * time.Second,
		MemInfo:                  &MemStatus{},
		logger:                   log.NewNopLogger(),
		CurrentBlockTime:         [BLOCKTIME_MAP_SIZE]int64{},
		index:                    0,
	}

	for _, option := range options {
		option(n)
	}

	return n
}

//func (n *Node) GetNodeInfo() string {
//	var res jsonrpc.Header
//	err := n.rpcClient.Call("Chain33.PeerInfo", nil, res)
//	if err != nil {
//		n.logger.Error("Get peer info failed...","err", err )
//		return ""
//	}
//
//	var result interface{}
//	data, err := json.MarshalIndent(result, "", "    ")
//	if err != nil {
//		n.logger.Error("MarshalInednt failed...","err", err )
//		return ""
//	}
//
//	return string(data)
//}

func (n *Node) Start() error {
	//info := n.GetNodeInfo()
	//fmt.Println(info)
	return nil
}

func (n *Node) Stop() {
	n.Online = false

	close(n.quit)
}

func (n *Node) SetLogger(l log.Logger) {
	n.logger = l
}

func (n *Node) UpdateBlockTimeArr(bt int64) {
	if n.index == BLOCKTIME_MAP_SIZE {
		n.index = 0
	}
	n.CurrentBlockTime[n.index] = bt
	n.index++
}

func (n *Node) GetAverageBlockTime() int64 {
	var sum, res int64
	for i := 0; i < BLOCKTIME_MAP_SIZE; i++ {
		if n.CurrentBlockTime[i] != 0 {
			sum += n.CurrentBlockTime[i]
		}
	}

	// 如果最后一个位置不为0, 说明数组已经满了。直接除以数组长度即可
	if n.CurrentBlockTime[BLOCKTIME_MAP_SIZE-1] != 0 {
		res = sum / BLOCKTIME_MAP_SIZE
	} else {
		res = sum / (n.index + 1)
	}
	return res
}
