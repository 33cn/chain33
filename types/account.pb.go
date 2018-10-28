// Code generated by protoc-gen-go.
// source: account.proto
// DO NOT EDIT!

/*
Package types is a generated protocol buffer package.

It is generated from these files:
	account.proto
	blockchain.proto
	common.proto
	config.proto
	db.proto
	executor.proto
	p2p.proto
	pbft.proto
	rpc.proto
	statistic.proto
	transaction.proto
	wallet.proto

It has these top-level messages:
	Account
	ReceiptExecAccountTransfer
	ReceiptAccountTransfer
	ReqBalance
	Accounts
	ExecAccount
	AllExecBalance
	Header
	Block
	Blocks
	BlockPid
	BlockDetails
	Headers
	HeadersPid
	BlockOverview
	BlockDetail
	Receipts
	PrivacyKV
	PrivacyKVToken
	ReceiptsAndPrivacyKV
	ReceiptCheckTxList
	ChainStatus
	ReqBlocks
	MempoolSize
	ReplyBlockHeight
	BlockBody
	IsCaughtUp
	IsNtpClockSync
	ChainExecutor
	BlockSequence
	BlockSequences
	ParaChainBlockDetail
	Reply
	ReqString
	ReplyString
	ReplyStrings
	ReqInt
	Int64
	ReqHash
	ReplyHash
	ReqNil
	ReqHashes
	ReplyHashes
	KeyValue
	TxHash
	TimeStatus
	ReqKey
	Config
	Log
	MemPool
	Consensus
	Wallet
	Store
	BlockChain
	P2P
	Rpc
	Exec
	Pprof
	LeafNode
	InnerNode
	MAVLProof
	StoreNode
	LocalDBSet
	LocalDBList
	LocalDBGet
	LocalReplyValue
	StoreSet
	StoreDel
	StoreSetWithSync
	StoreGet
	StoreReplyValue
	Genesis
	ExecTxList
	Query
	CreateTxIn
	ArrayConfig
	StringConfig
	Int32Config
	ConfigItem
	ModifyConfig
	ReceiptConfig
	ReplyConfig
	HistoryCertStore
	P2PGetPeerInfo
	P2PPeerInfo
	P2PVersion
	P2PVerAck
	P2PPing
	P2PPong
	P2PGetAddr
	P2PAddr
	P2PAddrList
	P2PExternalInfo
	P2PGetBlocks
	P2PGetMempool
	P2PInv
	Inventory
	P2PGetData
	P2PTx
	P2PBlock
	Versions
	BroadCastData
	P2PGetHeaders
	P2PHeaders
	InvData
	InvDatas
	Peer
	PeerList
	NodeNetInfo
	PeersReply
	PeersInfo
	Operation
	Checkpoint
	Entry
	ViewChange
	Summary
	Result
	Request
	RequestClient
	RequestPrePrepare
	RequestPrepare
	RequestCommit
	RequestCheckpoint
	RequestViewChange
	RequestAck
	RequestNewView
	ClientReply
	TotalFee
	ReqGetTotalCoins
	ReplyGetTotalCoins
	IterateRangeByStateHash
	TicketStatistic
	TicketMinerInfo
	TotalAmount
	AssetsGenesis
	AssetsTransferToExec
	AssetsWithdraw
	AssetsTransfer
	CreateTx
	CreateTransactionGroup
	UnsignTx
	NoBalanceTx
	SignedTx
	Transaction
	Transactions
	RingSignature
	RingSignatureItem
	Signature
	AddrOverview
	ReqAddr
	ReqPrivacy
	HexTx
	ReplyTxInfo
	ReqTxList
	ReplyTxList
	TxHashList
	ReplyTxInfos
	ReceiptLog
	Receipt
	ReceiptData
	TxResult
	TransactionDetail
	TransactionDetails
	ReqAddrs
	ReqDecodeRawTransaction
	UserWrite
	UpgradeMeta
	WalletTxDetail
	WalletTxDetails
	WalletAccountStore
	WalletPwHash
	WalletStatus
	WalletAccounts
	WalletAccount
	WalletUnLock
	GenSeedLang
	GetSeedByPw
	SaveSeedByPw
	ReplySeed
	ReqWalletSetPasswd
	ReqNewAccount
	ReqWalletTransactionList
	ReqWalletImportPrivkey
	ReqWalletSendToAddress
	ReqWalletSetFee
	ReqWalletSetLabel
	ReqWalletMergeBalance
	ReqTokenPreCreate
	ReqTokenFinishCreate
	ReqTokenRevokeCreate
	ReqModifyConfig
	ReqSignRawTx
	ReplySignRawTx
	ReportErrEvent
	Int32
	ReqCreateTransaction
	ReqAccountList
*/
package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Account 的信息
type Account struct {
	// coins标识，目前只有0 一个值
	Currency int32 `protobuf:"varint,1,opt,name=currency" json:"currency,omitempty"`
	// 账户可用余额
	Balance int64 `protobuf:"varint,2,opt,name=balance" json:"balance,omitempty"`
	// 账户冻结余额
	Frozen int64 `protobuf:"varint,3,opt,name=frozen" json:"frozen,omitempty"`
	// 账户的地址
	Addr string `protobuf:"bytes,4,opt,name=addr" json:"addr,omitempty"`
}

