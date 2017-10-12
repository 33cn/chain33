package types

const (
	EventTx = iota + 1

	EventGetBlocks
	EventBlocks

	EventGetBlockHeight
	EventReplyBlockHeight

	EventQueryTx
	EventMerkleProof

	EventReply

	EventTxBroadcast
	EventTxAddMempool

	EventTxList
	EventTxListReply

	EventAddBlock
	EventBlockBroadcast

	EventFetchBlocks
	EventAddBlocks
)
