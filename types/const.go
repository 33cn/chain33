package types

const (
	EventTx = 1

	EventGetBlocks = 2
	EventBlocks    = 3

	EventGetBlockHeight   = 4
	EventReplyBlockHeight = 5

	EventQueryTx     = 6
	EventMerkleProof = 7

	EventReply = 8

	EventTxBroadcast  = 9
	EventTxAddMempool = 10

	EventTxList      = 11
	EventTxListReply = 12

	EventAddBlock       = 13
	EventBlockBroadcast = 14

	EventFetchBlocks = 15
	EventAddBlocks   = 16

	EventTxHashList      = 17
	EventTxHashListReply = 18

	EventGetHeaders = 19
	EventHeaders    = 20

	EventGetMempoolSize = 21
	EventMempoolSize    = 22

	EventStoreGet      = 23
	EventStoreSet      = 24
	EventStoreGetReply = 25
	EventStoreSetReply = 26
)

var eventname = map[int]string{
	1:  "EventTx",
	2:  "EventGetBlocks",
	3:  "EventBlocks",
	4:  "EventGetBlockHeight",
	5:  "EventReplyBlockHeight",
	6:  "EventQueryTx",
	7:  "EventMerkleProof",
	8:  "EventReply",
	9:  "EventTxBroadcast",
	10: "EventTxAddMempool",
	11: "EventTxList",
	12: "EventTxListReply",
	13: "EventAddBlock",
	14: "EventBlockBroadcast",
	15: "EventFetchBlocks",
	16: "EventAddBlocks",
	17: "EventTxHashList",
	18: "EventTxHashListReply",
	19: "EventGetHeaders",
	20: "EventHeaders",
	21: "EventGetMempoolSize",
	22: "EventMempoolSize",
	23: "EventStoreGet",
	24: "EventStoreSet",
	25: "EventStoreGetReply",
	26: "EventStoreSetReply",
}

func GetEventName(event int) string {
	name, ok := eventname[event]
	if ok {
		return name
	}
	return "unknow-event"
}
