// Code generated by protoc-gen-go. DO NOT EDIT.
// source: executor.proto

package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Genesis struct {
	Isrun bool `protobuf:"varint,1,opt,name=isrun" json:"isrun,omitempty"`
}

func (m *Genesis) Reset()                    { *m = Genesis{} }
func (m *Genesis) String() string            { return proto.CompactTextString(m) }
func (*Genesis) ProtoMessage()               {}
func (*Genesis) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{0} }

func (m *Genesis) GetIsrun() bool {
	if m != nil {
		return m.Isrun
	}
	return false
}

type ExecTxList struct {
	StateHash  []byte         `protobuf:"bytes,1,opt,name=stateHash,proto3" json:"stateHash,omitempty"`
	Txs        []*Transaction `protobuf:"bytes,2,rep,name=txs" json:"txs,omitempty"`
	BlockTime  int64          `protobuf:"varint,3,opt,name=blockTime" json:"blockTime,omitempty"`
	Height     int64          `protobuf:"varint,4,opt,name=height" json:"height,omitempty"`
	Difficulty uint64         `protobuf:"varint,5,opt,name=difficulty" json:"difficulty,omitempty"`
	IsMempool  bool           `protobuf:"varint,6,opt,name=isMempool" json:"isMempool,omitempty"`
}

func (m *ExecTxList) Reset()                    { *m = ExecTxList{} }
func (m *ExecTxList) String() string            { return proto.CompactTextString(m) }
func (*ExecTxList) ProtoMessage()               {}
func (*ExecTxList) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{1} }

func (m *ExecTxList) GetStateHash() []byte {
	if m != nil {
		return m.StateHash
	}
	return nil
}

func (m *ExecTxList) GetTxs() []*Transaction {
	if m != nil {
		return m.Txs
	}
	return nil
}

func (m *ExecTxList) GetBlockTime() int64 {
	if m != nil {
		return m.BlockTime
	}
	return 0
}

func (m *ExecTxList) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *ExecTxList) GetDifficulty() uint64 {
	if m != nil {
		return m.Difficulty
	}
	return 0
}

func (m *ExecTxList) GetIsMempool() bool {
	if m != nil {
		return m.IsMempool
	}
	return false
}

type Query struct {
	Execer   []byte `protobuf:"bytes,1,opt,name=execer,proto3" json:"execer,omitempty"`
	FuncName string `protobuf:"bytes,2,opt,name=funcName" json:"funcName,omitempty"`
	Payload  []byte `protobuf:"bytes,3,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (m *Query) Reset()                    { *m = Query{} }
func (m *Query) String() string            { return proto.CompactTextString(m) }
func (*Query) ProtoMessage()               {}
func (*Query) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{2} }

func (m *Query) GetExecer() []byte {
	if m != nil {
		return m.Execer
	}
	return nil
}

func (m *Query) GetFuncName() string {
	if m != nil {
		return m.FuncName
	}
	return ""
}

func (m *Query) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

type CreateTxIn struct {
	Execer     []byte `protobuf:"bytes,1,opt,name=execer,proto3" json:"execer,omitempty"`
	ActionName string `protobuf:"bytes,2,opt,name=actionName" json:"actionName,omitempty"`
	Payload    []byte `protobuf:"bytes,3,opt,name=payload,proto3" json:"payload,omitempty"`
}

func (m *CreateTxIn) Reset()                    { *m = CreateTxIn{} }
func (m *CreateTxIn) String() string            { return proto.CompactTextString(m) }
func (*CreateTxIn) ProtoMessage()               {}
func (*CreateTxIn) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{3} }

func (m *CreateTxIn) GetExecer() []byte {
	if m != nil {
		return m.Execer
	}
	return nil
}

func (m *CreateTxIn) GetActionName() string {
	if m != nil {
		return m.ActionName
	}
	return ""
}

func (m *CreateTxIn) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

type Cert struct {
	CertId     []byte `protobuf:"bytes,1,opt,name=certId,proto3" json:"certId,omitempty"`
	CreateTime int64  `protobuf:"varint,2,opt,name=createTime" json:"createTime,omitempty"`
	Key        string `protobuf:"bytes,3,opt,name=key" json:"key,omitempty"`
	Value      []byte `protobuf:"bytes,4,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *Cert) Reset()                    { *m = Cert{} }
func (m *Cert) String() string            { return proto.CompactTextString(m) }
func (*Cert) ProtoMessage()               {}
func (*Cert) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{4} }

