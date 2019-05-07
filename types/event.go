// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

// event
const (
	EventTx                    = 1
	EventGetBlocks             = 2
	EventBlocks                = 3
	EventGetBlockHeight        = 4
	EventReplyBlockHeight      = 5
	EventQueryTx               = 6
	EventTransactionDetail     = 7
	EventReply                 = 8
	EventTxBroadcast           = 9
	EventPeerInfo              = 10
	EventTxList                = 11
	EventReplyTxList           = 12
	EventAddBlock              = 13
	EventBlockBroadcast        = 14
	EventFetchBlocks           = 15
	EventAddBlocks             = 16
	EventTxHashList            = 17
	EventTxHashListReply       = 18
	EventGetHeaders            = 19
	EventHeaders               = 20
	EventGetMempoolSize        = 21
	EventMempoolSize           = 22
	EventStoreGet              = 23
	EventStoreSet              = 24
	EventStoreGetReply         = 25
	EventStoreSetReply         = 26
	EventReceipts              = 27
	EventExecTxList            = 28
	EventPeerList              = 29
	EventGetLastHeader         = 30
	EventHeader                = 31
	EventAddBlockDetail        = 32
	EventGetMempool            = 33
	EventGetTransactionByAddr  = 34
	EventGetTransactionByHash  = 35
	EventReplyTxInfo           = 36
	EventWalletGetAccountList  = 37
	EventWalletAccountList     = 38
	EventNewAccount            = 39
	EventWalletAccount         = 40
	EventWalletTransactionList = 41
	EventWalletExecutor        = 42
	EventWalletImportPrivkey   = 43
	EventWalletSendToAddress   = 44
	EventWalletSetFee          = 45
	EventWalletSetLabel        = 46
	EventStoreDel              = 47
	EventWalletMergeBalance    = 48
	EventReplyHashes           = 49
	EventWalletSetPasswd       = 50
	EventWalletLock            = 51
	EventWalletUnLock          = 52
	EventTransactionDetails    = 53
	EventBroadcastAddBlock     = 54
	EventGetBlockOverview      = 55
	EventGetAddrOverview       = 56
	EventReplyBlockOverview    = 57
	EventReplyAddrOverview     = 58
	EventGetBlockHash          = 59
	EventBlockHash             = 60
	EventGetLastMempool        = 61
	EventMinerStart            = 63
	EventMinerStop             = 64
	EventWalletTickets         = 65
	EventStoreMemSet           = 66
	EventStoreRollback         = 67
	EventStoreCommit           = 68
	EventCheckBlock            = 69
	//seed
	EventGenSeed      = 70
	EventReplyGenSeed = 71
	EventSaveSeed     = 72
	EventGetSeed      = 73
	EventReplyGetSeed = 74
	EventDelBlock     = 75
	//local store
	EventLocalGet        = 76
	EventLocalReplyValue = 77
	EventLocalList       = 78
	EventLocalSet        = 79

	EventGetWalletStatus         = 80
	EventCheckTx                 = 81
	EventReceiptCheckTx          = 82
	EventReplyQuery              = 84
	EventAddBlockSeqCB           = 85
	EventFetchBlockHeaders       = 86
	EventAddBlockHeaders         = 87
	EventReplyWalletStatus       = 89
	EventGetLastBlock            = 90
	EventBlock                   = 91
	EventGetTicketCount          = 92
	EventReplyGetTicketCount     = 93
	EventDumpPrivkey             = 94
	EventReplyPrivkey            = 95
	EventIsSync                  = 96
	EventReplyIsSync             = 97
	EventCloseTickets            = 98
	EventGetAddrTxs              = 99
	EventReplyAddrTxs            = 100
	EventIsNtpClockSync          = 101
	EventReplyIsNtpClockSync     = 102
	EventDelTxList               = 103
	EventStoreGetTotalCoins      = 104
	EventGetTotalCoinsReply      = 105
	EventQueryTotalFee           = 106
	EventSignRawTx               = 107
	EventReplySignRawTx          = 108
	EventSyncBlock               = 109
	EventGetNetInfo              = 110
	EventReplyNetInfo            = 111
	EventErrToFront              = 112
	EventFatalFailure            = 113
	EventReplyFatalFailure       = 114
	EventBindMiner               = 115
	EventReplyBindMiner          = 116
	EventDecodeRawTx             = 117
	EventReplyDecodeRawTx        = 118
	EventGetLastBlockSequence    = 119
	EventReplyLastBlockSequence  = 120
	EventGetBlockSequences       = 121
	EventReplyBlockSequences     = 122
	EventGetBlockByHashes        = 123
	EventReplyBlockDetailsBySeqs = 124
	EventDelParaChainBlockDetail = 125
	EventAddParaChainBlockDetail = 126
	EventGetSeqByHash            = 127
	EventLocalPrefixCount        = 128
	EventWalletCreateTx          = 129
	EventStoreList               = 130
	EventStoreListReply          = 131
	EventListBlockSeqCB          = 132
	EventGetSeqCBLastNum         = 133
	EventGetBlockBySeq           = 134

	EventLocalBegin    = 135
	EventLocalCommit   = 136
	EventLocalRollback = 137
	EventLocalNew      = 138
	EventLocalClose    = 139

	//mempool
	EventGetProperFee   = 140
	EventReplyProperFee = 141

	EventReExecBlock = 142

	//exec
	EventBlockChainQuery = 212
	EventConsensusQuery  = 213
)

