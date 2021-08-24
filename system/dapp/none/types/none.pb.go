// Code generated by protoc-gen-go. DO NOT EDIT.
// source: none.proto

package types

import (
	fmt "fmt"
	math "math"

	types "github.com/33cn/chain33/types"
	proto "github.com/golang/protobuf/proto"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type NoneAction struct {
	// Types that are valid to be assigned to Value:
	//	*NoneAction_CommitDelayTx
	Value                isNoneAction_Value `protobuf_oneof:"value"`
	Ty                   int32              `protobuf:"varint,2,opt,name=Ty,proto3" json:"Ty,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *NoneAction) Reset()         { *m = NoneAction{} }
func (m *NoneAction) String() string { return proto.CompactTextString(m) }
func (*NoneAction) ProtoMessage()    {}
func (*NoneAction) Descriptor() ([]byte, []int) {
	return fileDescriptor_784bcb57d038eb7f, []int{0}
}

func (m *NoneAction) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NoneAction.Unmarshal(m, b)
}
func (m *NoneAction) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NoneAction.Marshal(b, m, deterministic)
}
func (m *NoneAction) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NoneAction.Merge(m, src)
}
func (m *NoneAction) XXX_Size() int {
	return xxx_messageInfo_NoneAction.Size(m)
}
func (m *NoneAction) XXX_DiscardUnknown() {
	xxx_messageInfo_NoneAction.DiscardUnknown(m)
}

var xxx_messageInfo_NoneAction proto.InternalMessageInfo

type isNoneAction_Value interface {
	isNoneAction_Value()
}

type NoneAction_CommitDelayTx struct {
	CommitDelayTx *CommitDelayTx `protobuf:"bytes,1,opt,name=commitDelayTx,proto3,oneof"`
}

func (*NoneAction_CommitDelayTx) isNoneAction_Value() {}

func (m *NoneAction) GetValue() isNoneAction_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *NoneAction) GetCommitDelayTx() *CommitDelayTx {
	if x, ok := m.GetValue().(*NoneAction_CommitDelayTx); ok {
		return x.CommitDelayTx
	}
	return nil
}

func (m *NoneAction) GetTy() int32 {
	if m != nil {
		return m.Ty
	}
	return 0
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*NoneAction) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*NoneAction_CommitDelayTx)(nil),
	}
}

// 提交延时交易类型
type CommitDelayTx struct {
	DelayTx                *types.Transaction `protobuf:"bytes,1,opt,name=delayTx,proto3" json:"delayTx,omitempty"`
	RelativeDelayTime      int64              `protobuf:"varint,2,opt,name=relativeDelayTime,proto3" json:"relativeDelayTime,omitempty"`
	IsBlockHeightDelayTime bool               `protobuf:"varint,3,opt,name=isBlockHeightDelayTime,proto3" json:"isBlockHeightDelayTime,omitempty"`
	XXX_NoUnkeyedLiteral   struct{}           `json:"-"`
	XXX_unrecognized       []byte             `json:"-"`
	XXX_sizecache          int32              `json:"-"`
}

func (m *CommitDelayTx) Reset()         { *m = CommitDelayTx{} }
func (m *CommitDelayTx) String() string { return proto.CompactTextString(m) }
func (*CommitDelayTx) ProtoMessage()    {}
func (*CommitDelayTx) Descriptor() ([]byte, []int) {
	return fileDescriptor_784bcb57d038eb7f, []int{1}
}

func (m *CommitDelayTx) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CommitDelayTx.Unmarshal(m, b)
}
func (m *CommitDelayTx) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CommitDelayTx.Marshal(b, m, deterministic)
}
func (m *CommitDelayTx) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CommitDelayTx.Merge(m, src)
}
func (m *CommitDelayTx) XXX_Size() int {
	return xxx_messageInfo_CommitDelayTx.Size(m)
}
func (m *CommitDelayTx) XXX_DiscardUnknown() {
	xxx_messageInfo_CommitDelayTx.DiscardUnknown(m)
}

var xxx_messageInfo_CommitDelayTx proto.InternalMessageInfo

func (m *CommitDelayTx) GetDelayTx() *types.Transaction {
	if m != nil {
		return m.DelayTx
	}
	return nil
}

func (m *CommitDelayTx) GetRelativeDelayTime() int64 {
	if m != nil {
		return m.RelativeDelayTime
	}
	return 0
}

func (m *CommitDelayTx) GetIsBlockHeightDelayTime() bool {
	if m != nil {
		return m.IsBlockHeightDelayTime
	}
	return false
}

// 提交延时交易回执
type CommitDelayTxLog struct {
	Submitter            string   `protobuf:"bytes,1,opt,name=submitter,proto3" json:"submitter,omitempty"`
	DelayTxHash          string   `protobuf:"bytes,2,opt,name=delayTxHash,proto3" json:"delayTxHash,omitempty"`
	EndDelayTime         int64    `protobuf:"varint,3,opt,name=endDelayTime,proto3" json:"endDelayTime,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CommitDelayTxLog) Reset()         { *m = CommitDelayTxLog{} }
func (m *CommitDelayTxLog) String() string { return proto.CompactTextString(m) }
func (*CommitDelayTxLog) ProtoMessage()    {}
func (*CommitDelayTxLog) Descriptor() ([]byte, []int) {
	return fileDescriptor_784bcb57d038eb7f, []int{2}
}

func (m *CommitDelayTxLog) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CommitDelayTxLog.Unmarshal(m, b)
}
func (m *CommitDelayTxLog) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CommitDelayTxLog.Marshal(b, m, deterministic)
}
func (m *CommitDelayTxLog) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CommitDelayTxLog.Merge(m, src)
}
func (m *CommitDelayTxLog) XXX_Size() int {
	return xxx_messageInfo_CommitDelayTxLog.Size(m)
}
func (m *CommitDelayTxLog) XXX_DiscardUnknown() {
	xxx_messageInfo_CommitDelayTxLog.DiscardUnknown(m)
}