func (m *Cert) GetCertId() []byte {
	if m != nil {
		return m.CertId
	}
	return nil
}

func (m *Cert) GetCreateTime() int64 {
	if m != nil {
		return m.CreateTime
	}
	return 0
}

func (m *Cert) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *Cert) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

type CertAction struct {
	// Types that are valid to be assigned to Value:
	//	*CertAction_Nput
	Value isCertAction_Value `protobuf_oneof:"value"`
	Ty    int32              `protobuf:"varint,5,opt,name=ty" json:"ty,omitempty"`
}

func (m *CertAction) Reset()                    { *m = CertAction{} }
func (m *CertAction) String() string            { return proto.CompactTextString(m) }
func (*CertAction) ProtoMessage()               {}
func (*CertAction) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{5} }

type isCertAction_Value interface {
	isCertAction_Value()
}

type CertAction_Nput struct {
	Nput *CertPut `protobuf:"bytes,1,opt,name=nput,oneof"`
}

func (*CertAction_Nput) isCertAction_Value() {}

func (m *CertAction) GetValue() isCertAction_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *CertAction) GetNput() *CertPut {
	if x, ok := m.GetValue().(*CertAction_Nput); ok {
		return x.Nput
	}
	return nil
}

func (m *CertAction) GetTy() int32 {
	if m != nil {
		return m.Ty
	}
	return 0
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*CertAction) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _CertAction_OneofMarshaler, _CertAction_OneofUnmarshaler, _CertAction_OneofSizer, []interface{}{
		(*CertAction_Nput)(nil),
	}
}

func _CertAction_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*CertAction)
	// value
	switch x := m.Value.(type) {
	case *CertAction_Nput:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Nput); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("CertAction.Value has unexpected type %T", x)
	}
	return nil
}

func _CertAction_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*CertAction)
	switch tag {
	case 1: // value.nput
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(CertPut)
		err := b.DecodeMessage(msg)
		m.Value = &CertAction_Nput{msg}
		return true, err
	default:
		return false, nil
	}
}

