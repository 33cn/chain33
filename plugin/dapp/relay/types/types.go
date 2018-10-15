package types

//Relay Transaction
type RelayOrderTx struct {
	Operation uint32 `json:"operation"`
	Coin      string `json:"coin"`
	Amount    uint64 `json:"coinamount"`
	Addr      string `json:"coinaddr"`
	CoinWait  uint32 `json:"waitblocks"`
	BtyAmount uint64 `json:"btyamount"`
}

type RelayAcceptTx struct {
	OrderId  string `json:"order_id"`
	CoinAddr string `json:"coinaddr"`
	CoinWait uint32 `json:"waitblocks"`
}

type RelayRevokeTx struct {
	OrderId string `json:"order_id"`
	Target  uint32 `json:"target"`
	Action  uint32 `json:"action"`
}

type RelayVerifyBTCTx struct {
	OrderId     string `json:"order_id"`
	RawTx       string `json:"raw_tx"`
	TxIndex     uint32 `json:"tx_index"`
	MerklBranch string `json:"merkle_branch"`
	BlockHash   string `json:"block_hash"`
}

type RelaySaveBTCHeadTx struct {
	Hash         string `json:"hash"`
	Height       uint64 `json:"height"`
	MerkleRoot   string `json:"merkleRoot"`
	PreviousHash string `json:"previousHash"`
	IsReset      bool   `json:"isReset"`
}
