// Code generated by protoc-gen-go. DO NOT EDIT.
// source: db.proto

package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// merkle avl tree
type LeafNode struct {
	Key    []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value  []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Height int32  `protobuf:"varint,3,opt,name=height" json:"height,omitempty"`
	Size   int32  `protobuf:"varint,4,opt,name=size" json:"size,omitempty"`
}

func (m *LeafNode) Reset()                    { *m = LeafNode{} }
func (m *LeafNode) String() string            { return proto.CompactTextString(m) }
func (*LeafNode) ProtoMessage()               {}
func (*LeafNode) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{0} }

func (m *LeafNode) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *LeafNode) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *LeafNode) GetHeight() int32 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *LeafNode) GetSize() int32 {
	if m != nil {
		return m.Size
	}
	return 0
}

type InnerNode struct {
	LeftHash  []byte `protobuf:"bytes,1,opt,name=leftHash,proto3" json:"leftHash,omitempty"`
	RightHash []byte `protobuf:"bytes,2,opt,name=rightHash,proto3" json:"rightHash,omitempty"`
	Height    int32  `protobuf:"varint,3,opt,name=height" json:"height,omitempty"`
	Size      int32  `protobuf:"varint,4,opt,name=size" json:"size,omitempty"`
}

func (m *InnerNode) Reset()                    { *m = InnerNode{} }
func (m *InnerNode) String() string            { return proto.CompactTextString(m) }
func (*InnerNode) ProtoMessage()               {}
func (*InnerNode) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{1} }

func (m *InnerNode) GetLeftHash() []byte {
	if m != nil {
		return m.LeftHash
	}
	return nil
}

func (m *InnerNode) GetRightHash() []byte {
	if m != nil {
		return m.RightHash
	}
	return nil
}

func (m *InnerNode) GetHeight() int32 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *InnerNode) GetSize() int32 {
	if m != nil {
		return m.Size
	}
	return 0
}

type MAVLProof struct {
	LeafHash   []byte       `protobuf:"bytes,1,opt,name=leafHash,proto3" json:"leafHash,omitempty"`
	InnerNodes []*InnerNode `protobuf:"bytes,2,rep,name=innerNodes" json:"innerNodes,omitempty"`
	RootHash   []byte       `protobuf:"bytes,3,opt,name=rootHash,proto3" json:"rootHash,omitempty"`
}

func (m *MAVLProof) Reset()                    { *m = MAVLProof{} }
func (m *MAVLProof) String() string            { return proto.CompactTextString(m) }
func (*MAVLProof) ProtoMessage()               {}
func (*MAVLProof) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{2} }

func (m *MAVLProof) GetLeafHash() []byte {
	if m != nil {
		return m.LeafHash
	}
	return nil
}

func (m *MAVLProof) GetInnerNodes() []*InnerNode {
	if m != nil {
		return m.InnerNodes
	}
	return nil
}

func (m *MAVLProof) GetRootHash() []byte {
	if m != nil {
		return m.RootHash
	}
	return nil
}

type StoreNode struct {
	Key        []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value      []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	LeftHash   []byte `protobuf:"bytes,3,opt,name=leftHash,proto3" json:"leftHash,omitempty"`
	RightHash  []byte `protobuf:"bytes,4,opt,name=rightHash,proto3" json:"rightHash,omitempty"`
	Height     int32  `protobuf:"varint,5,opt,name=height" json:"height,omitempty"`
	Size       int32  `protobuf:"varint,6,opt,name=size" json:"size,omitempty"`
	ParentHash []byte `protobuf:"bytes,7,opt,name=parentHash,proto3" json:"parentHash,omitempty"`
}

func (m *StoreNode) Reset()                    { *m = StoreNode{} }
func (m *StoreNode) String() string            { return proto.CompactTextString(m) }
func (*StoreNode) ProtoMessage()               {}
func (*StoreNode) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{3} }

func (m *StoreNode) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *StoreNode) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *StoreNode) GetLeftHash() []byte {
	if m != nil {
		return m.LeftHash
	}
	return nil
}

func (m *StoreNode) GetRightHash() []byte {
	if m != nil {
		return m.RightHash
	}
	return nil
}

