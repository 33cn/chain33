package types

import "code.aliyun.com/chian33/chain33/common"

type Header struct {
	ParentHash common.Hash `json:"parentHash"`
	TxHash     common.Hash `json:"transactionsRoot"`
	Number     int64       `json:"number"`
	Time       int64       `json:"timestamp"`
}

type Block struct {
	*Header `json:"header"`
	Txs     []Transaction `json:"txs"`
}

type Transaction struct {
	Payload []byte `json:"input"`
	// Signature values
	Sign []byte `json:"sign"`
}