var eventName = map[int]string{
	1:   "EventTx",
	2:   "EventGetBlocks",
	3:   "EventBlocks",
	4:   "EventGetBlockHeight",
	5:   "EventReplyBlockHeight",
	6:   "EventQueryTx",
	7:   "EventTransactionDetail",
	8:   "EventReply",
	9:   "EventTxBroadcast",
	10:  "EventPeerInfo",
	11:  "EventTxList",
	12:  "EventReplyTxList",
	13:  "EventAddBlock",
	14:  "EventBlockBroadcast",
	15:  "EventFetchBlocks",
	16:  "EventAddBlocks",
	17:  "EventTxHashList",
	18:  "EventTxHashListReply",
	19:  "EventGetHeaders",
	20:  "EventHeaders",
	21:  "EventGetMempoolSize",
	22:  "EventMempoolSize",
	23:  "EventStoreGet",
	24:  "EventStoreSet",
	25:  "EventStoreGetReply",
	26:  "EventStoreSetReply",
	27:  "EventReceipts",
	28:  "EventExecTxList",
	29:  "EventPeerList",
	30:  "EventGetLastHeader",
	31:  "EventHeader",
	32:  "EventAddBlockDetail",
	33:  "EventGetMempool",
	34:  "EventGetTransactionByAddr",
	35:  "EventGetTransactionByHash",
	36:  "EventReplyTxInfo",
	37:  "EventWalletGetAccountList",
	38:  "EventWalletAccountList",
	39:  "EventNewAccount",
	40:  "EventWalletAccount",
	41:  "EventWalletTransactionList",
	42:  "EventWalletExecutor",
	43:  "EventWalletImportPrivkey",
	44:  "EventWalletSendToAddress",
	45:  "EventWalletSetFee",
	46:  "EventWalletSetLabel",
	47:  "EventStoreDel",
	48:  "EventWalletMergeBalance",
	49:  "EventReplyHashes",
	50:  "EventWalletSetPasswd",
	51:  "EventWalletLock",
	52:  "EventWalletUnLock",
	53:  "EventTransactionDetails",
	54:  "EventBroadcastAddBlock",
	55:  "EventGetBlockOverview",
	56:  "EventGetAddrOverview",
	57:  "EventReplyBlockOverview",
	58:  "EventReplyAddrOverview",
	59:  "EventGetBlockHash",
	60:  "EventBlockHash",
	61:  "EventGetLastMempool",
	63:  "EventMinerStart",
	64:  "EventMinerStop",
	65:  "EventWalletTickets",
	66:  "EventStoreMemSet",
	67:  "EventStoreRollback",
	68:  "EventStoreCommit",
	69:  "EventCheckBlock",
	70:  "EventGenSeed",
	71:  "EventReplyGenSeed",
	72:  "EventSaveSeed",
	73:  "EventGetSeed",
	74:  "EventReplyGetSeed",
	75:  "EventDelBlock",
	76:  "EventLocalGet",
	77:  "EventLocalReplyValue",
	78:  "EventLocalList",
	79:  "EventLocalSet",
	80:  "EventGetWalletStatus",
	81:  "EventCheckTx",
	82:  "EventReceiptCheckTx",
	84:  "EventReplyQuery",
	86:  "EventFetchBlockHeaders",
	87:  "EventAddBlockHeaders",
	89:  "EventReplyWalletStatus",
	90:  "EventGetLastBlock",
	91:  "EventBlock",
	92:  "EventGetTicketCount",
	93:  "EventReplyGetTicketCount",
	94:  "EventDumpPrivkey",
	95:  "EventReplyPrivkey",
	96:  "EventIsSync",
	97:  "EventReplyIsSync",
	98:  "EventCloseTickets",
	99:  "EventGetAddrTxs",
	100: "EventReplyAddrTxs",
	101: "EventIsNtpClockSync",
	102: "EventReplyIsNtpClockSync",
	103: "EventDelTxList",
	104: "EventStoreGetTotalCoins",
	105: "EventGetTotalCoinsReply",
	106: "EventQueryTotalFee",
	107: "EventSignRawTx",
	108: "EventReplySignRawTx",
	109: "EventSyncBlock",
	110: "EventGetNetInfo",
	111: "EventReplyNetInfo",
	112: "EventErrToFront",
	113: "EventFatalFailure",
	114: "EventReplyFatalFailure",
	115: "EventBindMiner",
	116: "EventReplyBindMiner",
	117: "EventDecodeRawTx",
	118: "EventReplyDecodeRawTx",
	119: "EventGetLastBlockSequence",
	120: "EventReplyLastBlockSequence",
	121: "EventGetBlockSequences",
	122: "EventReplyBlockSequences",
	123: "EventGetBlockByHashes",
	124: "EventReplyBlockDetailsBySeqs",
	125: "EventDelParaChainBlockDetail",
	126: "EventAddParaChainBlockDetail",
	127: "EventGetSeqByHash",
	128: "EventLocalPrefixCount",
	//todo: 这个可能后面会删除
	EventWalletCreateTx: "EventWalletCreateTx",
	EventStoreList:      "EventStoreList",
	EventStoreListReply: "EventStoreListReply",
	// Token
	EventBlockChainQuery: "EventBlockChainQuery",
	EventConsensusQuery:  "EventConsensusQuery",
	EventGetBlockBySeq:   "EventGetBlockBySeq",

	EventLocalBegin:    "EventLocalBegin",
	EventLocalCommit:   "EventLocalCommit",
	EventLocalRollback: "EventLocalRollback",
	EventLocalNew:      "EventLocalNew",
	EventLocalClose:    "EventLocalClose",

	//mempool
	EventGetProperFee:   "EventGetProperFee",
	EventReplyProperFee: "EventReplyProperFee",
}
