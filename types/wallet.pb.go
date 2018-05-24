// Code generated by protoc-gen-go.
// source: wallet.proto
// DO NOT EDIT!

package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// 钱包模块存贮的tx交易详细信息
// 	 tx : tx交易信息
// 	 receipt :交易收据信息
// 	 height :交易所在的区块高度
// 	 index :交易所在区块中的索引
// 	 blocktime :交易所在区块的时标
// 	 amount :交易量
// 	 fromaddr :交易打出地址
// 	 txhash : 交易对应的哈希值
// 	 actionName  :交易对应的函数调用
type WalletTxDetail struct {
	Tx         *Transaction `protobuf:"bytes,1,opt,name=tx" json:"tx,omitempty"`
	Receipt    *ReceiptData `protobuf:"bytes,2,opt,name=receipt" json:"receipt,omitempty"`
	Height     int64        `protobuf:"varint,3,opt,name=height" json:"height,omitempty"`
	Index      int64        `protobuf:"varint,4,opt,name=index" json:"index,omitempty"`
	Blocktime  int64        `protobuf:"varint,5,opt,name=blocktime" json:"blocktime,omitempty"`
	Amount     int64        `protobuf:"varint,6,opt,name=amount" json:"amount,omitempty"`
	Fromaddr   string       `protobuf:"bytes,7,opt,name=fromaddr" json:"fromaddr,omitempty"`
	Txhash     []byte       `protobuf:"bytes,8,opt,name=txhash,proto3" json:"txhash,omitempty"`
	ActionName string       `protobuf:"bytes,9,opt,name=actionName" json:"actionName,omitempty"`
}

func (m *WalletTxDetail) Reset()                    { *m = WalletTxDetail{} }
func (m *WalletTxDetail) String() string            { return proto.CompactTextString(m) }
func (*WalletTxDetail) ProtoMessage()               {}
func (*WalletTxDetail) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{0} }

func (m *WalletTxDetail) GetTx() *Transaction {
	if m != nil {
		return m.Tx
	}
	return nil
}

func (m *WalletTxDetail) GetReceipt() *ReceiptData {
	if m != nil {
		return m.Receipt
	}
	return nil
}

func (m *WalletTxDetail) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *WalletTxDetail) GetIndex() int64 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *WalletTxDetail) GetBlocktime() int64 {
	if m != nil {
		return m.Blocktime
	}
	return 0
}

func (m *WalletTxDetail) GetAmount() int64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

func (m *WalletTxDetail) GetFromaddr() string {
	if m != nil {
		return m.Fromaddr
	}
	return ""
}

func (m *WalletTxDetail) GetTxhash() []byte {
	if m != nil {
		return m.Txhash
	}
	return nil
}

func (m *WalletTxDetail) GetActionName() string {
	if m != nil {
		return m.ActionName
	}
	return ""
}

type WalletTxDetails struct {
	TxDetails []*WalletTxDetail `protobuf:"bytes,1,rep,name=txDetails" json:"txDetails,omitempty"`
}

func (m *WalletTxDetails) Reset()                    { *m = WalletTxDetails{} }
func (m *WalletTxDetails) String() string            { return proto.CompactTextString(m) }
func (*WalletTxDetails) ProtoMessage()               {}
func (*WalletTxDetails) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{1} }

func (m *WalletTxDetails) GetTxDetails() []*WalletTxDetail {
	if m != nil {
		return m.TxDetails
	}
	return nil
}

// 钱包模块存贮的账户信息
// 	 privkey : 账户地址对应的私钥
// 	 label :账户地址对应的标签
// 	 addr :账户地址
// 	 timeStamp :创建账户时的时标
type WalletAccountStore struct {
	Privkey   string `protobuf:"bytes,1,opt,name=privkey" json:"privkey,omitempty"`
	Label     string `protobuf:"bytes,2,opt,name=label" json:"label,omitempty"`
	Addr      string `protobuf:"bytes,3,opt,name=addr" json:"addr,omitempty"`
	TimeStamp string `protobuf:"bytes,4,opt,name=timeStamp" json:"timeStamp,omitempty"`
}

func (m *WalletAccountStore) Reset()                    { *m = WalletAccountStore{} }
func (m *WalletAccountStore) String() string            { return proto.CompactTextString(m) }
func (*WalletAccountStore) ProtoMessage()               {}
func (*WalletAccountStore) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{2} }

func (m *WalletAccountStore) GetPrivkey() string {
	if m != nil {
		return m.Privkey
	}
	return ""
}

func (m *WalletAccountStore) GetLabel() string {
	if m != nil {
		return m.Label
	}
	return ""
}

func (m *WalletAccountStore) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

func (m *WalletAccountStore) GetTimeStamp() string {
	if m != nil {
		return m.TimeStamp
	}
	return ""
}

// 钱包模块通过一个随机值对钱包密码加密
// 	 pwHash : 对钱包密码和一个随机值组合进行哈希计算
// 	 randstr :对钱包密码加密的一个随机值
type WalletPwHash struct {
	PwHash  []byte `protobuf:"bytes,1,opt,name=pwHash,proto3" json:"pwHash,omitempty"`
	Randstr string `protobuf:"bytes,2,opt,name=randstr" json:"randstr,omitempty"`
}

func (m *WalletPwHash) Reset()                    { *m = WalletPwHash{} }
func (m *WalletPwHash) String() string            { return proto.CompactTextString(m) }
func (*WalletPwHash) ProtoMessage()               {}
func (*WalletPwHash) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{3} }

func (m *WalletPwHash) GetPwHash() []byte {
	if m != nil {
		return m.PwHash
	}
	return nil
}

func (m *WalletPwHash) GetRandstr() string {
	if m != nil {
		return m.Randstr
	}
	return ""
}

// 钱包当前的状态
// 	 isWalletLock : 钱包是否锁状态，true锁定，false解锁
// 	 isAutoMining :钱包是否开启挖矿功能，true开启挖矿，false关闭挖矿
// 	 isHasSeed : 钱包是否有种子，true已有，false没有
// 	 isTicketLock :钱包挖矿买票锁状态，true锁定，false解锁，只能用于挖矿转账
type WalletStatus struct {
	IsWalletLock bool `protobuf:"varint,1,opt,name=isWalletLock" json:"isWalletLock,omitempty"`
	IsAutoMining bool `protobuf:"varint,2,opt,name=isAutoMining" json:"isAutoMining,omitempty"`
	IsHasSeed    bool `protobuf:"varint,3,opt,name=isHasSeed" json:"isHasSeed,omitempty"`
	IsTicketLock bool `protobuf:"varint,4,opt,name=isTicketLock" json:"isTicketLock,omitempty"`
}

