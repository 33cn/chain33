// Code generated by protoc-gen-go.
// source: valnode.proto
// DO NOT EDIT!

package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type ValNode struct {
	PubKey []byte `protobuf:"bytes,1,opt,name=pubKey,proto3" json:"pubKey,omitempty"`
	Power  int64  `protobuf:"varint,2,opt,name=power" json:"power,omitempty"`
}

func (m *ValNode) Reset()                    { *m = ValNode{} }
func (m *ValNode) String() string            { return proto.CompactTextString(m) }
func (*ValNode) ProtoMessage()               {}
func (*ValNode) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

func (m *ValNode) GetPubKey() []byte {
	if m != nil {
		return m.PubKey
	}
	return nil
}

func (m *ValNode) GetPower() int64 {
	if m != nil {
		return m.Power
	}
	return 0
}

type ValNodes struct {
	Nodes []*ValNode `protobuf:"bytes,1,rep,name=nodes" json:"nodes,omitempty"`
}

func (m *ValNodes) Reset()                    { *m = ValNodes{} }
func (m *ValNodes) String() string            { return proto.CompactTextString(m) }
func (*ValNodes) ProtoMessage()               {}
func (*ValNodes) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

func (m *ValNodes) GetNodes() []*ValNode {
	if m != nil {
		return m.Nodes
	}
	return nil
}

type ValNodeAction struct {
	// Types that are valid to be assigned to Value:
	//	*ValNodeAction_Node
	//	*ValNodeAction_BlockInfo
	Value isValNodeAction_Value `protobuf_oneof:"value"`
	Ty    int32                 `protobuf:"varint,3,opt,name=Ty" json:"Ty,omitempty"`
}

func (m *ValNodeAction) Reset()                    { *m = ValNodeAction{} }
func (m *ValNodeAction) String() string            { return proto.CompactTextString(m) }
func (*ValNodeAction) ProtoMessage()               {}
func (*ValNodeAction) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{2} }

type isValNodeAction_Value interface {
	isValNodeAction_Value()
}

type ValNodeAction_Node struct {
	Node *ValNode `protobuf:"bytes,1,opt,name=node,oneof"`
}
type ValNodeAction_BlockInfo struct {
	BlockInfo *TendermintBlockInfo `protobuf:"bytes,2,opt,name=blockInfo,oneof"`
}

func (*ValNodeAction_Node) isValNodeAction_Value()      {}
func (*ValNodeAction_BlockInfo) isValNodeAction_Value() {}

func (m *ValNodeAction) GetValue() isValNodeAction_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *ValNodeAction) GetNode() *ValNode {
	if x, ok := m.GetValue().(*ValNodeAction_Node); ok {
		return x.Node
	}
	return nil
}

func (m *ValNodeAction) GetBlockInfo() *TendermintBlockInfo {
	if x, ok := m.GetValue().(*ValNodeAction_BlockInfo); ok {
		return x.BlockInfo
	}
	return nil
}

func (m *ValNodeAction) GetTy() int32 {
	if m != nil {
		return m.Ty
	}
	return 0
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*ValNodeAction) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _ValNodeAction_OneofMarshaler, _ValNodeAction_OneofUnmarshaler, _ValNodeAction_OneofSizer, []interface{}{
		(*ValNodeAction_Node)(nil),
		(*ValNodeAction_BlockInfo)(nil),
	}
}

func _ValNodeAction_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*ValNodeAction)
	// value
	switch x := m.Value.(type) {
	case *ValNodeAction_Node:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Node); err != nil {
			return err
		}
	case *ValNodeAction_BlockInfo:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.BlockInfo); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("ValNodeAction.Value has unexpected type %T", x)
	}
	return nil
}

func _ValNodeAction_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*ValNodeAction)
	switch tag {
	case 1: // value.node
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(ValNode)
		err := b.DecodeMessage(msg)
		m.Value = &ValNodeAction_Node{msg}
		return true, err
	case 2: // value.blockInfo
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(TendermintBlockInfo)
		err := b.DecodeMessage(msg)
		m.Value = &ValNodeAction_BlockInfo{msg}
		return true, err
	default:
		return false, nil
	}
}