func (m *Account) Reset()                    { *m = Account{} }
func (m *Account) String() string            { return proto.CompactTextString(m) }
func (*Account) ProtoMessage()               {}
func (*Account) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Account) GetCurrency() int32 {
	if m != nil {
		return m.Currency
	}
	return 0
}

func (m *Account) GetBalance() int64 {
	if m != nil {
		return m.Balance
	}
	return 0
}

func (m *Account) GetFrozen() int64 {
	if m != nil {
		return m.Frozen
	}
	return 0
}

func (m *Account) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

// 账户余额改变的一个交易回报（合约内）
type ReceiptExecAccountTransfer struct {
	// 合约地址
	ExecAddr string `protobuf:"bytes,1,opt,name=execAddr" json:"execAddr,omitempty"`
	// 转移前
	Prev *Account `protobuf:"bytes,2,opt,name=prev" json:"prev,omitempty"`
	// 转移后
	Current *Account `protobuf:"bytes,3,opt,name=current" json:"current,omitempty"`
}

func (m *ReceiptExecAccountTransfer) Reset()                    { *m = ReceiptExecAccountTransfer{} }
func (m *ReceiptExecAccountTransfer) String() string            { return proto.CompactTextString(m) }
func (*ReceiptExecAccountTransfer) ProtoMessage()               {}
func (*ReceiptExecAccountTransfer) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *ReceiptExecAccountTransfer) GetExecAddr() string {
	if m != nil {
		return m.ExecAddr
	}
	return ""
}

func (m *ReceiptExecAccountTransfer) GetPrev() *Account {
	if m != nil {
		return m.Prev
	}
	return nil
}

func (m *ReceiptExecAccountTransfer) GetCurrent() *Account {
	if m != nil {
		return m.Current
	}
	return nil
}

// 账户余额改变的一个交易回报（coins内）
type ReceiptAccountTransfer struct {
	// 转移前
	Prev *Account `protobuf:"bytes,1,opt,name=prev" json:"prev,omitempty"`
	// 转移后
	Current *Account `protobuf:"bytes,2,opt,name=current" json:"current,omitempty"`
}

func (m *ReceiptAccountTransfer) Reset()                    { *m = ReceiptAccountTransfer{} }
func (m *ReceiptAccountTransfer) String() string            { return proto.CompactTextString(m) }
func (*ReceiptAccountTransfer) ProtoMessage()               {}
func (*ReceiptAccountTransfer) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *ReceiptAccountTransfer) GetPrev() *Account {
	if m != nil {
		return m.Prev
	}
	return nil
}

func (m *ReceiptAccountTransfer) GetCurrent() *Account {
	if m != nil {
		return m.Current
	}
	return nil
}

// 查询一个地址列表在某个执行器中余额
type ReqBalance struct {
	// 地址列表
	Addresses []string `protobuf:"bytes,1,rep,name=addresses" json:"addresses,omitempty"`
	// 执行器名称
	Execer    string `protobuf:"bytes,2,opt,name=execer" json:"execer,omitempty"`
	StateHash string `protobuf:"bytes,3,opt,name=stateHash" json:"stateHash,omitempty"`
}

