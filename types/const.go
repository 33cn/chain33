package types

import "errors"

var ErrNotFound = errors.New("ErrNotFound")
var ErrNoBalance = errors.New("ErrNoBalance")
var ErrBlockExec = errors.New("ErrBlockExec")
var ErrCheckStateHash = errors.New("ErrCheckStateHash")
var ErrCheckTxHash = errors.New("ErrCheckTxHash")
var ErrReRunGenesis = errors.New("ErrReRunGenesis")
var ErrActionNotSupport = errors.New("ErrActionNotSupport")
var ErrChannelFull = errors.New("ErrChannelFull")

const Coin int64 = 1e8

const (
	EventTx = 1

	EventGetBlocks = 2
	EventBlocks    = 3

	EventGetBlockHeight   = 4
	EventReplyBlockHeight = 5

	EventQueryTx           = 6
	EventTransactionDetail = 7

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

	EventStoreGet             = 23
	EventStoreSet             = 24
	EventStoreGetReply        = 25
	EventStoreSetReply        = 26
	EventReceipts             = 27
	EventExecTxList           = 28
	EventPeerList             = 29
	EventGetLastHeader        = 30
	EventHeader               = 31
	EventAddBlockDetail       = 32
	EventGetMempool           = 33
	EventGetTransactionByAddr = 34
	EventGetTransactionByHash = 35
	EventReplyTxInfo          = 36
	//wallet event
	EventWalletGetAccountList  = 37
	EventWalletAccountList     = 38
	EventNewAccount            = 39
	EventWalletAccount         = 40
	EventWalletTransactionList = 41
	//EventReplyTxList           = 42
	EventWalletImportprivkey = 43
	EventWalletSendToAddress = 44
	EventWalletSetFee        = 45
	EventWalletSetLabel      = 46
	//EventWalletAccount       = 47
	EventWalletMergeBalance = 48
	EventReplyHashes        = 49
	EventWalletSetPasswd    = 50
	EventWalletLock         = 51
	EventWalletUnLock       = 52
	EventTransactionDetails = 53
	EventBroadcastAddBlock  = 54
	EventGetBlockOverview   = 55
	EventGetAddrOverview    = 56
	EventReplyBlockOverview = 57
	EventReplyAddrOverview  = 58
)

var eventname = map[int]string{
	1:  "EventTx",
	2:  "EventGetBlocks",
	3:  "EventBlocks",
	4:  "EventGetBlockHeight",
	5:  "EventReplyBlockHeight",
	6:  "EventQueryTx",
	7:  "EventTransactionDetail",
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
	30: "EventGetLastHeader",
	31: "EventHeader",
	32: "EventAddBlockDetail",
	33: "EventGetMempool",
	34: "EventGetTransactionByAddr",
	35: "EventGetTransactionByHash",
	36: "EventReplyTxInfo",
	37: "EventWalletGetAccountList",
	38: "EventWalletAccountList",
	39: "EventNewAccount",
	40: "EventWalletAccount",
	41: "EventWalletTransactionList",
	//42: "EventReplyTxList",
	43: "EventWalletImportPrivKey",
	44: "EventWalletSendToAddress",
	45: "EventWalletSetFee",
	46: "EventWalletSetLabel",
	//47: "EventWalletAccount",
	48: "EventWalletMergeBalance",
	49: "EventReplyHashes",
	50: "EventWalletSetPasswd",
	51: "EventWalletLock",
	52: "EventWalletUnLock",
	53: "EventTransactionDetails",
	54: "EventBroadcastAddBlock",
	55: "EventGetBlockOverview",
	56: "EventGetAddrOverview",
	57: "EventReplyBlockOverview",
	58: "EventReplyAddrOverview",
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
	TyLogErr      = 1
	TyLogFee      = 2
	TyLogTransfer = 3
	TyLogGenesis  = 4
)

//exec type
const (
	ExecErr  = 0
	ExecPack = 1
	ExecOk   = 2
)

//coinsaction
const (
	CoinsActionTransfer = 1
	CoinsActionGenesis  = 2
)