func _ValNodeAction_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*ValNodeAction)
	// value
	switch x := m.Value.(type) {
	case *ValNodeAction_Node:
		s := proto.Size(x.Node)
		n += proto.SizeVarint(1<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case *ValNodeAction_BlockInfo:
		s := proto.Size(x.BlockInfo)
		n += proto.SizeVarint(2<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type ReqNodeInfo struct {
	Height int64 `protobuf:"varint,1,opt,name=height" json:"height,omitempty"`
}

func (m *ReqNodeInfo) Reset()                    { *m = ReqNodeInfo{} }
func (m *ReqNodeInfo) String() string            { return proto.CompactTextString(m) }
func (*ReqNodeInfo) ProtoMessage()               {}
func (*ReqNodeInfo) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{3} }

func (m *ReqNodeInfo) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

type ReqBlockInfo struct {
	Height int64 `protobuf:"varint,1,opt,name=height" json:"height,omitempty"`
}

func (m *ReqBlockInfo) Reset()                    { *m = ReqBlockInfo{} }
func (m *ReqBlockInfo) String() string            { return proto.CompactTextString(m) }
func (*ReqBlockInfo) ProtoMessage()               {}
func (*ReqBlockInfo) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{4} }

func (m *ReqBlockInfo) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func init() {
	proto.RegisterType((*ValNode)(nil), "types.ValNode")
	proto.RegisterType((*ValNodes)(nil), "types.ValNodes")
	proto.RegisterType((*ValNodeAction)(nil), "types.ValNodeAction")
	proto.RegisterType((*ReqNodeInfo)(nil), "types.ReqNodeInfo")
	proto.RegisterType((*ReqBlockInfo)(nil), "types.ReqBlockInfo")
}

func init() { proto.RegisterFile("valnode.proto", fileDescriptor1) }

var fileDescriptor1 = []byte{
	// 261 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x74, 0x90, 0x41, 0x4b, 0xc3, 0x30,
	0x14, 0xc7, 0x97, 0xd6, 0x74, 0xfa, 0xba, 0x0d, 0x09, 0x32, 0xca, 0x4e, 0x25, 0x4c, 0xe9, 0xa9,
	0xc8, 0x3c, 0x08, 0xde, 0xdc, 0x69, 0x22, 0x78, 0x08, 0xc5, 0x7b, 0xbb, 0x3e, 0x5d, 0xb1, 0x26,
	0x5d, 0x9b, 0x4d, 0xfa, 0x15, 0xfc, 0xd4, 0xd2, 0x34, 0xd6, 0x83, 0xec, 0xf8, 0xcf, 0xef, 0xf7,
	0xf2, 0x7f, 0x09, 0x4c, 0x8f, 0x69, 0x29, 0x55, 0x8e, 0x71, 0x55, 0x2b, 0xad, 0x18, 0xd5, 0x6d,
	0x85, 0xcd, 0xe2, 0x52, 0xa3, 0xcc, 0xb1, 0xfe, 0x2c, 0xa4, 0xee, 0x01, 0xbf, 0x87, 0xf1, 0x6b,
	0x5a, 0xbe, 0xa8, 0x1c, 0xd9, 0x1c, 0xbc, 0xea, 0x90, 0x3d, 0x63, 0x1b, 0x90, 0x90, 0x44, 0x13,
	0x61, 0x13, 0xbb, 0x02, 0x5a, 0xa9, 0x2f, 0xac, 0x03, 0x27, 0x24, 0x91, 0x2b, 0xfa, 0xc0, 0x6f,
	0xe1, 0xdc, 0x0e, 0x36, 0x6c, 0x09, 0xb4, 0xeb, 0x6a, 0x02, 0x12, 0xba, 0x91, 0xbf, 0x9a, 0xc5,
	0xa6, 0x2d, 0xb6, 0x5c, 0xf4, 0x90, 0x7f, 0x13, 0x98, 0xda, 0xa3, 0xc7, 0xad, 0x2e, 0x94, 0x64,
	0x4b, 0x38, 0xeb, 0x90, 0xe9, 0xfb, 0x37, 0xb6, 0x19, 0x09, 0x43, 0xd9, 0x03, 0x5c, 0x64, 0xa5,
	0xda, 0x7e, 0x3c, 0xc9, 0x37, 0x65, 0x76, 0xf0, 0x57, 0x0b, 0xab, 0x26, 0xc3, 0x73, 0xd6, 0xbf,
	0xc6, 0x66, 0x24, 0xfe, 0x74, 0x36, 0x03, 0x27, 0x69, 0x03, 0x37, 0x24, 0x11, 0x15, 0x4e, 0xd2,
	0xae, 0xc7, 0x40, 0x8f, 0x69, 0x79, 0x40, 0x7e, 0x0d, 0xbe, 0xc0, 0x7d, 0xd7, 0x63, 0xbc, 0x39,
	0x78, 0x3b, 0x2c, 0xde, 0x77, 0xda, 0xec, 0xe2, 0x0a, 0x9b, 0xf8, 0x0d, 0x4c, 0x04, 0xee, 0x87,
	0xcb, 0x4f, 0x79, 0x99, 0x67, 0x7e, 0xf3, 0xee, 0x27, 0x00, 0x00, 0xff, 0xff, 0x0a, 0x4a, 0x26,
	0xac, 0x77, 0x01, 0x00, 0x00,
}