func (m *WalletStatus) Reset()                    { *m = WalletStatus{} }
func (m *WalletStatus) String() string            { return proto.CompactTextString(m) }
func (*WalletStatus) ProtoMessage()               {}
func (*WalletStatus) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{4} }

func (m *WalletStatus) GetIsWalletLock() bool {
	if m != nil {
		return m.IsWalletLock
	}
	return false
}

func (m *WalletStatus) GetIsAutoMining() bool {
	if m != nil {
		return m.IsAutoMining
	}
	return false
}

func (m *WalletStatus) GetIsHasSeed() bool {
	if m != nil {
		return m.IsHasSeed
	}
	return false
}

func (m *WalletStatus) GetIsTicketLock() bool {
	if m != nil {
		return m.IsTicketLock
	}
	return false
}

type WalletAccounts struct {
	Wallets []*WalletAccount `protobuf:"bytes,1,rep,name=wallets" json:"wallets,omitempty"`
}

func (m *WalletAccounts) Reset()                    { *m = WalletAccounts{} }
func (m *WalletAccounts) String() string            { return proto.CompactTextString(m) }
func (*WalletAccounts) ProtoMessage()               {}
func (*WalletAccounts) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{5} }

func (m *WalletAccounts) GetWallets() []*WalletAccount {
	if m != nil {
		return m.Wallets
	}
	return nil
}

type WalletAccount struct {
	Acc   *Account `protobuf:"bytes,1,opt,name=acc" json:"acc,omitempty"`
	Label string   `protobuf:"bytes,2,opt,name=label" json:"label,omitempty"`
}

func (m *WalletAccount) Reset()                    { *m = WalletAccount{} }
func (m *WalletAccount) String() string            { return proto.CompactTextString(m) }
func (*WalletAccount) ProtoMessage()               {}
func (*WalletAccount) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{6} }

func (m *WalletAccount) GetAcc() *Account {
	if m != nil {
		return m.Acc
	}
	return nil
}

func (m *WalletAccount) GetLabel() string {
	if m != nil {
		return m.Label
	}
	return ""
}

// 钱包解锁
// 	 passwd : 钱包密码
// 	 timeout :钱包解锁时间，0，一直解锁，非0值，超时之后继续锁定
// 	 walletOrTicket :解锁整个钱包还是只解锁挖矿买票功能，1只解锁挖矿买票，0解锁整个钱包
type WalletUnLock struct {
	Passwd         string `protobuf:"bytes,1,opt,name=passwd" json:"passwd,omitempty"`
	Timeout        int64  `protobuf:"varint,2,opt,name=timeout" json:"timeout,omitempty"`
	WalletOrTicket bool   `protobuf:"varint,3,opt,name=walletOrTicket" json:"walletOrTicket,omitempty"`
}

func (m *WalletUnLock) Reset()                    { *m = WalletUnLock{} }
func (m *WalletUnLock) String() string            { return proto.CompactTextString(m) }
func (*WalletUnLock) ProtoMessage()               {}
func (*WalletUnLock) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{7} }

func (m *WalletUnLock) GetPasswd() string {
	if m != nil {
		return m.Passwd
	}
	return ""
}

func (m *WalletUnLock) GetTimeout() int64 {
	if m != nil {
		return m.Timeout
	}
	return 0
}

func (m *WalletUnLock) GetWalletOrTicket() bool {
	if m != nil {
		return m.WalletOrTicket
	}
	return false
}

type GenSeedLang struct {
	Lang int32 `protobuf:"varint,1,opt,name=lang" json:"lang,omitempty"`
}

func (m *GenSeedLang) Reset()                    { *m = GenSeedLang{} }
func (m *GenSeedLang) String() string            { return proto.CompactTextString(m) }
func (*GenSeedLang) ProtoMessage()               {}
func (*GenSeedLang) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{8} }

func (m *GenSeedLang) GetLang() int32 {
	if m != nil {
		return m.Lang
	}
	return 0
}

type GetSeedByPw struct {
	Passwd string `protobuf:"bytes,1,opt,name=passwd" json:"passwd,omitempty"`
}

func (m *GetSeedByPw) Reset()                    { *m = GetSeedByPw{} }
func (m *GetSeedByPw) String() string            { return proto.CompactTextString(m) }
func (*GetSeedByPw) ProtoMessage()               {}
func (*GetSeedByPw) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{9} }

func (m *GetSeedByPw) GetPasswd() string {
	if m != nil {
		return m.Passwd
	}
	return ""
}

// 存储钱包的种子
// 	 seed : 钱包种子
// 	 passwd :钱包密码
type SaveSeedByPw struct {
	Seed   string `protobuf:"bytes,1,opt,name=seed" json:"seed,omitempty"`
	Passwd string `protobuf:"bytes,2,opt,name=passwd" json:"passwd,omitempty"`
}

func (m *SaveSeedByPw) Reset()                    { *m = SaveSeedByPw{} }
func (m *SaveSeedByPw) String() string            { return proto.CompactTextString(m) }
func (*SaveSeedByPw) ProtoMessage()               {}
func (*SaveSeedByPw) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{10} }

func (m *SaveSeedByPw) GetSeed() string {
	if m != nil {
		return m.Seed
	}
	return ""
}

func (m *SaveSeedByPw) GetPasswd() string {
	if m != nil {
		return m.Passwd
	}
	return ""
}

type ReplySeed struct {
	Seed string `protobuf:"bytes,1,opt,name=seed" json:"seed,omitempty"`
}

func (m *ReplySeed) Reset()                    { *m = ReplySeed{} }
func (m *ReplySeed) String() string            { return proto.CompactTextString(m) }
func (*ReplySeed) ProtoMessage()               {}
func (*ReplySeed) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{11} }

func (m *ReplySeed) GetSeed() string {
	if m != nil {
		return m.Seed
	}
	return ""
}

type ReqWalletSetPasswd struct {
	OldPass string `protobuf:"bytes,1,opt,name=oldPass" json:"oldPass,omitempty"`
	NewPass string `protobuf:"bytes,2,opt,name=newPass" json:"newPass,omitempty"`
}