func _CertAction_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*CertAction)
	// value
	switch x := m.Value.(type) {
	case *CertAction_Nput:
		s := proto.Size(x.Nput)
		n += proto.SizeVarint(1<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type CertPut struct {
	Key   string `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *CertPut) Reset()                    { *m = CertPut{} }
func (m *CertPut) String() string            { return proto.CompactTextString(m) }
func (*CertPut) ProtoMessage()               {}
func (*CertPut) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{6} }

func (m *CertPut) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *CertPut) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

// 配置修改部分
type ArrayConfig struct {
	Value []string `protobuf:"bytes,3,rep,name=value" json:"value,omitempty"`
}

func (m *ArrayConfig) Reset()                    { *m = ArrayConfig{} }
func (m *ArrayConfig) String() string            { return proto.CompactTextString(m) }
func (*ArrayConfig) ProtoMessage()               {}
func (*ArrayConfig) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{7} }

func (m *ArrayConfig) GetValue() []string {
	if m != nil {
		return m.Value
	}
	return nil
}

type StringConfig struct {
	Value string `protobuf:"bytes,3,opt,name=value" json:"value,omitempty"`
}

func (m *StringConfig) Reset()                    { *m = StringConfig{} }
func (m *StringConfig) String() string            { return proto.CompactTextString(m) }
func (*StringConfig) ProtoMessage()               {}
func (*StringConfig) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{8} }

func (m *StringConfig) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

type Int32Config struct {
	Value int32 `protobuf:"varint,3,opt,name=value" json:"value,omitempty"`
}

func (m *Int32Config) Reset()                    { *m = Int32Config{} }
func (m *Int32Config) String() string            { return proto.CompactTextString(m) }
func (*Int32Config) ProtoMessage()               {}
func (*Int32Config) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{9} }

func (m *Int32Config) GetValue() int32 {
	if m != nil {
		return m.Value
	}
	return 0
}

type ConfigItem struct {
	Key  string `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Addr string `protobuf:"bytes,2,opt,name=addr" json:"addr,omitempty"`
	// Types that are valid to be assigned to Value:
	//	*ConfigItem_Arr
	//	*ConfigItem_Str
	//	*ConfigItem_Int
	Value isConfigItem_Value `protobuf_oneof:"value"`
	Ty    int32              `protobuf:"varint,11,opt,name=Ty" json:"Ty,omitempty"`
}

func (m *ConfigItem) Reset()                    { *m = ConfigItem{} }
func (m *ConfigItem) String() string            { return proto.CompactTextString(m) }
func (*ConfigItem) ProtoMessage()               {}
func (*ConfigItem) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{10} }

type isConfigItem_Value interface {
	isConfigItem_Value()
}

type ConfigItem_Arr struct {
	Arr *ArrayConfig `protobuf:"bytes,3,opt,name=arr,oneof"`
}
type ConfigItem_Str struct {
	Str *StringConfig `protobuf:"bytes,4,opt,name=str,oneof"`
}
type ConfigItem_Int struct {
	Int *Int32Config `protobuf:"bytes,5,opt,name=int,oneof"`
}

func (*ConfigItem_Arr) isConfigItem_Value() {}
func (*ConfigItem_Str) isConfigItem_Value() {}
func (*ConfigItem_Int) isConfigItem_Value() {}

func (m *ConfigItem) GetValue() isConfigItem_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *ConfigItem) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *ConfigItem) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

func (m *ConfigItem) GetArr() *ArrayConfig {
	if x, ok := m.GetValue().(*ConfigItem_Arr); ok {
		return x.Arr
	}
	return nil
}

func (m *ConfigItem) GetStr() *StringConfig {
	if x, ok := m.GetValue().(*ConfigItem_Str); ok {
		return x.Str
	}
	return nil
}

func (m *ConfigItem) GetInt() *Int32Config {
	if x, ok := m.GetValue().(*ConfigItem_Int); ok {
		return x.Int
	}
	return nil
}

func (m *ConfigItem) GetTy() int32 {
	if m != nil {
		return m.Ty
	}
	return 0
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*ConfigItem) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _ConfigItem_OneofMarshaler, _ConfigItem_OneofUnmarshaler, _ConfigItem_OneofSizer, []interface{}{
		(*ConfigItem_Arr)(nil),
		(*ConfigItem_Str)(nil),
		(*ConfigItem_Int)(nil),
	}
}

func _ConfigItem_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*ConfigItem)
	// value
	switch x := m.Value.(type) {
	case *ConfigItem_Arr:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Arr); err != nil {
			return err
		}
	case *ConfigItem_Str:
		b.EncodeVarint(4<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Str); err != nil {
			return err
		}
	case *ConfigItem_Int:
		b.EncodeVarint(5<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Int); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("ConfigItem.Value has unexpected type %T", x)
	}
	return nil
}

func _ConfigItem_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*ConfigItem)
	switch tag {
	case 3: // value.arr
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(ArrayConfig)
		err := b.DecodeMessage(msg)
		m.Value = &ConfigItem_Arr{msg}
		return true, err
	case 4: // value.str
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(StringConfig)
		err := b.DecodeMessage(msg)
		m.Value = &ConfigItem_Str{msg}
		return true, err
	case 5: // value.int
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(Int32Config)
		err := b.DecodeMessage(msg)
		m.Value = &ConfigItem_Int{msg}
		return true, err
	default:
		return false, nil
	}
}

