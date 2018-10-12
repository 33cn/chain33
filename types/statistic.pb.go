// Code generated by protoc-gen-go. DO NOT EDIT.
// source: statistic.proto

package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// 手续费
type TotalFee struct {
	Fee     int64 `protobuf:"varint,1,opt,name=fee" json:"fee,omitempty"`
	TxCount int64 `protobuf:"varint,2,opt,name=txCount" json:"txCount,omitempty"`
}

func (m *TotalFee) Reset()                    { *m = TotalFee{} }
func (m *TotalFee) String() string            { return proto.CompactTextString(m) }
func (*TotalFee) ProtoMessage()               {}
func (*TotalFee) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{0} }

func (m *TotalFee) GetFee() int64 {
	if m != nil {
		return m.Fee
	}
	return 0
}

func (m *TotalFee) GetTxCount() int64 {
	if m != nil {
		return m.TxCount
	}
	return 0
}

// 查询symbol代币总额
type ReqGetTotalCoins struct {
	Symbol    string `protobuf:"bytes,1,opt,name=symbol" json:"symbol,omitempty"`
	StateHash []byte `protobuf:"bytes,2,opt,name=stateHash,proto3" json:"stateHash,omitempty"`
	StartKey  []byte `protobuf:"bytes,3,opt,name=startKey,proto3" json:"startKey,omitempty"`
	Count     int64  `protobuf:"varint,4,opt,name=count" json:"count,omitempty"`
	Execer    string `protobuf:"bytes,5,opt,name=execer" json:"execer,omitempty"`
}

func (m *ReqGetTotalCoins) Reset()                    { *m = ReqGetTotalCoins{} }
func (m *ReqGetTotalCoins) String() string            { return proto.CompactTextString(m) }
func (*ReqGetTotalCoins) ProtoMessage()               {}
func (*ReqGetTotalCoins) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{1} }

func (m *ReqGetTotalCoins) GetSymbol() string {
	if m != nil {
		return m.Symbol
	}
	return ""
}

func (m *ReqGetTotalCoins) GetStateHash() []byte {
	if m != nil {
		return m.StateHash
	}
	return nil
}

func (m *ReqGetTotalCoins) GetStartKey() []byte {
	if m != nil {
		return m.StartKey
	}
	return nil
}

func (m *ReqGetTotalCoins) GetCount() int64 {
	if m != nil {
		return m.Count
	}
	return 0
}

func (m *ReqGetTotalCoins) GetExecer() string {
	if m != nil {
		return m.Execer
	}
	return ""
}

// 查询symbol代币总额应答
type ReplyGetTotalCoins struct {
	Count   int64  `protobuf:"varint,1,opt,name=count" json:"count,omitempty"`
	Num     int64  `protobuf:"varint,2,opt,name=num" json:"num,omitempty"`
	Amount  int64  `protobuf:"varint,3,opt,name=amount" json:"amount,omitempty"`
	NextKey []byte `protobuf:"bytes,4,opt,name=nextKey,proto3" json:"nextKey,omitempty"`
}

func (m *ReplyGetTotalCoins) Reset()                    { *m = ReplyGetTotalCoins{} }
func (m *ReplyGetTotalCoins) String() string            { return proto.CompactTextString(m) }
func (*ReplyGetTotalCoins) ProtoMessage()               {}
func (*ReplyGetTotalCoins) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{2} }

func (m *ReplyGetTotalCoins) GetCount() int64 {
	if m != nil {
		return m.Count
	}
	return 0
}

func (m *ReplyGetTotalCoins) GetNum() int64 {
	if m != nil {
		return m.Num
	}
	return 0
}

func (m *ReplyGetTotalCoins) GetAmount() int64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

func (m *ReplyGetTotalCoins) GetNextKey() []byte {
	if m != nil {
		return m.NextKey
	}
	return nil
}

// 迭代查询symbol代币总额
type IterateRangeByStateHash struct {
	StateHash []byte `protobuf:"bytes,1,opt,name=stateHash,proto3" json:"stateHash,omitempty"`
	Start     []byte `protobuf:"bytes,2,opt,name=start,proto3" json:"start,omitempty"`
	End       []byte `protobuf:"bytes,3,opt,name=end,proto3" json:"end,omitempty"`
	Count     int64  `protobuf:"varint,4,opt,name=count" json:"count,omitempty"`
}

