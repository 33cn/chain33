// Code generated by protoc-gen-go.
// source: common.proto
// DO NOT EDIT!

package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Reply struct {
	IsOk bool   `protobuf:"varint,1,opt,name=isOk" json:"isOk,omitempty"`
	Msg  []byte `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
}

func (m *Reply) Reset()                    { *m = Reply{} }
func (m *Reply) String() string            { return proto.CompactTextString(m) }
func (*Reply) ProtoMessage()               {}
func (*Reply) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{0} }

func (m *Reply) GetIsOk() bool {
	if m != nil {
		return m.IsOk
	}
	return false
}

func (m *Reply) GetMsg() []byte {
	if m != nil {
		return m.Msg
	}
	return nil
}

type ReqString struct {
	Data string `protobuf:"bytes,1,opt,name=data" json:"data,omitempty"`
}

func (m *ReqString) Reset()                    { *m = ReqString{} }
func (m *ReqString) String() string            { return proto.CompactTextString(m) }
func (*ReqString) ProtoMessage()               {}
func (*ReqString) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{1} }

func (m *ReqString) GetData() string {
	if m != nil {
		return m.Data
	}
	return ""
}

type ReplyString struct {
	Data string `protobuf:"bytes,1,opt,name=data" json:"data,omitempty"`
}

func (m *ReplyString) Reset()                    { *m = ReplyString{} }
func (m *ReplyString) String() string            { return proto.CompactTextString(m) }
func (*ReplyString) ProtoMessage()               {}
func (*ReplyString) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{2} }

func (m *ReplyString) GetData() string {
	if m != nil {
		return m.Data
	}
	return ""
}

type ReplyStrings struct {
	Datas []string `protobuf:"bytes,1,rep,name=datas" json:"datas,omitempty"`
}

func (m *ReplyStrings) Reset()                    { *m = ReplyStrings{} }
func (m *ReplyStrings) String() string            { return proto.CompactTextString(m) }
func (*ReplyStrings) ProtoMessage()               {}
func (*ReplyStrings) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{3} }

func (m *ReplyStrings) GetDatas() []string {
	if m != nil {
		return m.Datas
	}
	return nil
}

type ReqInt struct {
	Height int64 `protobuf:"varint,1,opt,name=height" json:"height,omitempty"`
}

func (m *ReqInt) Reset()                    { *m = ReqInt{} }
func (m *ReqInt) String() string            { return proto.CompactTextString(m) }
func (*ReqInt) ProtoMessage()               {}
func (*ReqInt) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{4} }

func (m *ReqInt) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

type Int64 struct {
	Data int64 `protobuf:"varint,1,opt,name=data" json:"data,omitempty"`
}

func (m *Int64) Reset()                    { *m = Int64{} }
func (m *Int64) String() string            { return proto.CompactTextString(m) }
func (*Int64) ProtoMessage()               {}
func (*Int64) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{5} }

func (m *Int64) GetData() int64 {
	if m != nil {
		return m.Data
	}
	return 0
}

type ReqHash struct {
	Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
}

func (m *ReqHash) Reset()                    { *m = ReqHash{} }
func (m *ReqHash) String() string            { return proto.CompactTextString(m) }
func (*ReqHash) ProtoMessage()               {}
func (*ReqHash) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{6} }

func (m *ReqHash) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

type ReplyHash struct {
	Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
}

func (m *ReplyHash) Reset()                    { *m = ReplyHash{} }
func (m *ReplyHash) String() string            { return proto.CompactTextString(m) }
func (*ReplyHash) ProtoMessage()               {}
func (*ReplyHash) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{7} }

func (m *ReplyHash) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

type ReqNil struct {
}

func (m *ReqNil) Reset()                    { *m = ReqNil{} }
func (m *ReqNil) String() string            { return proto.CompactTextString(m) }
func (*ReqNil) ProtoMessage()               {}
func (*ReqNil) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{8} }

type ReqHashes struct {
	Hashes [][]byte `protobuf:"bytes,1,rep,name=hashes,proto3" json:"hashes,omitempty"`
}

func (m *ReqHashes) Reset()                    { *m = ReqHashes{} }
func (m *ReqHashes) String() string            { return proto.CompactTextString(m) }
func (*ReqHashes) ProtoMessage()               {}
func (*ReqHashes) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{9} }

func (m *ReqHashes) GetHashes() [][]byte {
	if m != nil {
		return m.Hashes
	}
	return nil
}

type ReplyHashes struct {
	Hashes [][]byte `protobuf:"bytes,1,rep,name=hashes,proto3" json:"hashes,omitempty"`
}

func (m *ReplyHashes) Reset()                    { *m = ReplyHashes{} }
func (m *ReplyHashes) String() string            { return proto.CompactTextString(m) }
func (*ReplyHashes) ProtoMessage()               {}
func (*ReplyHashes) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{10} }

func (m *ReplyHashes) GetHashes() [][]byte {
	if m != nil {
		return m.Hashes
	}
	return nil
}

type KeyValue struct {
	Key   []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *KeyValue) Reset()                    { *m = KeyValue{} }
func (m *KeyValue) String() string            { return proto.CompactTextString(m) }
func (*KeyValue) ProtoMessage()               {}
func (*KeyValue) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{11} }

func (m *KeyValue) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *KeyValue) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

type TimeStatus struct {
	NtpTime   string `protobuf:"bytes,1,opt,name=ntpTime" json:"ntpTime,omitempty"`
	LocalTime string `protobuf:"bytes,2,opt,name=localTime" json:"localTime,omitempty"`
	Diff      int64  `protobuf:"varint,3,opt,name=diff" json:"diff,omitempty"`
}

func (m *TimeStatus) Reset()                    { *m = TimeStatus{} }
func (m *TimeStatus) String() string            { return proto.CompactTextString(m) }
func (*TimeStatus) ProtoMessage()               {}
func (*TimeStatus) Descriptor() ([]byte, []int) { return fileDescriptor2, []int{12} }

func (m *TimeStatus) GetNtpTime() string {
	if m != nil {
		return m.NtpTime
	}
	return ""
}

func (m *TimeStatus) GetLocalTime() string {
	if m != nil {
		return m.LocalTime
	}
	return ""
}

func (m *TimeStatus) GetDiff() int64 {
	if m != nil {
		return m.Diff
	}
	return 0
}

func init() {
	proto.RegisterType((*Reply)(nil), "types.Reply")
	proto.RegisterType((*ReqString)(nil), "types.ReqString")
	proto.RegisterType((*ReplyString)(nil), "types.ReplyString")
	proto.RegisterType((*ReplyStrings)(nil), "types.ReplyStrings")
	proto.RegisterType((*ReqInt)(nil), "types.ReqInt")
	proto.RegisterType((*Int64)(nil), "types.Int64")
	proto.RegisterType((*ReqHash)(nil), "types.ReqHash")
	proto.RegisterType((*ReplyHash)(nil), "types.ReplyHash")
	proto.RegisterType((*ReqNil)(nil), "types.ReqNil")
	proto.RegisterType((*ReqHashes)(nil), "types.ReqHashes")
	proto.RegisterType((*ReplyHashes)(nil), "types.ReplyHashes")
	proto.RegisterType((*KeyValue)(nil), "types.KeyValue")
	proto.RegisterType((*TimeStatus)(nil), "types.TimeStatus")
}

func init() { proto.RegisterFile("common.proto", fileDescriptor2) }

var fileDescriptor2 = []byte{
	// 309 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x84, 0x52, 0x4f, 0x4b, 0xfc, 0x30,
	0x14, 0xa4, 0xdb, 0x5f, 0x77, 0xb7, 0xef, 0xd7, 0x83, 0x04, 0x91, 0x82, 0x8a, 0x35, 0x2a, 0xf4,
	0xa2, 0x07, 0x15, 0x3f, 0x83, 0x8b, 0xa0, 0x90, 0x15, 0xf1, 0x1a, 0xd7, 0xd7, 0x36, 0x6c, 0xfa,
	0x6f, 0x93, 0x15, 0xfa, 0xed, 0x25, 0xaf, 0x11, 0xf5, 0xa0, 0xde, 0x66, 0xde, 0x4c, 0x32, 0x79,
	0x43, 0x20, 0x59, 0xb5, 0x75, 0xdd, 0x36, 0x17, 0xdd, 0xa6, 0xb5, 0x2d, 0x8b, 0xec, 0xd0, 0xa1,
	0xe1, 0xe7, 0x10, 0x09, 0xec, 0xf4, 0xc0, 0x18, 0xfc, 0x53, 0xe6, 0x61, 0x9d, 0x06, 0x59, 0x90,
	0xcf, 0x05, 0x61, 0xb6, 0x03, 0x61, 0x6d, 0xca, 0x74, 0x92, 0x05, 0x79, 0x22, 0x1c, 0xe4, 0x47,
	0x10, 0x0b, 0xec, 0x97, 0x76, 0xa3, 0x9a, 0xd2, 0x1d, 0x79, 0x95, 0x56, 0xd2, 0x91, 0x58, 0x10,
	0xe6, 0xc7, 0xf0, 0x9f, 0xee, 0xfb, 0xc5, 0x72, 0x0a, 0xc9, 0x17, 0x8b, 0x61, 0xbb, 0x10, 0xb9,
	0xb9, 0x49, 0x83, 0x2c, 0xcc, 0x63, 0x31, 0x12, 0x9e, 0xc1, 0x54, 0x60, 0xbf, 0x68, 0x2c, 0xdb,
	0x83, 0x69, 0x85, 0xaa, 0xac, 0x2c, 0xdd, 0x12, 0x0a, 0xcf, 0xf8, 0x3e, 0x44, 0x8b, 0xc6, 0xde,
	0x5c, 0x7f, 0x0b, 0x09, 0x7d, 0xc8, 0x21, 0xcc, 0x04, 0xf6, 0xb7, 0xd2, 0x54, 0x4e, 0xae, 0xa4,
	0xa9, 0x48, 0x4e, 0x04, 0xe1, 0x71, 0x8f, 0x4e, 0x0f, 0x3f, 0x1a, 0xe6, 0x14, 0x7f, 0xaf, 0x34,
	0x3f, 0xa1, 0x95, 0x9d, 0x11, 0x0d, 0xbd, 0x85, 0x10, 0x3d, 0x36, 0x11, 0x9e, 0xf1, 0x33, 0xbf,
	0xf6, 0x1f, 0xb6, 0x4b, 0x98, 0xdf, 0xe1, 0xf0, 0x24, 0xf5, 0x16, 0x5d, 0xb9, 0x6b, 0x1c, 0x7c,
	0xa8, 0x83, 0xae, 0x88, 0x37, 0x27, 0xf9, 0xc2, 0x47, 0xc2, 0x9f, 0x01, 0x1e, 0x55, 0x8d, 0x4b,
	0x2b, 0xed, 0xd6, 0xb0, 0x14, 0x66, 0x8d, 0xed, 0xdc, 0xc0, 0x77, 0xfa, 0x41, 0xd9, 0x01, 0xc4,
	0xba, 0x5d, 0x49, 0x4d, 0xda, 0x84, 0xb4, 0xcf, 0x01, 0x75, 0xa4, 0x8a, 0x22, 0x0d, 0x7d, 0x47,
	0xaa, 0x28, 0x5e, 0xa6, 0xf4, 0x13, 0xae, 0xde, 0x03, 0x00, 0x00, 0xff, 0xff, 0x1b, 0xfc, 0x61,
	0x41, 0x19, 0x02, 0x00, 0x00,
}