func (m *ReqWalletSetPasswd) Reset()                    { *m = ReqWalletSetPasswd{} }
func (m *ReqWalletSetPasswd) String() string            { return proto.CompactTextString(m) }
func (*ReqWalletSetPasswd) ProtoMessage()               {}
func (*ReqWalletSetPasswd) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{12} }

func (m *ReqWalletSetPasswd) GetOldPass() string {
	if m != nil {
		return m.OldPass
	}
	return ""
}

func (m *ReqWalletSetPasswd) GetNewPass() string {
	if m != nil {
		return m.NewPass
	}
	return ""
}

type ReqNewAccount struct {
	Label string `protobuf:"bytes,1,opt,name=label" json:"label,omitempty"`
}

func (m *ReqNewAccount) Reset()                    { *m = ReqNewAccount{} }
func (m *ReqNewAccount) String() string            { return proto.CompactTextString(m) }
func (*ReqNewAccount) ProtoMessage()               {}
func (*ReqNewAccount) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{13} }

func (m *ReqNewAccount) GetLabel() string {
	if m != nil {
		return m.Label
	}
	return ""
}

type MinerFlag struct {
	Flag    int32 `protobuf:"varint,1,opt,name=flag" json:"flag,omitempty"`
	Reserve int64 `protobuf:"varint,2,opt,name=reserve" json:"reserve,omitempty"`
}

func (m *MinerFlag) Reset()                    { *m = MinerFlag{} }
func (m *MinerFlag) String() string            { return proto.CompactTextString(m) }
func (*MinerFlag) ProtoMessage()               {}
func (*MinerFlag) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{14} }

func (m *MinerFlag) GetFlag() int32 {
	if m != nil {
		return m.Flag
	}
	return 0
}

func (m *MinerFlag) GetReserve() int64 {
	if m != nil {
		return m.Reserve
	}
	return 0
}

// 获取钱包交易的详细信息
// 	 fromTx : []byte( Sprintf("%018d", height*100000 + index)，
// 				表示从高度 height 中的 index 开始获取交易列表；
// 			    第一次传参为空，获取最新的交易。)
// 	 count :获取交易列表的个数。
// 	 direction :查找方式；0，上一页；1，下一页。
type ReqWalletTransactionList struct {
	FromTx    []byte `protobuf:"bytes,1,opt,name=fromTx,proto3" json:"fromTx,omitempty"`
	Count     int32  `protobuf:"varint,2,opt,name=count" json:"count,omitempty"`
	Direction int32  `protobuf:"varint,3,opt,name=direction" json:"direction,omitempty"`
}

func (m *ReqWalletTransactionList) Reset()                    { *m = ReqWalletTransactionList{} }
func (m *ReqWalletTransactionList) String() string            { return proto.CompactTextString(m) }
func (*ReqWalletTransactionList) ProtoMessage()               {}
func (*ReqWalletTransactionList) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{15} }

func (m *ReqWalletTransactionList) GetFromTx() []byte {
	if m != nil {
		return m.FromTx
	}
	return nil
}

func (m *ReqWalletTransactionList) GetCount() int32 {
	if m != nil {
		return m.Count
	}
	return 0
}

func (m *ReqWalletTransactionList) GetDirection() int32 {
	if m != nil {
		return m.Direction
	}
	return 0
}

type ReqWalletImportPrivKey struct {
	// bitcoin 的私钥格式
	Privkey string `protobuf:"bytes,1,opt,name=privkey" json:"privkey,omitempty"`
	Label   string `protobuf:"bytes,2,opt,name=label" json:"label,omitempty"`
}

func (m *ReqWalletImportPrivKey) Reset()                    { *m = ReqWalletImportPrivKey{} }
func (m *ReqWalletImportPrivKey) String() string            { return proto.CompactTextString(m) }
func (*ReqWalletImportPrivKey) ProtoMessage()               {}
func (*ReqWalletImportPrivKey) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{16} }

func (m *ReqWalletImportPrivKey) GetPrivkey() string {
	if m != nil {
		return m.Privkey
	}
	return ""
}

func (m *ReqWalletImportPrivKey) GetLabel() string {
	if m != nil {
		return m.Label
	}
	return ""
}

// 发送交易
// 	 from : 打出地址
// 	 to :接受地址
// 	 amount : 转账额度
// 	 note :转账备注
type ReqWalletSendToAddress struct {
	From        string `protobuf:"bytes,1,opt,name=from" json:"from,omitempty"`
	To          string `protobuf:"bytes,2,opt,name=to" json:"to,omitempty"`
	Amount      int64  `protobuf:"varint,3,opt,name=amount" json:"amount,omitempty"`
	Note        string `protobuf:"bytes,4,opt,name=note" json:"note,omitempty"`
	IsToken     bool   `protobuf:"varint,5,opt,name=isToken" json:"isToken,omitempty"`
	TokenSymbol string `protobuf:"bytes,6,opt,name=tokenSymbol" json:"tokenSymbol,omitempty"`
}

func (m *ReqWalletSendToAddress) Reset()                    { *m = ReqWalletSendToAddress{} }
func (m *ReqWalletSendToAddress) String() string            { return proto.CompactTextString(m) }
func (*ReqWalletSendToAddress) ProtoMessage()               {}
func (*ReqWalletSendToAddress) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{17} }

func (m *ReqWalletSendToAddress) GetFrom() string {
	if m != nil {
		return m.From
	}
	return ""
}

func (m *ReqWalletSendToAddress) GetTo() string {
	if m != nil {
		return m.To
	}
	return ""
}

func (m *ReqWalletSendToAddress) GetAmount() int64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

func (m *ReqWalletSendToAddress) GetNote() string {
	if m != nil {
		return m.Note
	}
	return ""
}

func (m *ReqWalletSendToAddress) GetIsToken() bool {
	if m != nil {
		return m.IsToken
	}
	return false
}

func (m *ReqWalletSendToAddress) GetTokenSymbol() string {
	if m != nil {
		return m.TokenSymbol
	}
	return ""
}

type ReqWalletSetFee struct {
	Amount int64 `protobuf:"varint,1,opt,name=amount" json:"amount,omitempty"`
}

func (m *ReqWalletSetFee) Reset()                    { *m = ReqWalletSetFee{} }
func (m *ReqWalletSetFee) String() string            { return proto.CompactTextString(m) }
func (*ReqWalletSetFee) ProtoMessage()               {}
func (*ReqWalletSetFee) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{18} }

