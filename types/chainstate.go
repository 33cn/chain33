package types

type ChainStatus struct {
	CurrentHeight `json:"currentHeight"`
	MempoolSize   `json:"mempoolSize"`
	MsgQueueSize  `json:"msgQueueSize"`
}
