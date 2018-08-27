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
func (*TotalFee) Descriptor() ([]byte, []int) { return fileDescriptor15, []int{0} }

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
func (*ReqGetTotalCoins) Descriptor() ([]byte, []int) { return fileDescriptor15, []int{1} }

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
func (*ReplyGetTotalCoins) Descriptor() ([]byte, []int) { return fileDescriptor15, []int{2} }

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
func (*IterateRangeByStateHash) Descriptor() ([]byte, []int) { return fileDescriptor15, []int{3} }

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
func (*TicketStatistic) Descriptor() ([]byte, []int) { return fileDescriptor15, []int{4} }

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
func (*TicketMinerInfo) Descriptor() ([]byte, []int) { return fileDescriptor15, []int{5} }

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

func init() { proto.RegisterFile("statistic.proto", fileDescriptor15) }

var fileDescriptor15 = []byte{
	// 449 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x93, 0xdf, 0x8b, 0xd3, 0x40,
	0x10, 0xc7, 0xc9, 0xe5, 0xd2, 0x6b, 0x87, 0x83, 0x96, 0xe5, 0xd0, 0x20, 0x22, 0x92, 0xa7, 0xc3,
	0x07, 0x5f, 0x04, 0xdf, 0x35, 0xe0, 0x59, 0x44, 0x84, 0x6d, 0xf1, 0x7d, 0x2f, 0x9d, 0xd3, 0x60,
	0xb2, 0x89, 0xbb, 0x13, 0x69, 0xfe, 0x0d, 0xfd, 0x13, 0xfc, 0x47, 0x65, 0x66, 0xd3, 0x34, 0xbd,
	0xfa, 0xb6, 0xdf, 0xcf, 0x64, 0xbe, 0xf3, 0x63, 0x37, 0xb0, 0xf4, 0x64, 0xa8, 0xf4, 0x54, 0x16,
	0xaf, 0x5b, 0xd7, 0x50, 0xa3, 0x12, 0xea, 0x5b, 0xf4, 0xd9, 0x5b, 0x98, 0x6f, 0x1b, 0x32, 0xd5,
	0x07, 0x44, 0xb5, 0x82, 0xf8, 0x01, 0x31, 0x8d, 0x5e, 0x46, 0xb7, 0xb1, 0xe6, 0xa3, 0x4a, 0xe1,
	0x8a, 0xf6, 0x79, 0xd3, 0x59, 0x4a, 0x2f, 0x84, 0x1e, 0x64, 0xf6, 0x3b, 0x82, 0x95, 0xc6, 0x9f,
	0x77, 0x48, 0x92, 0x9e, 0x37, 0xa5, 0xf5, 0xea, 0x09, 0xcc, 0x7c, 0x5f, 0xdf, 0x37, 0x95, 0x78,
	0x2c, 0xf4, 0xa0, 0xd4, 0x73, 0x58, 0x70, 0x79, 0xfc, 0x68, 0xfc, 0x77, 0x31, 0xba, 0xd6, 0x47,
	0xa0, 0x9e, 0xc1, 0xdc, 0x93, 0x71, 0xf4, 0x09, 0xfb, 0x34, 0x96, 0xe0, 0xa8, 0xd5, 0x0d, 0x24,
	0x85, 0x94, 0xbf, 0x94, 0xf2, 0x41, 0x70, 0x1d, 0xdc, 0x63, 0x81, 0x2e, 0x4d, 0x42, 0x9d, 0xa0,
	0x32, 0x0b, 0x4a, 0x63, 0x5b, 0xf5, 0xa7, 0x5d, 0x8d, 0x1e, 0xd1, 0xd4, 0x63, 0x05, 0xb1, 0xed,
	0xea, 0x61, 0x2c, 0x3e, 0xb2, 0xab, 0xa9, 0xe5, 0xc3, 0x58, 0xe0, 0xa0, 0x78, 0x09, 0x16, 0xf7,
	0xd2, 0xde, 0xa5, 0xb4, 0x77, 0x90, 0x59, 0x07, 0x4f, 0xd7, 0x84, 0xce, 0x10, 0x6a, 0x63, 0xbf,
	0xe1, 0xfb, 0x7e, 0x33, 0x0e, 0x75, 0x32, 0x72, 0xf4, 0x78, 0xe4, 0x1b, 0x48, 0x64, 0xc4, 0x61,
	0x19, 0x41, 0x70, 0x4b, 0x68, 0x77, 0xc3, 0x0e, 0xf8, 0xf8, 0xff, 0xf1, 0xb3, 0x3f, 0x11, 0x2c,
	0xb7, 0x65, 0xf1, 0x03, 0x69, 0x73, 0xb8, 0x54, 0xf5, 0x0a, 0x56, 0x45, 0xe7, 0x1c, 0x5a, 0xfa,
	0xd2, 0xa2, 0xcd, 0x27, 0xf3, 0x9e, 0x71, 0x75, 0x0b, 0x4b, 0xe2, 0xf5, 0x7c, 0x2e, 0x2d, 0xba,
	0xe9, 0xed, 0x3e, 0xc6, 0xec, 0x2a, 0x28, 0x37, 0xb6, 0xa8, 0x30, 0x9f, 0x2c, 0xe7, 0x8c, 0x67,
	0x7f, 0x2f, 0x0e, 0x5d, 0x89, 0xc1, 0xda, 0x3e, 0x34, 0x7c, 0xb5, 0x24, 0x68, 0xbd, 0x1b, 0x9e,
	0xc4, 0xa8, 0xe5, 0xb1, 0x90, 0xa1, 0xce, 0x4b, 0xf1, 0x44, 0x0f, 0x4a, 0xbd, 0x00, 0x68, 0x1d,
	0xfe, 0xda, 0x84, 0x58, 0x2c, 0xb1, 0x09, 0xe1, 0xcd, 0x96, 0xfe, 0x0e, 0x2d, 0xfa, 0xd2, 0xcb,
	0x5e, 0xe6, 0xfa, 0x08, 0x38, 0xbb, 0x70, 0x68, 0x08, 0xb7, 0x65, 0x8d, 0xf2, 0x3c, 0x62, 0x3d,
	0x21, 0x9c, 0x5d, 0x73, 0x7b, 0x12, 0x9e, 0x49, 0xf8, 0x08, 0x38, 0x5a, 0x54, 0x8d, 0x0f, 0xc9,
	0x57, 0x21, 0x3a, 0x02, 0xf6, 0x96, 0x4f, 0xbf, 0x9a, 0xaa, 0xc3, 0x74, 0x1e, 0xbc, 0x8f, 0x44,
	0x65, 0x70, 0x2d, 0xea, 0xdd, 0x6e, 0xe7, 0xd0, 0xfb, 0x74, 0x21, 0x13, 0x9f, 0xb0, 0xfb, 0x99,
	0xfc, 0x7d, 0x6f, 0xfe, 0x05, 0x00, 0x00, 0xff, 0xff, 0xa3, 0xfc, 0x9e, 0x4c, 0x90, 0x03, 0x00,
	0x00,
}
