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
	rpc.proto
	transaction.proto
	wallet.proto

It has these top-level messages:
	Accounts
	Account
	ReceiptExecAccountTransfer
	ReqBalance
	ReceiptAccountTransfer
	Header
	Block
	Blocks
	BlockDetails
	Headers
	BlockOverview
	BlockDetail
	Receipts
	ReceiptCheckTxList
	ChainStatus
	ReqBlocks
	MempoolSize
	ReplyBlockHeight
	BlockBody
	IsCaughtUp
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
	Config
	MemPool
	Consensus
	Wallet
	Store
	LocalStore
	BlockChain
	P2P
	LeafNode
	InnerNode
	MAVLProof
	StoreNode
	LocalDBSet
	LocalDBList
	LocalDBGet
	LocalReplyValue
	StoreSet
	StoreGet
	StoreReplyValue
	Genesis
	CoinsAction
	CoinsGenesis
	CoinsTransfer
	CoinsWithdraw
	Hashlock
	HashlockAction
	HashlockLock
	HashlockUnlock
	HashlockSend
	HashRecv
	Hashlockquery
	Ticket
	TicketAction
	TicketMiner
	TicketBind
	TicketOpen
	TicketGenesis
	TicketClose
	TicketList
	TicketInfos
	ReplyTicketList
	ReplyWalletTickets
	ReceiptTicket
	ReceiptTicketBind
	ExecTxList
	Query
	Norm
	NormAction
	NormPut
	RetrievePara
	Retrieve
	RetrieveAction
	BackupRetrieve
	PreRetrieve
	PerformRetrieve
	CancelRetrieve
	P2PGetPeerInfo
	P2PPeerInfo
	P2PVersion
	P2PVerAck
	P2PPing
	P2PPong
	P2PGetAddr
	P2PAddr
	P2PExternalInfo
	P2PGetBlocks
	P2PGetMempool
	P2PInv
	Inventory
	P2PGetData
	P2PTx
	P2PBlock
	BroadCastData
	P2PGetHeaders
	P2PHeaders
	InvData
	InvDatas
	Peer
	PeerList
	CreateTx
	UnsignTx
	SignedTx
	Transaction
	Signature
	AddrOverview
	ReqAddr
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
	MinerFlag
	ReqWalletTransactionList
	ReqWalletImportPrivKey
	ReqWalletSendToAddress
	ReqWalletSetFee
	ReqWalletSetLabel
	ReqWalletMergeBalance
	ReqStr
	ReplyStr
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

// currency = 0 -> origin account
type Accounts struct {
	Acc []*Account `protobuf:"bytes,1,rep,name=acc" json:"acc,omitempty"`
}

func (m *Accounts) Reset()                    { *m = Accounts{} }
func (m *Accounts) String() string            { return proto.CompactTextString(m) }
func (*Accounts) ProtoMessage()               {}
func (*Accounts) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Accounts) GetAcc() []*Account {
	if m != nil {
		return m.Acc
	}
	return nil
}

type Account struct {
	Currency int32  `protobuf:"varint,1,opt,name=currency" json:"currency,omitempty"`
	Balance  int64  `protobuf:"varint,2,opt,name=balance" json:"balance,omitempty"`
	Frozen   int64  `protobuf:"varint,3,opt,name=frozen" json:"frozen,omitempty"`
	Addr     string `protobuf:"bytes,4,opt,name=addr" json:"addr,omitempty"`
}

func (m *Account) Reset()                    { *m = Account{} }
func (m *Account) String() string            { return proto.CompactTextString(m) }
func (*Account) ProtoMessage()               {}
func (*Account) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

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

type ReceiptExecAccountTransfer struct {
	ExecAddr string   `protobuf:"bytes,1,opt,name=execAddr" json:"execAddr,omitempty"`
	Prev     *Account `protobuf:"bytes,2,opt,name=prev" json:"prev,omitempty"`
	Current  *Account `protobuf:"bytes,3,opt,name=current" json:"current,omitempty"`
}

func (m *ReceiptExecAccountTransfer) Reset()                    { *m = ReceiptExecAccountTransfer{} }
func (m *ReceiptExecAccountTransfer) String() string            { return proto.CompactTextString(m) }
func (*ReceiptExecAccountTransfer) ProtoMessage()               {}
func (*ReceiptExecAccountTransfer) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

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