func (m *IterateRangeByStateHash) Reset()                    { *m = IterateRangeByStateHash{} }
func (m *IterateRangeByStateHash) String() string            { return proto.CompactTextString(m) }
func (*IterateRangeByStateHash) ProtoMessage()               {}
func (*IterateRangeByStateHash) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{3} }

func (m *IterateRangeByStateHash) GetStateHash() []byte {
	if m != nil {
		return m.StateHash
	}
	return nil
}

func (m *IterateRangeByStateHash) GetStart() []byte {
	if m != nil {
		return m.Start
	}
	return nil
}

func (m *IterateRangeByStateHash) GetEnd() []byte {
	if m != nil {
		return m.End
	}
	return nil
}

func (m *IterateRangeByStateHash) GetCount() int64 {
	if m != nil {
		return m.Count
	}
	return 0
}

type TicketStatistic struct {
	// 当前在挖的ticket
	CurrentOpenCount int64 `protobuf:"varint,1,opt,name=currentOpenCount" json:"currentOpenCount,omitempty"`
	// 一共挖到的ticket
	TotalMinerCount int64 `protobuf:"varint,2,opt,name=totalMinerCount" json:"totalMinerCount,omitempty"`
	// 一共取消的ticket
	TotalCancleCount int64 `protobuf:"varint,3,opt,name=totalCancleCount" json:"totalCancleCount,omitempty"`
}

func (m *TicketStatistic) Reset()                    { *m = TicketStatistic{} }
func (m *TicketStatistic) String() string            { return proto.CompactTextString(m) }
func (*TicketStatistic) ProtoMessage()               {}
func (*TicketStatistic) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{4} }

func (m *TicketStatistic) GetCurrentOpenCount() int64 {
	if m != nil {
		return m.CurrentOpenCount
	}
	return 0
}

func (m *TicketStatistic) GetTotalMinerCount() int64 {
	if m != nil {
		return m.TotalMinerCount
	}
	return 0
}

func (m *TicketStatistic) GetTotalCancleCount() int64 {
	if m != nil {
		return m.TotalCancleCount
	}
	return 0
}

type TicketMinerInfo struct {
	TicketId string `protobuf:"bytes,1,opt,name=ticketId" json:"ticketId,omitempty"`
	// 1 -> 可挖矿 2 -> 已挖成功 3-> 已关闭
	Status     int32 `protobuf:"varint,2,opt,name=status" json:"status,omitempty"`
	PrevStatus int32 `protobuf:"varint,3,opt,name=prevStatus" json:"prevStatus,omitempty"`
	// genesis 创建的私钥比较特殊
	IsGenesis bool `protobuf:"varint,4,opt,name=isGenesis" json:"isGenesis,omitempty"`
	// 创建ticket时间
	CreateTime int64 `protobuf:"varint,5,opt,name=createTime" json:"createTime,omitempty"`
	// ticket挖矿时间
	MinerTime int64 `protobuf:"varint,6,opt,name=minerTime" json:"minerTime,omitempty"`
	// 关闭ticket时间
	CloseTime int64 `protobuf:"varint,7,opt,name=closeTime" json:"closeTime,omitempty"`
	// 挖到的币的数目
	MinerValue   int64  `protobuf:"varint,8,opt,name=minerValue" json:"minerValue,omitempty"`
	MinerAddress string `protobuf:"bytes,9,opt,name=minerAddress" json:"minerAddress,omitempty"`
}

func (m *TicketMinerInfo) Reset()                    { *m = TicketMinerInfo{} }
func (m *TicketMinerInfo) String() string            { return proto.CompactTextString(m) }
func (*TicketMinerInfo) ProtoMessage()               {}
func (*TicketMinerInfo) Descriptor() ([]byte, []int) { return fileDescriptor11, []int{5} }

func (m *TicketMinerInfo) GetTicketId() string {
	if m != nil {
		return m.TicketId
	}
	return ""
}

func (m *TicketMinerInfo) GetStatus() int32 {
	if m != nil {
		return m.Status
	}
	return 0
}