func (m *ReqBalance) Reset()                    { *m = ReqBalance{} }
func (m *ReqBalance) String() string            { return proto.CompactTextString(m) }
func (*ReqBalance) ProtoMessage()               {}
func (*ReqBalance) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *ReqBalance) GetAddresses() []string {
	if m != nil {
		return m.Addresses
	}
	return nil
}

func (m *ReqBalance) GetExecer() string {
	if m != nil {
		return m.Execer
	}
	return ""
}

func (m *ReqBalance) GetStateHash() string {
	if m != nil {
		return m.StateHash
	}
	return ""
}

// Account 的列表
type Accounts struct {
	Acc []*Account `protobuf:"bytes,1,rep,name=acc" json:"acc,omitempty"`
}

func (m *Accounts) Reset()                    { *m = Accounts{} }
func (m *Accounts) String() string            { return proto.CompactTextString(m) }
func (*Accounts) ProtoMessage()               {}
func (*Accounts) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *Accounts) GetAcc() []*Account {
	if m != nil {
		return m.Acc
	}
	return nil
}

type ExecAccount struct {
	Execer  string   `protobuf:"bytes,1,opt,name=execer" json:"execer,omitempty"`
	Account *Account `protobuf:"bytes,2,opt,name=account" json:"account,omitempty"`
}

func (m *ExecAccount) Reset()                    { *m = ExecAccount{} }
func (m *ExecAccount) String() string            { return proto.CompactTextString(m) }
func (*ExecAccount) ProtoMessage()               {}
func (*ExecAccount) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *ExecAccount) GetExecer() string {
	if m != nil {
		return m.Execer
	}
	return ""
}

func (m *ExecAccount) GetAccount() *Account {
	if m != nil {
		return m.Account
	}
	return nil
}

type AllExecBalance struct {
	Addr        string         `protobuf:"bytes,1,opt,name=addr" json:"addr,omitempty"`
	ExecAccount []*ExecAccount `protobuf:"bytes,2,rep,name=ExecAccount" json:"ExecAccount,omitempty"`
}

func (m *AllExecBalance) Reset()                    { *m = AllExecBalance{} }
func (m *AllExecBalance) String() string            { return proto.CompactTextString(m) }
func (*AllExecBalance) ProtoMessage()               {}
func (*AllExecBalance) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *AllExecBalance) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

func (m *AllExecBalance) GetExecAccount() []*ExecAccount {
	if m != nil {
		return m.ExecAccount
	}
	return nil
}

func init() {
	proto.RegisterType((*Account)(nil), "types.Account")
	proto.RegisterType((*ReceiptExecAccountTransfer)(nil), "types.ReceiptExecAccountTransfer")
	proto.RegisterType((*ReceiptAccountTransfer)(nil), "types.ReceiptAccountTransfer")
	proto.RegisterType((*ReqBalance)(nil), "types.ReqBalance")
	proto.RegisterType((*Accounts)(nil), "types.Accounts")
	proto.RegisterType((*ExecAccount)(nil), "types.ExecAccount")
	proto.RegisterType((*AllExecBalance)(nil), "types.AllExecBalance")
}