func _ConfigItem_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*ConfigItem)
	// value
	switch x := m.Value.(type) {
	case *ConfigItem_Arr:
		s := proto.Size(x.Arr)
		n += proto.SizeVarint(3<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case *ConfigItem_Str:
		s := proto.Size(x.Str)
		n += proto.SizeVarint(4<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case *ConfigItem_Int:
		s := proto.Size(x.Int)
		n += proto.SizeVarint(5<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type ModifyConfig struct {
	Key   string `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	Op    string `protobuf:"bytes,3,opt,name=op" json:"op,omitempty"`
	Addr  string `protobuf:"bytes,4,opt,name=addr" json:"addr,omitempty"`
}

func (m *ModifyConfig) Reset()                    { *m = ModifyConfig{} }
func (m *ModifyConfig) String() string            { return proto.CompactTextString(m) }
func (*ModifyConfig) ProtoMessage()               {}
func (*ModifyConfig) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{11} }

func (m *ModifyConfig) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *ModifyConfig) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

func (m *ModifyConfig) GetOp() string {
	if m != nil {
		return m.Op
	}
	return ""
}

func (m *ModifyConfig) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

type ReceiptConfig struct {
	Prev    *ConfigItem `protobuf:"bytes,1,opt,name=prev" json:"prev,omitempty"`
	Current *ConfigItem `protobuf:"bytes,2,opt,name=current" json:"current,omitempty"`
}

func (m *ReceiptConfig) Reset()                    { *m = ReceiptConfig{} }
func (m *ReceiptConfig) String() string            { return proto.CompactTextString(m) }
func (*ReceiptConfig) ProtoMessage()               {}
func (*ReceiptConfig) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{12} }

func (m *ReceiptConfig) GetPrev() *ConfigItem {
	if m != nil {
		return m.Prev
	}
	return nil
}

func (m *ReceiptConfig) GetCurrent() *ConfigItem {
	if m != nil {
		return m.Current
	}
	return nil
}

type ReplyConfig struct {
	Key   string `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
}

func (m *ReplyConfig) Reset()                    { *m = ReplyConfig{} }
func (m *ReplyConfig) String() string            { return proto.CompactTextString(m) }
func (*ReplyConfig) ProtoMessage()               {}
func (*ReplyConfig) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{13} }

func (m *ReplyConfig) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *ReplyConfig) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

type HistoryCertStore struct {
	Rootcerts         [][]byte `protobuf:"bytes,1,rep,name=rootcerts,proto3" json:"rootcerts,omitempty"`
	IntermediateCerts [][]byte `protobuf:"bytes,2,rep,name=intermediateCerts,proto3" json:"intermediateCerts,omitempty"`
	RevocationList    [][]byte `protobuf:"bytes,3,rep,name=revocationList,proto3" json:"revocationList,omitempty"`
	CurHeigth         int64    `protobuf:"varint,4,opt,name=curHeigth" json:"curHeigth,omitempty"`
	NxtHeight         int64    `protobuf:"varint,5,opt,name=nxtHeight" json:"nxtHeight,omitempty"`
}

func (m *HistoryCertStore) Reset()                    { *m = HistoryCertStore{} }
func (m *HistoryCertStore) String() string            { return proto.CompactTextString(m) }
func (*HistoryCertStore) ProtoMessage()               {}
func (*HistoryCertStore) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{14} }

func (m *HistoryCertStore) GetRootcerts() [][]byte {
	if m != nil {
		return m.Rootcerts
	}
	return nil
}

func (m *HistoryCertStore) GetIntermediateCerts() [][]byte {
	if m != nil {
		return m.IntermediateCerts
	}
	return nil
}

func (m *HistoryCertStore) GetRevocationList() [][]byte {
	if m != nil {
		return m.RevocationList
	}
	return nil
}

func (m *HistoryCertStore) GetCurHeigth() int64 {
	if m != nil {
		return m.CurHeigth
	}
	return 0
}

func (m *HistoryCertStore) GetNxtHeight() int64 {
	if m != nil {
		return m.NxtHeight
	}
	return 0
}

func init() {
	proto.RegisterType((*Genesis)(nil), "types.Genesis")
	proto.RegisterType((*ExecTxList)(nil), "types.ExecTxList")
	proto.RegisterType((*Query)(nil), "types.Query")
	proto.RegisterType((*CreateTxIn)(nil), "types.CreateTxIn")
	proto.RegisterType((*Cert)(nil), "types.Cert")
	proto.RegisterType((*CertAction)(nil), "types.CertAction")
	proto.RegisterType((*CertPut)(nil), "types.CertPut")
	proto.RegisterType((*ArrayConfig)(nil), "types.ArrayConfig")
	proto.RegisterType((*StringConfig)(nil), "types.StringConfig")
	proto.RegisterType((*Int32Config)(nil), "types.Int32Config")
	proto.RegisterType((*ConfigItem)(nil), "types.ConfigItem")
	proto.RegisterType((*ModifyConfig)(nil), "types.ModifyConfig")
	proto.RegisterType((*ReceiptConfig)(nil), "types.ReceiptConfig")
	proto.RegisterType((*ReplyConfig)(nil), "types.ReplyConfig")
	proto.RegisterType((*HistoryCertStore)(nil), "types.HistoryCertStore")
}

func init() { proto.RegisterFile("executor.proto", fileDescriptor5) }

var fileDescriptor5 = []byte{
	// 706 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x54, 0x61, 0x6f, 0xf3, 0x34,
	0x10, 0x5e, 0x9a, 0x76, 0x5d, 0xaf, 0xa5, 0xda, 0x0c, 0x42, 0xd1, 0x84, 0x46, 0x95, 0x95, 0x51,
	0x09, 0x54, 0x44, 0x2b, 0x7e, 0xc0, 0x56, 0x21, 0x5a, 0xc1, 0x10, 0x78, 0xe5, 0xcb, 0x3e, 0x20,
	0x79, 0xe9, 0xb5, 0xb5, 0xd6, 0xda, 0x91, 0x73, 0x99, 0x9a, 0xbf, 0x87, 0xf8, 0x61, 0xc8, 0x4e,
	0xd2, 0x44, 0xac, 0xf0, 0xea, 0xfd, 0x94, 0xdc, 0xdd, 0xa3, 0xe7, 0x39, 0x3f, 0x3e, 0x1f, 0xf4,
	0xf1, 0x80, 0x51, 0x4a, 0xda, 0x8c, 0x63, 0xa3, 0x49, 0xb3, 0x16, 0x65, 0x31, 0x26, 0xd7, 0x57,
	0x64, 0x84, 0x4a, 0x44, 0x44, 0x52, 0xab, 0xbc, 0x12, 0x7e, 0x09, 0xed, 0x9f, 0x50, 0x61, 0x22,
	0x13, 0xf6, 0x19, 0xb4, 0x64, 0x62, 0x52, 0x15, 0x78, 0x03, 0x6f, 0x74, 0xc1, 0xf3, 0x20, 0xfc,
	0xdb, 0x03, 0xf8, 0xf1, 0x80, 0xd1, 0xf2, 0xf0, 0x8b, 0x4c, 0x88, 0x7d, 0x01, 0x9d, 0x84, 0x04,
	0xe1, 0x5c, 0x24, 0x5b, 0x07, 0xec, 0xf1, 0x2a, 0xc1, 0x86, 0xe0, 0xd3, 0x21, 0x09, 0x1a, 0x03,
	0x7f, 0xd4, 0x9d, 0xb0, 0xb1, 0x53, 0x1d, 0x2f, 0x2b, 0x51, 0x6e, 0xcb, 0x96, 0xe3, 0x65, 0xa7,
	0xa3, 0xd7, 0xa5, 0xdc, 0x63, 0xe0, 0x0f, 0xbc, 0x91, 0xcf, 0xab, 0x04, 0xfb, 0x1c, 0xce, 0xb7,
	0x28, 0x37, 0x5b, 0x0a, 0x9a, 0xae, 0x54, 0x44, 0xec, 0x06, 0x60, 0x25, 0xd7, 0x6b, 0x19, 0xa5,
	0x3b, 0xca, 0x82, 0xd6, 0xc0, 0x1b, 0x35, 0x79, 0x2d, 0x63, 0x59, 0x65, 0xf2, 0x88, 0xfb, 0x58,
	0xeb, 0x5d, 0x70, 0xee, 0x8e, 0x50, 0x25, 0xc2, 0x3f, 0xa0, 0xf5, 0x7b, 0x8a, 0x26, 0xb3, 0xf4,
	0xd6, 0x1c, 0x34, 0x45, 0xf7, 0x45, 0xc4, 0xae, 0xe1, 0x62, 0x9d, 0xaa, 0xe8, 0x57, 0xb1, 0xc7,
	0xa0, 0x31, 0xf0, 0x46, 0x1d, 0x7e, 0x8c, 0x59, 0x00, 0xed, 0x58, 0x64, 0x3b, 0x2d, 0x56, 0xae,
	0xdd, 0x1e, 0x2f, 0xc3, 0xf0, 0x4f, 0x80, 0x99, 0x41, 0x41, 0xb8, 0x3c, 0x2c, 0xd4, 0x7f, 0x72,
	0xdf, 0x00, 0xe4, 0xe7, 0xaf, 0xb1, 0xd7, 0x32, 0xff, 0xc3, 0xbf, 0x86, 0xe6, 0x0c, 0x0d, 0x59,
	0xe6, 0x08, 0x0d, 0x2d, 0x56, 0x25, 0x73, 0x1e, 0x59, 0xe6, 0x28, 0xd7, 0x97, 0x05, 0xb3, 0xcf,
	0x6b, 0x19, 0x76, 0x09, 0xfe, 0x2b, 0x66, 0x8e, 0xb5, 0xc3, 0xed, 0xaf, 0xbd, 0xe5, 0x37, 0xb1,
	0x4b, 0xd1, 0xb9, 0xdb, 0xe3, 0x79, 0x10, 0xfe, 0x0c, 0x60, 0x75, 0xee, 0x5d, 0x4f, 0x6c, 0x08,
	0x4d, 0x15, 0xa7, 0xe4, 0xb4, 0xba, 0x93, 0x7e, 0x71, 0x8f, 0x16, 0xf0, 0x5b, 0x4a, 0xf3, 0x33,
	0xee, 0xaa, 0xac, 0x0f, 0x8d, 0xe2, 0x22, 0x5a, 0xbc, 0x41, 0xd9, 0x43, 0xbb, 0x60, 0x0e, 0xbf,
	0x87, 0x76, 0x81, 0x2d, 0xf5, 0xbd, 0x13, 0xfa, 0x8d, 0xba, 0xfe, 0x2d, 0x74, 0xef, 0x8d, 0x11,
	0xd9, 0x4c, 0xab, 0xb5, 0xdc, 0x54, 0x20, 0x7f, 0xe0, 0x8f, 0x3a, 0x25, 0x68, 0x08, 0xbd, 0x27,
	0x32, 0x52, 0x6d, 0xde, 0xa3, 0xbc, 0x0a, 0x75, 0x0b, 0xdd, 0x85, 0xa2, 0xe9, 0xe4, 0x14, 0xa8,
	0x55, 0x82, 0xec, 0x54, 0xe7, 0x80, 0x05, 0xe1, 0xfe, 0x44, 0x9b, 0x0c, 0x9a, 0x62, 0xb5, 0x32,
	0xc5, 0x65, 0xb9, 0x7f, 0x76, 0x07, 0xbe, 0x30, 0xc6, 0x11, 0x55, 0xd3, 0x5d, 0x6b, 0x7b, 0x7e,
	0xc6, 0x2d, 0x80, 0x7d, 0x0d, 0x7e, 0x42, 0xc6, 0x19, 0xdc, 0x9d, 0x7c, 0x5a, 0xe0, 0xea, 0x9d,
	0x5b, 0x60, 0x42, 0x8e, 0x50, 0x2a, 0x72, 0x16, 0x56, 0x84, 0xb5, 0xe6, 0x2d, 0x4e, 0x2a, 0xe7,
	0xf4, 0x32, 0x0b, 0xba, 0xb9, 0xd3, 0xcb, 0x9a, 0xd3, 0xcf, 0xd0, 0x7b, 0xd4, 0x2b, 0xb9, 0x2e,
	0x7d, 0xfb, 0x80, 0xdd, 0xa5, 0x47, 0x96, 0x50, 0xc7, 0x85, 0x6d, 0x0d, 0x1d, 0x1f, 0x4f, 0xdb,
	0xac, 0x4e, 0x1b, 0x46, 0xf0, 0x09, 0xc7, 0x08, 0x65, 0x4c, 0x05, 0xf9, 0x57, 0xd0, 0x8c, 0x0d,
	0xbe, 0x15, 0x53, 0x71, 0x55, 0x4e, 0xc5, 0xd1, 0x45, 0xee, 0xca, 0xec, 0x1b, 0x68, 0x47, 0xa9,
	0x31, 0xa8, 0xc8, 0x69, 0x9e, 0x44, 0x96, 0x88, 0xf0, 0x07, 0xe8, 0x72, 0x8c, 0x77, 0x1f, 0xd9,
	0x7f, 0xf8, 0x97, 0x07, 0x97, 0x73, 0x99, 0x90, 0x36, 0x99, 0x9d, 0xb4, 0x27, 0xd2, 0x06, 0xed,
	0x02, 0x30, 0x5a, 0x93, 0x7d, 0x19, 0x49, 0xe0, 0x0d, 0x7c, 0xbb, 0x9a, 0x8e, 0x09, 0xf6, 0x2d,
	0x5c, 0x49, 0x45, 0x68, 0xf6, 0xb8, 0x92, 0x82, 0x70, 0xe6, 0x50, 0x0d, 0x87, 0x7a, 0x5f, 0x60,
	0x77, 0xd0, 0x37, 0xf8, 0xa6, 0x23, 0x61, 0xdf, 0x83, 0x5d, 0x7c, 0x6e, 0x12, 0x7b, 0xfc, 0x5f,
	0x59, 0xab, 0x19, 0xa5, 0x66, 0x8e, 0x72, 0x43, 0xdb, 0x62, 0x5f, 0x55, 0x09, 0x5b, 0x55, 0x07,
	0x9a, 0xe7, 0xdb, 0xac, 0x95, 0x57, 0x8f, 0x89, 0x87, 0xe1, 0x73, 0xb8, 0x91, 0xb4, 0x13, 0x2f,
	0xe3, 0xe9, 0x74, 0x1c, 0xa9, 0xef, 0xa2, 0xad, 0x90, 0x6a, 0x3a, 0x3d, 0x7e, 0x9d, 0x6b, 0x2f,
	0xe7, 0x6e, 0x4f, 0x4f, 0xff, 0x09, 0x00, 0x00, 0xff, 0xff, 0x8a, 0xff, 0xc3, 0x9d, 0xd3, 0x05,
	0x00, 0x00,
}