func (m *TicketMinerInfo) GetPrevStatus() int32 {
	if m != nil {
		return m.PrevStatus
	}
	return 0
}

func (m *TicketMinerInfo) GetIsGenesis() bool {
	if m != nil {
		return m.IsGenesis
	}
	return false
}

func (m *TicketMinerInfo) GetCreateTime() int64 {
	if m != nil {
		return m.CreateTime
	}
	return 0
}

func (m *TicketMinerInfo) GetMinerTime() int64 {
	if m != nil {
		return m.MinerTime
	}
	return 0
}

func (m *TicketMinerInfo) GetCloseTime() int64 {
	if m != nil {
		return m.CloseTime
	}
	return 0
}

func (m *TicketMinerInfo) GetMinerValue() int64 {
	if m != nil {
		return m.MinerValue
	}
	return 0
}

func (m *TicketMinerInfo) GetMinerAddress() string {
	if m != nil {
		return m.MinerAddress
	}
	return ""
}

func init() {
	proto.RegisterType((*TotalFee)(nil), "types.TotalFee")
	proto.RegisterType((*ReqGetTotalCoins)(nil), "types.ReqGetTotalCoins")
	proto.RegisterType((*ReplyGetTotalCoins)(nil), "types.ReplyGetTotalCoins")
	proto.RegisterType((*IterateRangeByStateHash)(nil), "types.IterateRangeByStateHash")
	proto.RegisterType((*TicketStatistic)(nil), "types.TicketStatistic")
	proto.RegisterType((*TicketMinerInfo)(nil), "types.TicketMinerInfo")
}

func init() { proto.RegisterFile("statistic.proto", fileDescriptor11) }