func init() { proto.RegisterFile("account.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 357 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x8c, 0x52, 0x41, 0x4f, 0xc2, 0x30,
	0x14, 0x4e, 0x19, 0x30, 0xf6, 0x88, 0x1c, 0x7a, 0x20, 0x0b, 0xf1, 0xb0, 0x34, 0x1e, 0x76, 0x30,
	0x33, 0x71, 0xfe, 0x01, 0x48, 0x4c, 0xbc, 0x99, 0x34, 0x9e, 0x38, 0x59, 0xca, 0x43, 0x88, 0xcb,
	0x36, 0xdb, 0x62, 0xc4, 0x1f, 0xe0, 0xef, 0x36, 0x2d, 0x1d, 0x0c, 0x89, 0xc6, 0xd3, 0xf6, 0xbd,
	0xaf, 0xef, 0xfb, 0xbe, 0xf7, 0x5a, 0xb8, 0x10, 0x52, 0x56, 0xdb, 0xd2, 0x64, 0xb5, 0xaa, 0x4c,
	0x45, 0x7b, 0x66, 0x57, 0xa3, 0x66, 0xaf, 0x10, 0x4e, 0xf7, 0x75, 0x3a, 0x81, 0x81, 0xdc, 0x2a,
	0x85, 0xa5, 0xdc, 0xc5, 0x24, 0x21, 0x69, 0x8f, 0x1f, 0x30, 0x8d, 0x21, 0x5c, 0x88, 0x42, 0x94,
	0x12, 0xe3, 0x4e, 0x42, 0xd2, 0x80, 0x37, 0x90, 0x8e, 0xa1, 0xbf, 0x52, 0xd5, 0x27, 0x96, 0x71,
	0xe0, 0x08, 0x8f, 0x28, 0x85, 0xae, 0x58, 0x2e, 0x55, 0xdc, 0x4d, 0x48, 0x1a, 0x71, 0xf7, 0xcf,
	0xbe, 0x08, 0x4c, 0x38, 0x4a, 0xdc, 0xd4, 0xe6, 0xfe, 0x03, 0xa5, 0x37, 0x7e, 0x52, 0xa2, 0xd4,
	0x2b, 0x54, 0x36, 0x00, 0xda, 0xb2, 0x6d, 0x23, 0xae, 0xed, 0x80, 0x29, 0x83, 0x6e, 0xad, 0xf0,
	0xdd, 0xb9, 0x0f, 0x6f, 0x47, 0x99, 0x4b, 0x9f, 0x79, 0x05, 0xee, 0x38, 0x9a, 0x42, 0xb8, 0x0f,
	0x6c, 0x5c, 0x96, 0xf3, 0x63, 0x0d, 0xcd, 0x56, 0x30, 0xf6, 0x39, 0x7e, 0x66, 0x68, 0x7c, 0xc8,
	0xff, 0x7c, 0x3a, 0x7f, 0xfb, 0x3c, 0x03, 0x70, 0x7c, 0x9b, 0xf9, 0x55, 0x5d, 0x42, 0x64, 0xd7,
	0x80, 0x5a, 0xa3, 0x8e, 0x49, 0x12, 0xa4, 0x11, 0x3f, 0x16, 0xec, 0x22, 0xed, 0xb4, 0xa8, 0x9c,
	0x68, 0xc4, 0x3d, 0xb2, 0x5d, 0xda, 0x08, 0x83, 0x0f, 0x42, 0xaf, 0xdd, 0x5c, 0x11, 0x3f, 0x16,
	0xd8, 0x35, 0x0c, 0xbc, 0xab, 0xa6, 0x09, 0x04, 0x42, 0x4a, 0xa7, 0x7c, 0x9e, 0xc9, 0x52, 0xec,
	0x11, 0x86, 0xad, 0xc5, 0xb7, 0x2c, 0xc9, 0x89, 0x65, 0x0a, 0xa1, 0x7f, 0x2c, 0xbf, 0x0d, 0xe8,
	0x69, 0x36, 0x87, 0xd1, 0xb4, 0x28, 0xac, 0x66, 0x33, 0x64, 0x73, 0xef, 0xe4, 0x78, 0xef, 0xf4,
	0xee, 0xc4, 0x36, 0xee, 0xb8, 0x80, 0xd4, 0x6b, 0xb6, 0x18, 0xde, 0x3e, 0x36, 0xbb, 0x9a, 0xb3,
	0x97, 0x8d, 0x29, 0xc4, 0x22, 0xcb, 0xf3, 0x4c, 0x96, 0x37, 0x72, 0x2d, 0x36, 0x65, 0x9e, 0x1f,
	0xbe, 0xae, 0x7d, 0xd1, 0x77, 0xcf, 0x39, 0xff, 0x0e, 0x00, 0x00, 0xff, 0xff, 0xd7, 0x2c, 0xad,
	0xbb, 0xdf, 0x02, 0x00, 0x00,
}