func (m *ReqWalletSetFee) GetAmount() int64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

type ReqWalletSetLabel struct {
	Addr  string `protobuf:"bytes,1,opt,name=addr" json:"addr,omitempty"`
	Label string `protobuf:"bytes,2,opt,name=label" json:"label,omitempty"`
}

func (m *ReqWalletSetLabel) Reset()                    { *m = ReqWalletSetLabel{} }
func (m *ReqWalletSetLabel) String() string            { return proto.CompactTextString(m) }
func (*ReqWalletSetLabel) ProtoMessage()               {}
func (*ReqWalletSetLabel) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{19} }

func (m *ReqWalletSetLabel) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

func (m *ReqWalletSetLabel) GetLabel() string {
	if m != nil {
		return m.Label
	}
	return ""
}

type ReqWalletMergeBalance struct {
	To string `protobuf:"bytes,1,opt,name=to" json:"to,omitempty"`
}

func (m *ReqWalletMergeBalance) Reset()                    { *m = ReqWalletMergeBalance{} }
func (m *ReqWalletMergeBalance) String() string            { return proto.CompactTextString(m) }
func (*ReqWalletMergeBalance) ProtoMessage()               {}
func (*ReqWalletMergeBalance) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{20} }

func (m *ReqWalletMergeBalance) GetTo() string {
	if m != nil {
		return m.To
	}
	return ""
}

type ReqStr struct {
	ReqStr string `protobuf:"bytes,1,opt,name=reqStr" json:"reqStr,omitempty"`
}

func (m *ReqStr) Reset()                    { *m = ReqStr{} }
func (m *ReqStr) String() string            { return proto.CompactTextString(m) }
func (*ReqStr) ProtoMessage()               {}
func (*ReqStr) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{21} }

func (m *ReqStr) GetReqStr() string {
	if m != nil {
		return m.ReqStr
	}
	return ""
}

type ReplyStr struct {
	Replystr string `protobuf:"bytes,1,opt,name=replystr" json:"replystr,omitempty"`
}

func (m *ReplyStr) Reset()                    { *m = ReplyStr{} }
func (m *ReplyStr) String() string            { return proto.CompactTextString(m) }
func (*ReplyStr) ProtoMessage()               {}
func (*ReplyStr) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{22} }

func (m *ReplyStr) GetReplystr() string {
	if m != nil {
		return m.Replystr
	}
	return ""
}

type ReqTokenPreCreate struct {
	CreatorAddr  string `protobuf:"bytes,1,opt,name=creator_addr,json=creatorAddr" json:"creator_addr,omitempty"`
	Name         string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	Symbol       string `protobuf:"bytes,3,opt,name=symbol" json:"symbol,omitempty"`
	Introduction string `protobuf:"bytes,4,opt,name=introduction" json:"introduction,omitempty"`
	OwnerAddr    string `protobuf:"bytes,5,opt,name=owner_addr,json=ownerAddr" json:"owner_addr,omitempty"`
	Total        int64  `protobuf:"varint,6,opt,name=total" json:"total,omitempty"`
	Price        int64  `protobuf:"varint,7,opt,name=price" json:"price,omitempty"`
}

func (m *ReqTokenPreCreate) Reset()                    { *m = ReqTokenPreCreate{} }
func (m *ReqTokenPreCreate) String() string            { return proto.CompactTextString(m) }
func (*ReqTokenPreCreate) ProtoMessage()               {}
func (*ReqTokenPreCreate) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{23} }

func (m *ReqTokenPreCreate) GetCreatorAddr() string {
	if m != nil {
		return m.CreatorAddr
	}
	return ""
}

func (m *ReqTokenPreCreate) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *ReqTokenPreCreate) GetSymbol() string {
	if m != nil {
		return m.Symbol
	}
	return ""
}

func (m *ReqTokenPreCreate) GetIntroduction() string {
	if m != nil {
		return m.Introduction
	}
	return ""
}

func (m *ReqTokenPreCreate) GetOwnerAddr() string {
	if m != nil {
		return m.OwnerAddr
	}
	return ""
}

func (m *ReqTokenPreCreate) GetTotal() int64 {
	if m != nil {
		return m.Total
	}
	return 0
}

func (m *ReqTokenPreCreate) GetPrice() int64 {
	if m != nil {
		return m.Price
	}
	return 0
}

type ReqTokenFinishCreate struct {
	FinisherAddr string `protobuf:"bytes,1,opt,name=finisher_addr,json=finisherAddr" json:"finisher_addr,omitempty"`
	Symbol       string `protobuf:"bytes,2,opt,name=symbol" json:"symbol,omitempty"`
	OwnerAddr    string `protobuf:"bytes,3,opt,name=owner_addr,json=ownerAddr" json:"owner_addr,omitempty"`
}

func (m *ReqTokenFinishCreate) Reset()                    { *m = ReqTokenFinishCreate{} }
func (m *ReqTokenFinishCreate) String() string            { return proto.CompactTextString(m) }
func (*ReqTokenFinishCreate) ProtoMessage()               {}
func (*ReqTokenFinishCreate) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{24} }

func (m *ReqTokenFinishCreate) GetFinisherAddr() string {
	if m != nil {
		return m.FinisherAddr
	}
	return ""
}

func (m *ReqTokenFinishCreate) GetSymbol() string {
	if m != nil {
		return m.Symbol
	}
	return ""
}

func (m *ReqTokenFinishCreate) GetOwnerAddr() string {
	if m != nil {
		return m.OwnerAddr
	}
	return ""
}

type ReqTokenRevokeCreate struct {
	RevokerAddr string `protobuf:"bytes,1,opt,name=revoker_addr,json=revokerAddr" json:"revoker_addr,omitempty"`
	Symbol      string `protobuf:"bytes,2,opt,name=symbol" json:"symbol,omitempty"`
	OwnerAddr   string `protobuf:"bytes,3,opt,name=owner_addr,json=ownerAddr" json:"owner_addr,omitempty"`
}

func (m *ReqTokenRevokeCreate) Reset()                    { *m = ReqTokenRevokeCreate{} }
func (m *ReqTokenRevokeCreate) String() string            { return proto.CompactTextString(m) }
func (*ReqTokenRevokeCreate) ProtoMessage()               {}
func (*ReqTokenRevokeCreate) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{25} }