func (m *StoreNode) GetHeight() int32 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *StoreNode) GetSize() int32 {
	if m != nil {
		return m.Size
	}
	return 0
}

func (m *StoreNode) GetParentHash() []byte {
	if m != nil {
		return m.ParentHash
	}
	return nil
}

type LocalDBSet struct {
	KV []*KeyValue `protobuf:"bytes,2,rep,name=KV" json:"KV,omitempty"`
}

func (m *LocalDBSet) Reset()                    { *m = LocalDBSet{} }
func (m *LocalDBSet) String() string            { return proto.CompactTextString(m) }
func (*LocalDBSet) ProtoMessage()               {}
func (*LocalDBSet) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{4} }

func (m *LocalDBSet) GetKV() []*KeyValue {
	if m != nil {
		return m.KV
	}
	return nil
}

type LocalDBList struct {
	Prefix    []byte `protobuf:"bytes,1,opt,name=prefix,proto3" json:"prefix,omitempty"`
	Key       []byte `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	Direction int32  `protobuf:"varint,3,opt,name=direction" json:"direction,omitempty"`
	Count     int32  `protobuf:"varint,4,opt,name=count" json:"count,omitempty"`
}

func (m *LocalDBList) Reset()                    { *m = LocalDBList{} }
func (m *LocalDBList) String() string            { return proto.CompactTextString(m) }
func (*LocalDBList) ProtoMessage()               {}
func (*LocalDBList) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{5} }

func (m *LocalDBList) GetPrefix() []byte {
	if m != nil {
		return m.Prefix
	}
	return nil
}

func (m *LocalDBList) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *LocalDBList) GetDirection() int32 {
	if m != nil {
		return m.Direction
	}
	return 0
}

func (m *LocalDBList) GetCount() int32 {
	if m != nil {
		return m.Count
	}
	return 0
}

type LocalDBGet struct {
	Keys [][]byte `protobuf:"bytes,2,rep,name=keys,proto3" json:"keys,omitempty"`
}

func (m *LocalDBGet) Reset()                    { *m = LocalDBGet{} }
func (m *LocalDBGet) String() string            { return proto.CompactTextString(m) }
func (*LocalDBGet) ProtoMessage()               {}
func (*LocalDBGet) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{6} }

func (m *LocalDBGet) GetKeys() [][]byte {
	if m != nil {
		return m.Keys
	}
	return nil
}

type LocalReplyValue struct {
	Values [][]byte `protobuf:"bytes,2,rep,name=values,proto3" json:"values,omitempty"`
}

func (m *LocalReplyValue) Reset()                    { *m = LocalReplyValue{} }
func (m *LocalReplyValue) String() string            { return proto.CompactTextString(m) }
func (*LocalReplyValue) ProtoMessage()               {}
func (*LocalReplyValue) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{7} }

func (m *LocalReplyValue) GetValues() [][]byte {
	if m != nil {
		return m.Values
	}
	return nil
}

type StoreSet struct {
	StateHash []byte      `protobuf:"bytes,1,opt,name=stateHash,proto3" json:"stateHash,omitempty"`
	KV        []*KeyValue `protobuf:"bytes,2,rep,name=KV" json:"KV,omitempty"`
	Height    int64       `protobuf:"varint,3,opt,name=height" json:"height,omitempty"`
}

func (m *StoreSet) Reset()                    { *m = StoreSet{} }
func (m *StoreSet) String() string            { return proto.CompactTextString(m) }
func (*StoreSet) ProtoMessage()               {}
func (*StoreSet) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{8} }

func (m *StoreSet) GetStateHash() []byte {
	if m != nil {
		return m.StateHash
	}
	return nil
}

func (m *StoreSet) GetKV() []*KeyValue {
	if m != nil {
		return m.KV
	}
	return nil
}

func (m *StoreSet) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

type StoreDel struct {
	StateHash []byte `protobuf:"bytes,1,opt,name=stateHash,proto3" json:"stateHash,omitempty"`
	Height    int64  `protobuf:"varint,2,opt,name=height" json:"height,omitempty"`
}

func (m *StoreDel) Reset()                    { *m = StoreDel{} }
func (m *StoreDel) String() string            { return proto.CompactTextString(m) }
func (*StoreDel) ProtoMessage()               {}
func (*StoreDel) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{9} }

func (m *StoreDel) GetStateHash() []byte {
	if m != nil {
		return m.StateHash
	}
	return nil
}

func (m *StoreDel) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

type StoreSetWithSync struct {
	Storeset *StoreSet `protobuf:"bytes,1,opt,name=storeset" json:"storeset,omitempty"`
	Sync     bool      `protobuf:"varint,2,opt,name=sync" json:"sync,omitempty"`
}

func (m *StoreSetWithSync) Reset()                    { *m = StoreSetWithSync{} }
func (m *StoreSetWithSync) String() string            { return proto.CompactTextString(m) }
func (*StoreSetWithSync) ProtoMessage()               {}
func (*StoreSetWithSync) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{10} }

func (m *StoreSetWithSync) GetStoreset() *StoreSet {
	if m != nil {
		return m.Storeset
	}
	return nil
}

func (m *StoreSetWithSync) GetSync() bool {
	if m != nil {
		return m.Sync
	}
	return false
}

type StoreGet struct {
	StateHash []byte   `protobuf:"bytes,1,opt,name=stateHash,proto3" json:"stateHash,omitempty"`
	Keys      [][]byte `protobuf:"bytes,2,rep,name=keys,proto3" json:"keys,omitempty"`
}

func (m *StoreGet) Reset()                    { *m = StoreGet{} }
func (m *StoreGet) String() string            { return proto.CompactTextString(m) }
func (*StoreGet) ProtoMessage()               {}
func (*StoreGet) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{11} }

func (m *StoreGet) GetStateHash() []byte {
	if m != nil {
		return m.StateHash
	}
	return nil
}

func (m *StoreGet) GetKeys() [][]byte {
	if m != nil {
		return m.Keys
	}
	return nil
}

type StoreReplyValue struct {
	Values [][]byte `protobuf:"bytes,2,rep,name=values,proto3" json:"values,omitempty"`
}

func (m *StoreReplyValue) Reset()                    { *m = StoreReplyValue{} }
func (m *StoreReplyValue) String() string            { return proto.CompactTextString(m) }
func (*StoreReplyValue) ProtoMessage()               {}
func (*StoreReplyValue) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{12} }

func (m *StoreReplyValue) GetValues() [][]byte {
	if m != nil {
		return m.Values
	}
	return nil
}

type PruneData struct {
	// 对应keyHash下的区块高度
	Height int64 `protobuf:"varint,1,opt,name=height" json:"height,omitempty"`
	// hash+prefix的长度
	Lenth int32 `protobuf:"varint,2,opt,name=lenth" json:"lenth,omitempty"`
}

func (m *PruneData) Reset()                    { *m = PruneData{} }
func (m *PruneData) String() string            { return proto.CompactTextString(m) }
func (*PruneData) ProtoMessage()               {}
func (*PruneData) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{13} }

func (m *PruneData) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *PruneData) GetLenth() int32 {
	if m != nil {
		return m.Lenth
	}
	return 0
}

// 用于存储db Pool数据的Value
type StoreValuePool struct {
	Values [][]byte `protobuf:"bytes,1,rep,name=values,proto3" json:"values,omitempty"`
}

func (m *StoreValuePool) Reset()                    { *m = StoreValuePool{} }
func (m *StoreValuePool) String() string            { return proto.CompactTextString(m) }
func (*StoreValuePool) ProtoMessage()               {}
func (*StoreValuePool) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{14} }

func (m *StoreValuePool) GetValues() [][]byte {
	if m != nil {
		return m.Values
	}
	return nil
}

func init() {
	proto.RegisterType((*LeafNode)(nil), "types.LeafNode")
	proto.RegisterType((*InnerNode)(nil), "types.InnerNode")
	proto.RegisterType((*MAVLProof)(nil), "types.MAVLProof")
	proto.RegisterType((*StoreNode)(nil), "types.StoreNode")
	proto.RegisterType((*LocalDBSet)(nil), "types.LocalDBSet")
	proto.RegisterType((*LocalDBList)(nil), "types.LocalDBList")
	proto.RegisterType((*LocalDBGet)(nil), "types.LocalDBGet")
	proto.RegisterType((*LocalReplyValue)(nil), "types.LocalReplyValue")
	proto.RegisterType((*StoreSet)(nil), "types.StoreSet")
	proto.RegisterType((*StoreDel)(nil), "types.StoreDel")
	proto.RegisterType((*StoreSetWithSync)(nil), "types.StoreSetWithSync")
	proto.RegisterType((*StoreGet)(nil), "types.StoreGet")
	proto.RegisterType((*StoreReplyValue)(nil), "types.StoreReplyValue")
	proto.RegisterType((*PruneData)(nil), "types.PruneData")
	proto.RegisterType((*StoreValuePool)(nil), "types.StoreValuePool")
}

func init() { proto.RegisterFile("db.proto", fileDescriptor3) }

var fileDescriptor3 = []byte{
	// 537 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x94, 0xd1, 0x8f, 0xd2, 0x40,
	0x10, 0xc6, 0xd3, 0x16, 0xb0, 0x9d, 0x23, 0x1e, 0x69, 0x8c, 0x21, 0x17, 0xf4, 0xc8, 0x3e, 0x61,
	0x8c, 0x60, 0xe4, 0xc9, 0xc4, 0x07, 0xbd, 0x90, 0x9c, 0x06, 0x34, 0xa4, 0x24, 0x98, 0xf8, 0x60,
	0xb2, 0x94, 0xe1, 0xda, 0x50, 0x76, 0xb1, 0x5d, 0x8c, 0xf5, 0x3f, 0xf3, 0xbf, 0x33, 0x9d, 0xdd,
	0xd2, 0x9e, 0xf1, 0xc4, 0x7b, 0xdb, 0x19, 0x86, 0xef, 0xfb, 0xe6, 0xb7, 0x0b, 0xe0, 0xae, 0x57,
	0xc3, 0x7d, 0x2a, 0x95, 0xf4, 0x9b, 0x2a, 0xdf, 0x63, 0x76, 0xd1, 0x0e, 0xe5, 0x6e, 0x27, 0x85,
	0x6e, 0xb2, 0xaf, 0xe0, 0xce, 0x90, 0x6f, 0x3e, 0xc9, 0x35, 0xfa, 0x1d, 0x70, 0xb6, 0x98, 0x77,
	0xad, 0xbe, 0x35, 0x68, 0x07, 0xc5, 0xd1, 0x7f, 0x04, 0xcd, 0xef, 0x3c, 0x39, 0x60, 0xd7, 0xa6,
	0x9e, 0x2e, 0xfc, 0xc7, 0xd0, 0x8a, 0x30, 0xbe, 0x89, 0x54, 0xd7, 0xe9, 0x5b, 0x83, 0x66, 0x60,
	0x2a, 0xdf, 0x87, 0x46, 0x16, 0xff, 0xc4, 0x6e, 0x83, 0xba, 0x74, 0x66, 0xdf, 0xc0, 0xfb, 0x20,
	0x04, 0xa6, 0x64, 0x70, 0x01, 0x6e, 0x82, 0x1b, 0xf5, 0x9e, 0x67, 0x91, 0x71, 0x39, 0xd6, 0x7e,
	0x0f, 0xbc, 0xb4, 0x50, 0xa1, 0x0f, 0xb5, 0x5d, 0xd5, 0xb8, 0x97, 0xe5, 0x01, 0xbc, 0x8f, 0xef,
	0x96, 0xb3, 0x79, 0x2a, 0xe5, 0x46, 0x5b, 0xf2, 0xcd, 0x6d, 0x4b, 0x5d, 0xfb, 0x2f, 0x01, 0xe2,
	0x32, 0x5b, 0xd6, 0xb5, 0xfb, 0xce, 0xe0, 0xec, 0x55, 0x67, 0x48, 0x94, 0x86, 0xc7, 0xd0, 0x41,
	0x6d, 0xa6, 0x50, 0x4b, 0xa5, 0xd4, 0x19, 0x1d, 0xad, 0x56, 0xd6, 0xec, 0x97, 0x05, 0xde, 0x42,
	0xc9, 0x14, 0xef, 0xc5, 0xb2, 0x8e, 0xc4, 0xf9, 0x17, 0x92, 0xc6, 0xdd, 0x48, 0x9a, 0x7f, 0x45,
	0xd2, 0xaa, 0x90, 0xf8, 0x4f, 0x01, 0xf6, 0x3c, 0x45, 0xa1, 0xa5, 0x1e, 0x90, 0x54, 0xad, 0xc3,
	0x5e, 0x00, 0xcc, 0x64, 0xc8, 0x93, 0xc9, 0xd5, 0x02, 0x95, 0x7f, 0x09, 0xf6, 0x74, 0x69, 0x78,
	0x9c, 0x1b, 0x1e, 0x53, 0xcc, 0x97, 0x45, 0xe0, 0xc0, 0x9e, 0x2e, 0xd9, 0x16, 0xce, 0xcc, 0xf8,
	0x2c, 0xce, 0x54, 0x91, 0x64, 0x9f, 0xe2, 0x26, 0xfe, 0x61, 0xd6, 0x35, 0x55, 0xc9, 0xc0, 0xae,
	0x18, 0xf4, 0xc0, 0x5b, 0xc7, 0x29, 0x86, 0x2a, 0x96, 0xc2, 0xdc, 0x64, 0xd5, 0x28, 0x08, 0x85,
	0xf2, 0x20, 0x94, 0xb9, 0x4d, 0x5d, 0xb0, 0xfe, 0x31, 0xdb, 0x35, 0xd2, 0x76, 0x5b, 0xcc, 0xf5,
	0x6d, 0xb5, 0x03, 0x3a, 0xb3, 0x67, 0x70, 0x4e, 0x13, 0x01, 0xee, 0x13, 0x9d, 0xb2, 0x88, 0x44,
	0x7c, 0xcb, 0x41, 0x53, 0x31, 0x0e, 0x2e, 0xdd, 0x51, 0xb1, 0x66, 0x0f, 0xbc, 0x4c, 0x71, 0x85,
	0xb5, 0xb7, 0x51, 0x35, 0x4e, 0x42, 0xf8, 0xe3, 0x49, 0x3a, 0x25, 0x7f, 0xf6, 0xd6, 0x58, 0x4c,
	0x30, 0x39, 0x61, 0x51, 0x29, 0xd8, 0xb7, 0x14, 0x16, 0xd0, 0x29, 0x43, 0x7e, 0x8e, 0x55, 0xb4,
	0xc8, 0x45, 0xe8, 0x3f, 0x07, 0x37, 0x2b, 0x7a, 0x19, 0x2a, 0x12, 0xaa, 0x42, 0x95, 0xa3, 0xc1,
	0x71, 0x80, 0x9e, 0x40, 0x2e, 0x42, 0x92, 0x75, 0x03, 0x3a, 0xb3, 0x37, 0x26, 0xd6, 0xf5, 0xc9,
	0xcd, 0xef, 0x40, 0x4c, 0xdf, 0xfe, 0x0f, 0xc4, 0xaf, 0xc1, 0x9b, 0xa7, 0x07, 0x81, 0x13, 0xae,
	0x78, 0x6d, 0x45, 0xab, 0xbe, 0x62, 0x71, 0xd5, 0x09, 0x0a, 0xa5, 0x7f, 0xe9, 0xcd, 0x40, 0x17,
	0x6c, 0x00, 0x0f, 0xc9, 0x85, 0x0c, 0xe6, 0x52, 0x26, 0x35, 0x13, 0xab, 0x6e, 0x72, 0x75, 0xf9,
	0xe5, 0xc9, 0x4d, 0xac, 0xa2, 0xc3, 0x6a, 0x18, 0xca, 0xdd, 0x68, 0x3c, 0x0e, 0xc5, 0x28, 0x8c,
	0x78, 0x2c, 0xc6, 0xe3, 0x11, 0x51, 0x59, 0xb5, 0xe8, 0xef, 0x6d, 0xfc, 0x3b, 0x00, 0x00, 0xff,
	0xff, 0x0e, 0x37, 0xd4, 0x8b, 0xff, 0x04, 0x00, 0x00,
}