var fileDescriptor11 = []byte{
	// 477 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x93, 0xdf, 0x8a, 0xd3, 0x40,
	0x14, 0xc6, 0xc9, 0x66, 0xd3, 0x6d, 0x0f, 0x0b, 0x2d, 0xc3, 0xa2, 0x41, 0x44, 0x24, 0x78, 0xb1,
	0x78, 0x51, 0x2f, 0x02, 0xde, 0xbb, 0x01, 0xd7, 0x22, 0x22, 0xa4, 0xc5, 0x0b, 0xef, 0xa6, 0xd3,
	0xb3, 0xbb, 0x83, 0xc9, 0x24, 0xce, 0x9c, 0x48, 0xf3, 0x1a, 0xfa, 0x08, 0xbe, 0xa8, 0xcc, 0x49,
	0x9a, 0xa6, 0xbb, 0x5e, 0x65, 0xbe, 0xdf, 0xc9, 0x7c, 0xe7, 0xcf, 0xcc, 0xc0, 0xdc, 0x91, 0x24,
	0xed, 0x48, 0xab, 0x65, 0x6d, 0x2b, 0xaa, 0x44, 0x44, 0x6d, 0x8d, 0x2e, 0x79, 0x0f, 0xd3, 0x4d,
	0x45, 0xb2, 0xf8, 0x88, 0x28, 0x16, 0x10, 0xde, 0x21, 0xc6, 0xc1, 0xeb, 0xe0, 0x3a, 0xcc, 0xfd,
	0x52, 0xc4, 0x70, 0x41, 0xfb, 0xac, 0x6a, 0x0c, 0xc5, 0x67, 0x4c, 0x0f, 0x32, 0xf9, 0x1d, 0xc0,
	0x22, 0xc7, 0x9f, 0xb7, 0x48, 0xbc, 0x3d, 0xab, 0xb4, 0x71, 0xe2, 0x19, 0x4c, 0x5c, 0x5b, 0x6e,
	0xab, 0x82, 0x3d, 0x66, 0x79, 0xaf, 0xc4, 0x4b, 0x98, 0xf9, 0xf4, 0xf8, 0x49, 0xba, 0x07, 0x36,
	0xba, 0xcc, 0x8f, 0x40, 0xbc, 0x80, 0xa9, 0x23, 0x69, 0xe9, 0x33, 0xb6, 0x71, 0xc8, 0xc1, 0x41,
	0x8b, 0x2b, 0x88, 0x14, 0xa7, 0x3f, 0xe7, 0xf4, 0x9d, 0xf0, 0x79, 0x70, 0x8f, 0x0a, 0x6d, 0x1c,
	0x75, 0x79, 0x3a, 0x95, 0x18, 0x10, 0x39, 0xd6, 0x45, 0x7b, 0x5a, 0xd5, 0xe0, 0x11, 0x8c, 0x3d,
	0x16, 0x10, 0x9a, 0xa6, 0xec, 0xdb, 0xf2, 0x4b, 0xef, 0x2a, 0x4b, 0xfe, 0x31, 0x64, 0xd8, 0x2b,
	0x3f, 0x04, 0x83, 0x7b, 0x2e, 0xef, 0x9c, 0xcb, 0x3b, 0xc8, 0xa4, 0x81, 0xe7, 0x2b, 0x42, 0x2b,
	0x09, 0x73, 0x69, 0xee, 0xf1, 0xa6, 0x5d, 0x0f, 0x4d, 0x9d, 0xb4, 0x1c, 0x3c, 0x6e, 0xf9, 0x0a,
	0x22, 0x6e, 0xb1, 0x1f, 0x46, 0x27, 0x7c, 0x49, 0x68, 0x76, 0xfd, 0x0c, 0xfc, 0xf2, 0xff, 0xed,
	0x27, 0x7f, 0x02, 0x98, 0x6f, 0xb4, 0xfa, 0x81, 0xb4, 0x3e, 0x1c, 0xaa, 0x78, 0x0b, 0x0b, 0xd5,
	0x58, 0x8b, 0x86, 0xbe, 0xd6, 0x68, 0xb2, 0x51, 0xbf, 0x4f, 0xb8, 0xb8, 0x86, 0x39, 0xf9, 0xf1,
	0x7c, 0xd1, 0x06, 0xed, 0xf8, 0x74, 0x1f, 0x63, 0xef, 0xca, 0x28, 0x93, 0x46, 0x15, 0x98, 0x8d,
	0x86, 0xf3, 0x84, 0x27, 0x7f, 0xcf, 0x0e, 0x55, 0xb1, 0xc1, 0xca, 0xdc, 0x55, 0xfe, 0x68, 0x89,
	0xd1, 0x6a, 0xd7, 0x5f, 0x89, 0x41, 0xf3, 0x65, 0x21, 0x49, 0x8d, 0xe3, 0xe4, 0x51, 0xde, 0x2b,
	0xf1, 0x0a, 0xa0, 0xb6, 0xf8, 0x6b, 0xdd, 0xc5, 0x42, 0x8e, 0x8d, 0x88, 0x9f, 0xac, 0x76, 0xb7,
	0x68, 0xd0, 0x69, 0xc7, 0x73, 0x99, 0xe6, 0x47, 0xe0, 0x77, 0x2b, 0x8b, 0x92, 0x70, 0xa3, 0x4b,
	0xe4, 0xeb, 0x11, 0xe6, 0x23, 0xe2, 0x77, 0x97, 0xbe, 0x3c, 0x0e, 0x4f, 0x38, 0x7c, 0x04, 0x3e,
	0xaa, 0x8a, 0xca, 0x75, 0x9b, 0x2f, 0xba, 0xe8, 0x00, 0xbc, 0x37, 0xff, 0xfa, 0x4d, 0x16, 0x0d,
	0xc6, 0xd3, 0xce, 0xfb, 0x48, 0x44, 0x02, 0x97, 0xac, 0x3e, 0xec, 0x76, 0x16, 0x9d, 0x8b, 0x67,
	0xdc, 0xf1, 0x09, 0xbb, 0x79, 0xf3, 0x3d, 0xb9, 0xd7, 0x54, 0xc8, 0xed, 0x32, 0x4d, 0x97, 0xca,
	0xbc, 0x53, 0x0f, 0x52, 0x9b, 0x34, 0x1d, 0xbe, 0xfc, 0x2a, 0xb7, 0x13, 0x7e, 0xa3, 0xe9, 0xbf,
	0x00, 0x00, 0x00, 0xff, 0xff, 0x35, 0x58, 0xba, 0x8f, 0xb6, 0x03, 0x00, 0x00,
}