func (m *ReqTokenRevokeCreate) GetRevokerAddr() string {
	if m != nil {
		return m.RevokerAddr
	}
	return ""
}

func (m *ReqTokenRevokeCreate) GetSymbol() string {
	if m != nil {
		return m.Symbol
	}
	return ""
}

func (m *ReqTokenRevokeCreate) GetOwnerAddr() string {
	if m != nil {
		return m.OwnerAddr
	}
	return ""
}

type ReqSellToken struct {
	Sell  *TradeForSell `protobuf:"bytes,1,opt,name=sell" json:"sell,omitempty"`
	Owner string        `protobuf:"bytes,2,opt,name=owner" json:"owner,omitempty"`
}

func (m *ReqSellToken) Reset()                    { *m = ReqSellToken{} }
func (m *ReqSellToken) String() string            { return proto.CompactTextString(m) }
func (*ReqSellToken) ProtoMessage()               {}
func (*ReqSellToken) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{26} }

func (m *ReqSellToken) GetSell() *TradeForSell {
	if m != nil {
		return m.Sell
	}
	return nil
}

func (m *ReqSellToken) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

type ReqBuyToken struct {
	Buy   *TradeForBuy `protobuf:"bytes,1,opt,name=buy" json:"buy,omitempty"`
	Buyer string       `protobuf:"bytes,2,opt,name=buyer" json:"buyer,omitempty"`
}

func (m *ReqBuyToken) Reset()                    { *m = ReqBuyToken{} }
func (m *ReqBuyToken) String() string            { return proto.CompactTextString(m) }
func (*ReqBuyToken) ProtoMessage()               {}
func (*ReqBuyToken) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{27} }

func (m *ReqBuyToken) GetBuy() *TradeForBuy {
	if m != nil {
		return m.Buy
	}
	return nil
}

func (m *ReqBuyToken) GetBuyer() string {
	if m != nil {
		return m.Buyer
	}
	return ""
}

type ReqRevokeSell struct {
	Revoke *TradeForRevokeSell `protobuf:"bytes,1,opt,name=revoke" json:"revoke,omitempty"`
	Owner  string              `protobuf:"bytes,2,opt,name=owner" json:"owner,omitempty"`
}

func (m *ReqRevokeSell) Reset()                    { *m = ReqRevokeSell{} }
func (m *ReqRevokeSell) String() string            { return proto.CompactTextString(m) }
func (*ReqRevokeSell) ProtoMessage()               {}
func (*ReqRevokeSell) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{28} }

func (m *ReqRevokeSell) GetRevoke() *TradeForRevokeSell {
	if m != nil {
		return m.Revoke
	}
	return nil
}

func (m *ReqRevokeSell) GetOwner() string {
	if m != nil {
		return m.Owner
	}
	return ""
}