var xxx_messageInfo_CommitDelayTxLog proto.InternalMessageInfo

func (m *CommitDelayTxLog) GetSubmitter() string {
	if m != nil {
		return m.Submitter
	}
	return ""
}

func (m *CommitDelayTxLog) GetDelayTxHash() string {
	if m != nil {
		return m.DelayTxHash
	}
	return ""
}

func (m *CommitDelayTxLog) GetEndDelayTime() int64 {
	if m != nil {
		return m.EndDelayTime
	}
	return 0
}

func init() {
	proto.RegisterType((*NoneAction)(nil), "types.NoneAction")
	proto.RegisterType((*CommitDelayTx)(nil), "types.CommitDelayTx")
	proto.RegisterType((*CommitDelayTxLog)(nil), "types.CommitDelayTxLog")
}

func init() {
	proto.RegisterFile("none.proto", fileDescriptor_784bcb57d038eb7f)
}

var fileDescriptor_784bcb57d038eb7f = []byte{
	// 269 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x91, 0xbf, 0x4b, 0xc3, 0x40,
	0x14, 0xc7, 0x4d, 0x42, 0x6c, 0xf3, 0x6a, 0xc5, 0x3e, 0x44, 0x8a, 0x38, 0x84, 0x4c, 0x19, 0x4a,
	0x04, 0x05, 0x27, 0x17, 0xa3, 0x43, 0x06, 0x71, 0x38, 0x32, 0xb9, 0x5d, 0xd3, 0x47, 0x7b, 0x98,
	0xdc, 0x95, 0xdc, 0x35, 0x98, 0xbf, 0xc7, 0x7f, 0x54, 0xb8, 0x56, 0xd2, 0x43, 0xba, 0x7e, 0x7f,
	0xdc, 0xf7, 0x73, 0x3c, 0x00, 0xa9, 0x24, 0x65, 0xdb, 0x56, 0x19, 0x85, 0xa1, 0xe9, 0xb7, 0xa4,
	0x6f, 0x67, 0xa6, 0xe5, 0x52, 0xf3, 0xca, 0x08, 0x25, 0xf7, 0x4e, 0x52, 0x01, 0x7c, 0x28, 0x49,
	0x2f, 0x56, 0xc3, 0x67, 0x98, 0x56, 0xaa, 0x69, 0x84, 0x79, 0xa3, 0x9a, 0xf7, 0xe5, 0xf7, 0xdc,
	0x8b, 0xbd, 0x74, 0xf2, 0x70, 0x9d, 0xd9, 0x7e, 0xf6, 0x7a, 0xec, 0x15, 0x67, 0xcc, 0x0d, 0xe3,
	0x25, 0xf8, 0x65, 0x3f, 0xf7, 0x63, 0x2f, 0x0d, 0x99, 0x5f, 0xf6, 0xf9, 0x08, 0xc2, 0x8e, 0xd7,
	0x3b, 0x4a, 0x7e, 0x3c, 0x98, 0x3a, 0x5d, 0x5c, 0xc0, 0x68, 0xe5, 0x4c, 0xe0, 0x61, 0xa2, 0x1c,
	0x08, 0xd9, 0x5f, 0x04, 0x17, 0x30, 0x6b, 0xa9, 0xe6, 0x46, 0x74, 0xb4, 0x7f, 0x40, 0x34, 0x64,
	0x77, 0x02, 0xf6, 0xdf, 0xc0, 0x27, 0xb8, 0x11, 0x3a, 0xaf, 0x55, 0xf5, 0x55, 0x90, 0x58, 0x6f,
	0xcc, 0x50, 0x09, 0x62, 0x2f, 0x1d, 0xb3, 0x13, 0x6e, 0xd2, 0xc1, 0x95, 0x03, 0xf9, 0xae, 0xd6,
	0x78, 0x07, 0x91, 0xde, 0x2d, 0x1b, 0x61, 0x0c, 0xb5, 0x96, 0x34, 0x62, 0x83, 0x80, 0x31, 0x4c,
	0x0e, 0x88, 0x05, 0xd7, 0x1b, 0x4b, 0x14, 0xb1, 0x63, 0x09, 0x13, 0xb8, 0x20, 0xb9, 0x72, 0x09,
	0x02, 0xe6, 0x68, 0x39, 0x7c, 0x8e, 0xb3, 0xec, 0xde, 0x7e, 0x7f, 0x79, 0x6e, 0xaf, 0xf2, 0xf8,
	0x1b, 0x00, 0x00, 0xff, 0xff, 0xf9, 0xda, 0xf1, 0xcc, 0xbd, 0x01, 0x00, 0x00,
}
