package types

import "errors"

var ErrNotFound = errors.New("ErrNotFound")
var ErrNoBalance = errors.New("ErrNoBalance")

const (
	EventTx = 1

	EventGetBlocks = 2
	EventBlocks    = 3

	EventGetBlockHeight   = 4
	EventReplyBlockHeight = 5

	EventQueryTx     = 6
	EventMerkleProof = 7

	EventReply = 8

	EventTxBroadcast = 9
	EventPeerInfo    = 10

	EventTxList      = 11
	EventReplyTxList = 12

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
	EventReceipts      = 27
	EventExecTxList    = 28
	EventPeerList      = 29
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
	10: "EventPeerInfo",
	11: "EventTxList",
	12: "EventReplyTxList",
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
	27: "EventReceipts",
	28: "EventExecTxList",
	29: "EventPeerList",
}

func GetEventName(event int) string {
	name, ok := eventname[event]
	if ok {
		return name
	}
	return "unknow-event"
}

//ty = 1 -> secp256k1
//ty = 2 -> ed25519
//ty = 3 -> sm2
const (
	SECP256K1 = 1
	ED25519   = 2
	SM2       = 3
)

func GetSignatureTypeName(signType int) string {
	if signType == 1 {
		return "secp256k1"
	} else if signType == 2 {
		return "ed25519"
	} else if signType == 3 {
		return "sm2"
	} else {
		return "unknow"
	}
}

//log type
const (
	TyLogErr = 1
	TyLogFee = 2
)

//exec type

const (
	ExecErr  = 0
	ExecPack = 1
	ExecOk   = 2
)