type ReqModifyConfig struct {
	Key      string `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Op       string `protobuf:"bytes,2,opt,name=op" json:"op,omitempty"`
	Value    string `protobuf:"bytes,3,opt,name=value" json:"value,omitempty"`
	Modifier string `protobuf:"bytes,4,opt,name=modifier" json:"modifier,omitempty"`
}

func (m *ReqModifyConfig) Reset()                    { *m = ReqModifyConfig{} }
func (m *ReqModifyConfig) String() string            { return proto.CompactTextString(m) }
func (*ReqModifyConfig) ProtoMessage()               {}
func (*ReqModifyConfig) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{29} }

func (m *ReqModifyConfig) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *ReqModifyConfig) GetOp() string {
	if m != nil {
		return m.Op
	}
	return ""
}

func (m *ReqModifyConfig) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

func (m *ReqModifyConfig) GetModifier() string {
	if m != nil {
		return m.Modifier
	}
	return ""
}

type ReqSignRawTx struct {
	Addr    string `protobuf:"bytes,1,opt,name=addr" json:"addr,omitempty"`
	Privkey string `protobuf:"bytes,2,opt,name=privkey" json:"privkey,omitempty"`
	TxHex   string `protobuf:"bytes,3,opt,name=txHex" json:"txHex,omitempty"`
	Expire  string `protobuf:"bytes,4,opt,name=expire" json:"expire,omitempty"`
}

func (m *ReqSignRawTx) Reset()                    { *m = ReqSignRawTx{} }
func (m *ReqSignRawTx) String() string            { return proto.CompactTextString(m) }
func (*ReqSignRawTx) ProtoMessage()               {}
func (*ReqSignRawTx) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{30} }

func (m *ReqSignRawTx) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

func (m *ReqSignRawTx) GetPrivkey() string {
	if m != nil {
		return m.Privkey
	}
	return ""
}

func (m *ReqSignRawTx) GetTxHex() string {
	if m != nil {
		return m.TxHex
	}
	return ""
}

func (m *ReqSignRawTx) GetExpire() string {
	if m != nil {
		return m.Expire
	}
	return ""
}

type ReplySignRawTx struct {
	TxHex string `protobuf:"bytes,1,opt,name=txHex" json:"txHex,omitempty"`
}

func (m *ReplySignRawTx) Reset()                    { *m = ReplySignRawTx{} }
func (m *ReplySignRawTx) String() string            { return proto.CompactTextString(m) }
func (*ReplySignRawTx) ProtoMessage()               {}
func (*ReplySignRawTx) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{31} }

func (m *ReplySignRawTx) GetTxHex() string {
	if m != nil {
		return m.TxHex
	}
	return ""
}

func init() {
	proto.RegisterType((*WalletTxDetail)(nil), "types.WalletTxDetail")
	proto.RegisterType((*WalletTxDetails)(nil), "types.WalletTxDetails")
	proto.RegisterType((*WalletAccountStore)(nil), "types.WalletAccountStore")
	proto.RegisterType((*WalletPwHash)(nil), "types.WalletPwHash")
	proto.RegisterType((*WalletStatus)(nil), "types.WalletStatus")
	proto.RegisterType((*WalletAccounts)(nil), "types.WalletAccounts")
	proto.RegisterType((*WalletAccount)(nil), "types.WalletAccount")
	proto.RegisterType((*WalletUnLock)(nil), "types.WalletUnLock")
	proto.RegisterType((*GenSeedLang)(nil), "types.GenSeedLang")
	proto.RegisterType((*GetSeedByPw)(nil), "types.GetSeedByPw")
	proto.RegisterType((*SaveSeedByPw)(nil), "types.SaveSeedByPw")
	proto.RegisterType((*ReplySeed)(nil), "types.ReplySeed")
	proto.RegisterType((*ReqWalletSetPasswd)(nil), "types.ReqWalletSetPasswd")
	proto.RegisterType((*ReqNewAccount)(nil), "types.ReqNewAccount")
	proto.RegisterType((*MinerFlag)(nil), "types.MinerFlag")
	proto.RegisterType((*ReqWalletTransactionList)(nil), "types.ReqWalletTransactionList")
	proto.RegisterType((*ReqWalletImportPrivKey)(nil), "types.ReqWalletImportPrivKey")
	proto.RegisterType((*ReqWalletSendToAddress)(nil), "types.ReqWalletSendToAddress")
	proto.RegisterType((*ReqWalletSetFee)(nil), "types.ReqWalletSetFee")
	proto.RegisterType((*ReqWalletSetLabel)(nil), "types.ReqWalletSetLabel")
	proto.RegisterType((*ReqWalletMergeBalance)(nil), "types.ReqWalletMergeBalance")
	proto.RegisterType((*ReqStr)(nil), "types.ReqStr")
	proto.RegisterType((*ReplyStr)(nil), "types.ReplyStr")
	proto.RegisterType((*ReqTokenPreCreate)(nil), "types.ReqTokenPreCreate")
	proto.RegisterType((*ReqTokenFinishCreate)(nil), "types.ReqTokenFinishCreate")
	proto.RegisterType((*ReqTokenRevokeCreate)(nil), "types.ReqTokenRevokeCreate")
	proto.RegisterType((*ReqSellToken)(nil), "types.ReqSellToken")
	proto.RegisterType((*ReqBuyToken)(nil), "types.ReqBuyToken")
	proto.RegisterType((*ReqRevokeSell)(nil), "types.ReqRevokeSell")
	proto.RegisterType((*ReqModifyConfig)(nil), "types.ReqModifyConfig")
	proto.RegisterType((*ReqSignRawTx)(nil), "types.ReqSignRawTx")
	proto.RegisterType((*ReplySignRawTx)(nil), "types.ReplySignRawTx")
}

func init() { proto.RegisterFile("wallet.proto", fileDescriptor11) }

var fileDescriptor11 = []byte{
	// 1216 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xa4, 0x56, 0xdb, 0x6e, 0x1b, 0x37,
	0x13, 0xc6, 0x4a, 0x96, 0x6d, 0xd1, 0xb2, 0x93, 0x30, 0x07, 0xec, 0x6f, 0xfc, 0x6d, 0x15, 0xb6,
	0x49, 0x5c, 0xa0, 0x30, 0xd0, 0xe4, 0xaa, 0x05, 0x0a, 0xc4, 0x4e, 0xe0, 0x38, 0xa8, 0x9d, 0x1a,
	0x94, 0x8a, 0xf6, 0xae, 0xa0, 0x77, 0xc7, 0x12, 0xeb, 0xd5, 0x72, 0xcd, 0xa5, 0x4e, 0x6f, 0x52,
	0xf4, 0xba, 0x2f, 0xd4, 0x37, 0x2a, 0x38, 0x24, 0xf7, 0xe0, 0x38, 0x17, 0x45, 0xef, 0xf8, 0xcd,
	0xce, 0xf1, 0x9b, 0xe1, 0x70, 0xc9, 0x60, 0x29, 0xb2, 0x0c, 0xcc, 0x61, 0xa1, 0x95, 0x51, 0xb4,
	0x67, 0xd6, 0x05, 0x94, 0xfb, 0x0f, 0x8c, 0x16, 0x79, 0x29, 0x12, 0x23, 0x55, 0xee, 0xbe, 0xec,
	0xdf, 0xbf, 0xcc, 0x54, 0x72, 0x9d, 0x4c, 0x85, 0x0c, 0x92, 0x5d, 0x91, 0x24, 0x6a, 0x9e, 0x7b,
	0xd3, 0xfd, 0x3d, 0x58, 0x41, 0x32, 0x37, 0x4a, 0x7b, 0xfc, 0x30, 0xe0, 0xb1, 0x16, 0x29, 0x38,
	0x21, 0xfb, 0xb3, 0x43, 0xf6, 0x7e, 0xc1, 0x80, 0xe3, 0xd5, 0x5b, 0x30, 0x42, 0x66, 0x94, 0x91,
	0x8e, 0x59, 0xc5, 0xd1, 0x30, 0x3a, 0xd8, 0x79, 0x49, 0x0f, 0x31, 0xfe, 0xe1, 0xb8, 0x0e, 0xcf,
	0x3b, 0x66, 0x45, 0xbf, 0x21, 0x5b, 0x1a, 0x12, 0x90, 0x85, 0x89, 0x3b, 0x2d, 0x45, 0xee, 0xa4,
	0x6f, 0x85, 0x11, 0x3c, 0xa8, 0xd0, 0x27, 0x64, 0x73, 0x0a, 0x72, 0x32, 0x35, 0x71, 0x77, 0x18,
	0x1d, 0x74, 0xb9, 0x47, 0xf4, 0x11, 0xe9, 0xc9, 0x3c, 0x85, 0x55, 0xbc, 0x81, 0x62, 0x07, 0xe8,
	0xff, 0x49, 0x1f, 0x4b, 0x33, 0x72, 0x06, 0x71, 0x0f, 0xbf, 0xd4, 0x02, 0xeb, 0x4b, 0xcc, 0x6c,
	0x95, 0xf1, 0xa6, 0xf3, 0xe5, 0x10, 0xdd, 0x27, 0xdb, 0x57, 0x5a, 0xcd, 0x44, 0x9a, 0xea, 0x78,
	0x6b, 0x18, 0x1d, 0xf4, 0x79, 0x85, 0xad, 0x8d, 0x59, 0x4d, 0x45, 0x39, 0x8d, 0xb7, 0x87, 0xd1,
	0xc1, 0x80, 0x7b, 0x44, 0x3f, 0x27, 0xc4, 0xd5, 0xf4, 0x41, 0xcc, 0x20, 0xee, 0xa3, 0x55, 0x43,
	0xc2, 0x4e, 0xc8, 0xbd, 0x36, 0x37, 0x25, 0x7d, 0x45, 0xfa, 0x26, 0x80, 0x38, 0x1a, 0x76, 0x0f,
	0x76, 0x5e, 0x3e, 0xf6, 0xa5, 0xb7, 0x55, 0x79, 0xad, 0xc7, 0x16, 0x84, 0xba, 0x8f, 0x47, 0xae,
	0x41, 0x23, 0xa3, 0x34, 0xd0, 0x98, 0x6c, 0x15, 0x5a, 0x2e, 0xae, 0x61, 0x8d, 0x64, 0xf7, 0x79,
	0x80, 0x96, 0x97, 0x4c, 0x5c, 0x42, 0x86, 0xdc, 0xf6, 0xb9, 0x03, 0x94, 0x92, 0x0d, 0xac, 0xae,
	0x8b, 0x42, 0x3c, 0x5b, 0xae, 0x2c, 0x2b, 0x23, 0x23, 0x66, 0x05, 0xb2, 0xd8, 0xe7, 0xb5, 0x80,
	0xbd, 0x26, 0x03, 0x17, 0xf7, 0x62, 0x79, 0x6a, 0xeb, 0x7d, 0x42, 0x36, 0x0b, 0x3c, 0x61, 0xc0,
	0x01, 0xf7, 0xc8, 0x66, 0xa2, 0x45, 0x9e, 0x96, 0x46, 0xfb, 0x88, 0x01, 0xb2, 0x3f, 0xa2, 0xe0,
	0x62, 0x64, 0x84, 0x99, 0x97, 0x94, 0x91, 0x81, 0x2c, 0x9d, 0xe4, 0x4c, 0x25, 0xd7, 0xe8, 0x68,
	0x9b, 0xb7, 0x64, 0x4e, 0xe7, 0x68, 0x6e, 0xd4, 0xb9, 0xcc, 0x65, 0x3e, 0x41, 0x9f, 0xa8, 0x53,
	0xcb, 0x6c, 0xe2, 0xb2, 0x3c, 0x15, 0xe5, 0x08, 0x20, 0xc5, 0x8a, 0xb6, 0x79, 0x2d, 0x70, 0x1e,
	0xc6, 0x32, 0xb9, 0xf6, 0x51, 0x36, 0x82, 0x87, 0x5a, 0xc6, 0x5e, 0x87, 0xc1, 0xf5, 0xa4, 0x96,
	0xf4, 0x90, 0x6c, 0xb9, 0xbb, 0x13, 0x3a, 0xf3, 0xa8, 0xd5, 0x19, 0xaf, 0xc7, 0x83, 0x12, 0x7b,
	0x47, 0x76, 0x5b, 0x5f, 0xe8, 0x90, 0x74, 0x45, 0x92, 0xf8, 0xd1, 0xdf, 0xf3, 0xc6, 0xc1, 0xcc,
	0x7e, 0xba, 0xbb, 0x33, 0x6c, 0x1a, 0x48, 0xfa, 0x39, 0x47, 0x02, 0x2c, 0xcf, 0xa2, 0x2c, 0x97,
	0xa9, 0x6f, 0xac, 0x47, 0x96, 0x67, 0xdb, 0x1c, 0x35, 0x77, 0xb7, 0xa6, 0xcb, 0x03, 0xa4, 0xcf,
	0xc9, 0x9e, 0xcb, 0xea, 0x27, 0xed, 0x4a, 0xf4, 0x9c, 0xdc, 0x92, 0xb2, 0xa7, 0x64, 0xe7, 0x1d,
	0xe4, 0x96, 0xa3, 0x33, 0x91, 0x4f, 0xec, 0x48, 0x64, 0x22, 0x9f, 0x60, 0x98, 0x1e, 0xc7, 0x33,
	0x7b, 0x66, 0x55, 0x8c, 0x55, 0x39, 0x5e, 0x5f, 0x2c, 0x3f, 0x95, 0x0b, 0xfb, 0x9e, 0x0c, 0x46,
	0x62, 0x01, 0x95, 0x1e, 0x25, 0x1b, 0xa5, 0xed, 0x85, 0xd3, 0xc2, 0x73, 0xc3, 0xb6, 0xd3, 0xb2,
	0xfd, 0x82, 0xf4, 0x39, 0x14, 0xd9, 0x1a, 0x7b, 0x75, 0x87, 0x21, 0x3b, 0x25, 0x94, 0xc3, 0x8d,
	0x1f, 0x1c, 0x30, 0x17, 0x55, 0xf9, 0x2a, 0x4b, 0x2d, 0x08, 0x03, 0xef, 0xa1, 0xfd, 0x92, 0xc3,
	0x12, 0xbf, 0xf8, 0x01, 0xf4, 0x90, 0x3d, 0x23, 0xbb, 0x1c, 0x6e, 0x3e, 0xc0, 0x32, 0xf4, 0xa8,
	0xea, 0x40, 0xd4, 0xec, 0xc0, 0x77, 0xa4, 0x7f, 0x2e, 0x73, 0xd0, 0x27, 0x99, 0x40, 0x56, 0xae,
	0x32, 0x51, 0xb1, 0x62, 0xcf, 0x38, 0xe2, 0x50, 0x82, 0x5e, 0x40, 0xa0, 0xde, 0x43, 0x76, 0x45,
	0xe2, 0x2a, 0xd7, 0xc6, 0x9a, 0x3b, 0x93, 0x25, 0x2e, 0x2e, 0xbb, 0x44, 0xc6, 0xab, 0x70, 0x61,
	0x1c, 0xb2, 0x49, 0x60, 0x36, 0xe8, 0xab, 0xc7, 0x1d, 0xb0, 0x33, 0x9d, 0x4a, 0x0d, 0x68, 0x8e,
	0xfd, 0xeb, 0xf1, 0x5a, 0xc0, 0x4e, 0xc9, 0x93, 0x2a, 0xce, 0xfb, 0x59, 0xa1, 0xb4, 0xb9, 0xd0,
	0x72, 0xf1, 0x23, 0xac, 0xff, 0xed, 0x22, 0x60, 0x7f, 0x45, 0x0d, 0x57, 0x23, 0xc8, 0xd3, 0xb1,
	0x3a, 0x4a, 0x53, 0x0d, 0x65, 0x89, 0xa5, 0x6b, 0x35, 0x0b, 0xcd, 0xb0, 0x67, 0xba, 0x47, 0x3a,
	0x46, 0x79, 0x0f, 0x1d, 0xa3, 0x1a, 0x1b, 0xb4, 0xdb, 0xda, 0xa0, 0x94, 0x6c, 0xe4, 0xca, 0x80,
	0x5f, 0x23, 0x78, 0xb6, 0xa9, 0xc9, 0x72, 0xac, 0xae, 0x21, 0xc7, 0x4d, 0xbc, 0xcd, 0x03, 0xa4,
	0x43, 0xb2, 0x63, 0xec, 0x61, 0xb4, 0x9e, 0x5d, 0xaa, 0x0c, 0x97, 0x71, 0x9f, 0x37, 0x45, 0xec,
	0x6b, 0x72, 0xaf, 0x39, 0x04, 0x27, 0xd0, 0x5c, 0xde, 0x51, 0x33, 0x34, 0xfb, 0x81, 0x3c, 0x68,
	0xaa, 0x9e, 0xb5, 0xf6, 0x5d, 0xd4, 0xd8, 0x77, 0x77, 0x13, 0xf2, 0x82, 0x3c, 0xae, 0xcc, 0xcf,
	0x41, 0x4f, 0xe0, 0x58, 0x64, 0x22, 0x4f, 0xc0, 0x97, 0x1e, 0x85, 0xd2, 0xd9, 0x90, 0x6c, 0x72,
	0xb8, 0x19, 0x19, 0x7c, 0x12, 0x34, 0x9e, 0xc2, 0xb5, 0x70, 0x88, 0x3d, 0x27, 0xdb, 0x6e, 0xb4,
	0x8d, 0xb6, 0x4f, 0x8a, 0xb6, 0xe7, 0xb2, 0xd2, 0xaa, 0x30, 0xfb, 0x3b, 0xc2, 0x94, 0x91, 0x8b,
	0x0b, 0x0d, 0x6f, 0x34, 0x08, 0x03, 0xf4, 0x29, 0x19, 0x24, 0xf6, 0xa4, 0xf4, 0x6f, 0x8d, 0xd4,
	0x77, 0xbc, 0xcc, 0x36, 0x09, 0x59, 0xb6, 0xaf, 0x4d, 0xc7, 0xb3, 0x2c, 0xdc, 0x9b, 0x56, 0x3a,
	0x1a, 0xdd, 0x6e, 0xf7, 0x08, 0xd7, 0x60, 0x6e, 0xb4, 0x4a, 0xe7, 0x6e, 0xa6, 0x5c, 0x67, 0x5a,
	0x32, 0xfa, 0x19, 0x21, 0x6a, 0x99, 0x83, 0x0f, 0xd8, 0x73, 0x4f, 0x00, 0x4a, 0x8e, 0x3c, 0x61,
	0x46, 0x19, 0x91, 0xf9, 0xd7, 0xd2, 0x01, 0x2b, 0x2d, 0xb4, 0x4c, 0x00, 0x5f, 0xca, 0x2e, 0x77,
	0x80, 0x69, 0xf2, 0x28, 0x94, 0x74, 0x22, 0x73, 0x59, 0x4e, 0x7d, 0x55, 0x5f, 0x92, 0xdd, 0x2b,
	0xc4, 0xd0, 0x2a, 0x6b, 0x10, 0x84, 0x47, 0xfe, 0x8d, 0xf5, 0x35, 0x74, 0x5a, 0x35, 0xb4, 0xf3,
	0xeb, 0xde, 0xca, 0x8f, 0x15, 0x75, 0x4c, 0x0e, 0x0b, 0x75, 0xdd, 0x60, 0x52, 0x23, 0x6e, 0x33,
	0xe9, 0x65, 0xff, 0x25, 0xe2, 0x39, 0x19, 0xd8, 0x19, 0x80, 0x2c, 0x73, 0x83, 0xfc, 0xc2, 0xee,
	0xaf, 0x2c, 0xf3, 0x5b, 0xff, 0x61, 0xfd, 0xc3, 0x93, 0xc2, 0x89, 0xd2, 0x56, 0x8f, 0xa3, 0x82,
	0x25, 0x0d, 0xbd, 0x84, 0xd9, 0x43, 0xc0, 0xde, 0x93, 0x1d, 0x0e, 0x37, 0xc7, 0xf3, 0xb5, 0xf3,
	0xf6, 0x15, 0xe9, 0x5e, 0xce, 0xd7, 0x1f, 0xff, 0x3d, 0xa1, 0xb3, 0xe3, 0xf9, 0x9a, 0xdb, 0xcf,
	0xd6, 0xd5, 0xe5, 0x7c, 0x5d, 0xbb, 0x42, 0xc0, 0x7e, 0xc5, 0x5d, 0xe7, 0x68, 0xb0, 0x71, 0xe9,
	0xb7, 0x76, 0x48, 0x2d, 0xf2, 0xfe, 0xfe, 0x77, 0xcb, 0x5f, 0xad, 0xca, 0xbd, 0xe2, 0x27, 0x92,
	0x04, 0xbc, 0x8a, 0xe7, 0x2a, 0x95, 0x57, 0xeb, 0x37, 0x2a, 0xbf, 0x92, 0x13, 0x7a, 0x9f, 0x74,
	0xeb, 0x85, 0x63, 0x8f, 0xf6, 0xb2, 0xa8, 0x22, 0xec, 0x09, 0x55, 0x58, 0x57, 0x0b, 0x91, 0xcd,
	0xc1, 0x53, 0xe8, 0x80, 0xbd, 0x14, 0x33, 0xeb, 0x47, 0x82, 0xf6, 0xf3, 0x58, 0x61, 0xf6, 0xbb,
	0xa3, 0x56, 0x4e, 0x72, 0x2e, 0x96, 0xe3, 0xd5, 0x9d, 0x37, 0xb8, 0xb1, 0xec, 0x3a, 0x1f, 0x2d,
	0x3b, 0xb3, 0x3a, 0x85, 0x55, 0x88, 0x87, 0xc0, 0x76, 0x19, 0x56, 0x85, 0xd4, 0x61, 0x2f, 0x79,
	0xc4, 0x9e, 0x93, 0x3d, 0x77, 0x51, 0xab, 0x68, 0x95, 0x7d, 0xd4, 0xb0, 0xbf, 0xdc, 0xc4, 0xff,
	0xdc, 0x57, 0xff, 0x04, 0x00, 0x00, 0xff, 0xff, 0xf8, 0xe0, 0x6d, 0x43, 0x57, 0x0b, 0x00, 0x00,
}