type ReqBalance struct {
	Addresses []string `protobuf:"bytes,1,rep,name=addresses" json:"addresses,omitempty"`
	Execer    string   `protobuf:"bytes,2,opt,name=execer" json:"execer,omitempty"`
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

type ReceiptAccountTransfer struct {
	Prev    *Account `protobuf:"bytes,1,opt,name=prev" json:"prev,omitempty"`
	Current *Account `protobuf:"bytes,2,opt,name=current" json:"current,omitempty"`
}

func (m *ReceiptAccountTransfer) Reset()                    { *m = ReceiptAccountTransfer{} }
func (m *ReceiptAccountTransfer) String() string            { return proto.CompactTextString(m) }
func (*ReceiptAccountTransfer) ProtoMessage()               {}
func (*ReceiptAccountTransfer) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

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

func init() {
	proto.RegisterType((*Accounts)(nil), "types.Accounts")
	proto.RegisterType((*Account)(nil), "types.Account")
	proto.RegisterType((*ReceiptExecAccountTransfer)(nil), "types.ReceiptExecAccountTransfer")
	proto.RegisterType((*ReqBalance)(nil), "types.ReqBalance")
	proto.RegisterType((*ReceiptAccountTransfer)(nil), "types.ReceiptAccountTransfer")
}

func init() { proto.RegisterFile("account.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 269 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x8c, 0x91, 0xc1, 0x4a, 0x03, 0x31,
	0x10, 0x86, 0x49, 0x77, 0xdb, 0xed, 0x8e, 0xe8, 0x21, 0x87, 0x12, 0x8a, 0x87, 0x90, 0x53, 0x0e,
	0xb2, 0x07, 0x7d, 0x02, 0x0b, 0xbe, 0x40, 0xf0, 0x05, 0xd2, 0x74, 0x16, 0x44, 0xc9, 0xae, 0x49,
	0x2a, 0xd6, 0x07, 0xf0, 0xb9, 0x25, 0xd3, 0x6c, 0x05, 0x0b, 0xd2, 0xdb, 0xfe, 0xf3, 0xcf, 0xf0,
	0x7f, 0xff, 0x06, 0xae, 0xad, 0x73, 0xc3, 0xde, 0xa7, 0x6e, 0x0c, 0x43, 0x1a, 0xf8, 0x3c, 0x1d,
	0x46, 0x8c, 0xea, 0x0e, 0x96, 0x8f, 0xc7, 0x79, 0xe4, 0x12, 0x2a, 0xeb, 0x9c, 0x60, 0xb2, 0xd2,
	0x57, 0xf7, 0x37, 0x1d, 0x2d, 0x74, 0xc5, 0x35, 0xd9, 0x52, 0xaf, 0xd0, 0x14, 0xcd, 0xd7, 0xb0,
	0x74, 0xfb, 0x10, 0xd0, 0xbb, 0x83, 0x60, 0x92, 0xe9, 0xb9, 0x39, 0x69, 0x2e, 0xa0, 0xd9, 0xda,
	0x37, 0xeb, 0x1d, 0x8a, 0x99, 0x64, 0xba, 0x32, 0x93, 0xe4, 0x2b, 0x58, 0xf4, 0x61, 0xf8, 0x42,
	0x2f, 0x2a, 0x32, 0x8a, 0xe2, 0x1c, 0x6a, 0xbb, 0xdb, 0x05, 0x51, 0x4b, 0xa6, 0x5b, 0x43, 0xdf,
	0xea, 0x9b, 0xc1, 0xda, 0xa0, 0xc3, 0x97, 0x31, 0x3d, 0x7d, 0xa2, 0x2b, 0xc1, 0xcf, 0xc1, 0xfa,
	0xd8, 0x63, 0xc8, 0x00, 0x98, 0xc7, 0xf9, 0x8c, 0xd1, 0xd9, 0x49, 0x73, 0x05, 0xf5, 0x18, 0xf0,
	0x83, 0xd2, 0xcf, 0xab, 0x90, 0xc7, 0x35, 0x34, 0x47, 0xe0, 0x44, 0x2c, 0xe7, 0x6b, 0x93, 0xad,
	0x36, 0x00, 0x06, 0xdf, 0x37, 0xa5, 0xc2, 0x2d, 0xb4, 0x19, 0x0f, 0x63, 0xc4, 0x48, 0xff, 0xaa,
	0x35, 0xbf, 0x83, 0x5c, 0x30, 0x53, 0x60, 0xa0, 0xec, 0xd6, 0x14, 0xa5, 0x7a, 0x58, 0x95, 0x2e,
	0x7f, 0x7b, 0x4c, 0xac, 0xec, 0x32, 0xd6, 0xd9, 0xbf, 0xac, 0xdb, 0x05, 0xbd, 0xee, 0xc3, 0x4f,
	0x00, 0x00, 0x00, 0xff, 0xff, 0x14, 0xe5, 0x03, 0x90, 0xee, 0x01, 0x00, 0x00,
}
