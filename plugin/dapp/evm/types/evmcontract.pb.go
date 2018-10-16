// Code generated by protoc-gen-go.
// source: evmcontract.proto
// DO NOT EDIT!

/*
Package types is a generated protocol buffer package.

It is generated from these files:
	evmcontract.proto

It has these top-level messages:
	EVMContractObject
	EVMContractData
	EVMContractState
	EVMContractAction
	ReceiptEVMContract
	EVMStateChangeItem
	EVMContractDataCmd
	EVMContractStateCmd
	ReceiptEVMContractCmd
	CheckEVMAddrReq
	CheckEVMAddrResp
	EstimateEVMGasReq
	EstimateEVMGasResp
	EvmDebugReq
	EvmDebugResp
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

// 合约对象信息
type EVMContractObject struct {
	Addr  string            `protobuf:"bytes,1,opt,name=addr" json:"addr,omitempty"`
	Data  *EVMContractData  `protobuf:"bytes,2,opt,name=data" json:"data,omitempty"`
	State *EVMContractState `protobuf:"bytes,3,opt,name=state" json:"state,omitempty"`
}

func (m *EVMContractObject) Reset()                    { *m = EVMContractObject{} }
func (m *EVMContractObject) String() string            { return proto.CompactTextString(m) }
func (*EVMContractObject) ProtoMessage()               {}
func (*EVMContractObject) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *EVMContractObject) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

func (m *EVMContractObject) GetData() *EVMContractData {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *EVMContractObject) GetState() *EVMContractState {
	if m != nil {
		return m.State
	}
	return nil
}

// 存放合约固定数据
type EVMContractData struct {
	Creator  string `protobuf:"bytes,1,opt,name=creator" json:"creator,omitempty"`
	Name     string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	Alias    string `protobuf:"bytes,3,opt,name=alias" json:"alias,omitempty"`
	Addr     string `protobuf:"bytes,4,opt,name=addr" json:"addr,omitempty"`
	Code     []byte `protobuf:"bytes,5,opt,name=code,proto3" json:"code,omitempty"`
	CodeHash []byte `protobuf:"bytes,6,opt,name=codeHash,proto3" json:"codeHash,omitempty"`
}

func (m *EVMContractData) Reset()                    { *m = EVMContractData{} }
func (m *EVMContractData) String() string            { return proto.CompactTextString(m) }
func (*EVMContractData) ProtoMessage()               {}
func (*EVMContractData) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *EVMContractData) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *EVMContractData) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *EVMContractData) GetAlias() string {
	if m != nil {
		return m.Alias
	}
	return ""
}

func (m *EVMContractData) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

func (m *EVMContractData) GetCode() []byte {
	if m != nil {
		return m.Code
	}
	return nil
}

func (m *EVMContractData) GetCodeHash() []byte {
	if m != nil {
		return m.CodeHash
	}
	return nil
}

// 存放合约变化数据
type EVMContractState struct {
	Nonce       uint64            `protobuf:"varint,1,opt,name=nonce" json:"nonce,omitempty"`
	Suicided    bool              `protobuf:"varint,2,opt,name=suicided" json:"suicided,omitempty"`
	StorageHash []byte            `protobuf:"bytes,3,opt,name=storageHash,proto3" json:"storageHash,omitempty"`
	Storage     map[string][]byte `protobuf:"bytes,4,rep,name=storage" json:"storage,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (m *EVMContractState) Reset()                    { *m = EVMContractState{} }
func (m *EVMContractState) String() string            { return proto.CompactTextString(m) }
func (*EVMContractState) ProtoMessage()               {}
func (*EVMContractState) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *EVMContractState) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *EVMContractState) GetSuicided() bool {
	if m != nil {
		return m.Suicided
	}
	return false
}

func (m *EVMContractState) GetStorageHash() []byte {
	if m != nil {
		return m.StorageHash
	}
	return nil
}

func (m *EVMContractState) GetStorage() map[string][]byte {
	if m != nil {
		return m.Storage
	}
	return nil
}

// 创建/调用合约的请求结构
type EVMContractAction struct {
	// 转账金额
	Amount uint64 `protobuf:"varint,1,opt,name=amount" json:"amount,omitempty"`
	// 消耗限制，默认为Transaction.Fee
	GasLimit uint64 `protobuf:"varint,2,opt,name=gasLimit" json:"gasLimit,omitempty"`
	// gas价格，默认为1
	GasPrice uint32 `protobuf:"varint,3,opt,name=gasPrice" json:"gasPrice,omitempty"`
	// 合约数据
	Code []byte `protobuf:"bytes,4,opt,name=code,proto3" json:"code,omitempty"`
	// 合约别名，方便识别
	Alias string `protobuf:"bytes,5,opt,name=alias" json:"alias,omitempty"`
	// 交易备注
	Note string `protobuf:"bytes,6,opt,name=note" json:"note,omitempty"`
}

func (m *EVMContractAction) Reset()                    { *m = EVMContractAction{} }
func (m *EVMContractAction) String() string            { return proto.CompactTextString(m) }
func (*EVMContractAction) ProtoMessage()               {}
func (*EVMContractAction) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *EVMContractAction) GetAmount() uint64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

func (m *EVMContractAction) GetGasLimit() uint64 {
	if m != nil {
		return m.GasLimit
	}
	return 0
}

func (m *EVMContractAction) GetGasPrice() uint32 {
	if m != nil {
		return m.GasPrice
	}
	return 0
}

func (m *EVMContractAction) GetCode() []byte {
	if m != nil {
		return m.Code
	}
	return nil
}

func (m *EVMContractAction) GetAlias() string {
	if m != nil {
		return m.Alias
	}
	return ""
}

func (m *EVMContractAction) GetNote() string {
	if m != nil {
		return m.Note
	}
	return ""
}

// 合约创建/调用日志
type ReceiptEVMContract struct {
	Caller       string `protobuf:"bytes,1,opt,name=caller" json:"caller,omitempty"`
	ContractName string `protobuf:"bytes,2,opt,name=contractName" json:"contractName,omitempty"`
	ContractAddr string `protobuf:"bytes,3,opt,name=contractAddr" json:"contractAddr,omitempty"`
	UsedGas      uint64 `protobuf:"varint,4,opt,name=usedGas" json:"usedGas,omitempty"`
	// 创建合约返回的代码
	Ret []byte `protobuf:"bytes,5,opt,name=ret,proto3" json:"ret,omitempty"`
}

func (m *ReceiptEVMContract) Reset()                    { *m = ReceiptEVMContract{} }
func (m *ReceiptEVMContract) String() string            { return proto.CompactTextString(m) }
func (*ReceiptEVMContract) ProtoMessage()               {}
func (*ReceiptEVMContract) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *ReceiptEVMContract) GetCaller() string {
	if m != nil {
		return m.Caller
	}
	return ""
}

func (m *ReceiptEVMContract) GetContractName() string {
	if m != nil {
		return m.ContractName
	}
	return ""
}

func (m *ReceiptEVMContract) GetContractAddr() string {
	if m != nil {
		return m.ContractAddr
	}
	return ""
}

func (m *ReceiptEVMContract) GetUsedGas() uint64 {
	if m != nil {
		return m.UsedGas
	}
	return 0
}

func (m *ReceiptEVMContract) GetRet() []byte {
	if m != nil {
		return m.Ret
	}
	return nil
}

// 用于保存EVM只能合约中的状态数据变更
type EVMStateChangeItem struct {
	Key          string `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	PreValue     []byte `protobuf:"bytes,2,opt,name=preValue,proto3" json:"preValue,omitempty"`
	CurrentValue []byte `protobuf:"bytes,3,opt,name=currentValue,proto3" json:"currentValue,omitempty"`
}

func (m *EVMStateChangeItem) Reset()                    { *m = EVMStateChangeItem{} }
func (m *EVMStateChangeItem) String() string            { return proto.CompactTextString(m) }
func (*EVMStateChangeItem) ProtoMessage()               {}
func (*EVMStateChangeItem) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *EVMStateChangeItem) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *EVMStateChangeItem) GetPreValue() []byte {
	if m != nil {
		return m.PreValue
	}
	return nil
}

func (m *EVMStateChangeItem) GetCurrentValue() []byte {
	if m != nil {
		return m.CurrentValue
	}
	return nil
}

// 存放合约固定数据
type EVMContractDataCmd struct {
	Creator  string `protobuf:"bytes,1,opt,name=creator" json:"creator,omitempty"`
	Name     string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	Alias    string `protobuf:"bytes,3,opt,name=alias" json:"alias,omitempty"`
	Addr     string `protobuf:"bytes,4,opt,name=addr" json:"addr,omitempty"`
	Code     string `protobuf:"bytes,5,opt,name=code" json:"code,omitempty"`
	CodeHash string `protobuf:"bytes,6,opt,name=codeHash" json:"codeHash,omitempty"`
}

func (m *EVMContractDataCmd) Reset()                    { *m = EVMContractDataCmd{} }
func (m *EVMContractDataCmd) String() string            { return proto.CompactTextString(m) }
func (*EVMContractDataCmd) ProtoMessage()               {}
func (*EVMContractDataCmd) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *EVMContractDataCmd) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *EVMContractDataCmd) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *EVMContractDataCmd) GetAlias() string {
	if m != nil {
		return m.Alias
	}
	return ""
}

func (m *EVMContractDataCmd) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

func (m *EVMContractDataCmd) GetCode() string {
	if m != nil {
		return m.Code
	}
	return ""
}

func (m *EVMContractDataCmd) GetCodeHash() string {
	if m != nil {
		return m.CodeHash
	}
	return ""
}

// 存放合约变化数据
type EVMContractStateCmd struct {
	Nonce       uint64            `protobuf:"varint,1,opt,name=nonce" json:"nonce,omitempty"`
	Suicided    bool              `protobuf:"varint,2,opt,name=suicided" json:"suicided,omitempty"`
	StorageHash string            `protobuf:"bytes,3,opt,name=storageHash" json:"storageHash,omitempty"`
	Storage     map[string]string `protobuf:"bytes,4,rep,name=storage" json:"storage,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *EVMContractStateCmd) Reset()                    { *m = EVMContractStateCmd{} }
func (m *EVMContractStateCmd) String() string            { return proto.CompactTextString(m) }
func (*EVMContractStateCmd) ProtoMessage()               {}
func (*EVMContractStateCmd) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (m *EVMContractStateCmd) GetNonce() uint64 {
	if m != nil {
		return m.Nonce
	}
	return 0
}

func (m *EVMContractStateCmd) GetSuicided() bool {
	if m != nil {
		return m.Suicided
	}
	return false
}

func (m *EVMContractStateCmd) GetStorageHash() string {
	if m != nil {
		return m.StorageHash
	}
	return ""
}

func (m *EVMContractStateCmd) GetStorage() map[string]string {
	if m != nil {
		return m.Storage
	}
	return nil
}

// 合约创建/调用日志
type ReceiptEVMContractCmd struct {
	Caller string `protobuf:"bytes,1,opt,name=caller" json:"caller,omitempty"`
	// 合约创建时才会返回此内容
	ContractName string `protobuf:"bytes,2,opt,name=contractName" json:"contractName,omitempty"`
	ContractAddr string `protobuf:"bytes,3,opt,name=contractAddr" json:"contractAddr,omitempty"`
	UsedGas      uint64 `protobuf:"varint,4,opt,name=usedGas" json:"usedGas,omitempty"`
	// 创建合约返回的代码
	Ret string `protobuf:"bytes,5,opt,name=ret" json:"ret,omitempty"`
}

func (m *ReceiptEVMContractCmd) Reset()                    { *m = ReceiptEVMContractCmd{} }
func (m *ReceiptEVMContractCmd) String() string            { return proto.CompactTextString(m) }
func (*ReceiptEVMContractCmd) ProtoMessage()               {}
func (*ReceiptEVMContractCmd) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

func (m *ReceiptEVMContractCmd) GetCaller() string {
	if m != nil {
		return m.Caller
	}
	return ""
}

func (m *ReceiptEVMContractCmd) GetContractName() string {
	if m != nil {
		return m.ContractName
	}
	return ""
}

func (m *ReceiptEVMContractCmd) GetContractAddr() string {
	if m != nil {
		return m.ContractAddr
	}
	return ""
}

func (m *ReceiptEVMContractCmd) GetUsedGas() uint64 {
	if m != nil {
		return m.UsedGas
	}
	return 0
}

func (m *ReceiptEVMContractCmd) GetRet() string {
	if m != nil {
		return m.Ret
	}
	return ""
}

type CheckEVMAddrReq struct {
	Addr string `protobuf:"bytes,1,opt,name=addr" json:"addr,omitempty"`
}

func (m *CheckEVMAddrReq) Reset()                    { *m = CheckEVMAddrReq{} }
func (m *CheckEVMAddrReq) String() string            { return proto.CompactTextString(m) }
func (*CheckEVMAddrReq) ProtoMessage()               {}
func (*CheckEVMAddrReq) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

func (m *CheckEVMAddrReq) GetAddr() string {
	if m != nil {
		return m.Addr
	}
	return ""
}

type CheckEVMAddrResp struct {
	Contract     bool   `protobuf:"varint,1,opt,name=contract" json:"contract,omitempty"`
	ContractAddr string `protobuf:"bytes,2,opt,name=contractAddr" json:"contractAddr,omitempty"`
	ContractName string `protobuf:"bytes,3,opt,name=contractName" json:"contractName,omitempty"`
	AliasName    string `protobuf:"bytes,4,opt,name=aliasName" json:"aliasName,omitempty"`
}

func (m *CheckEVMAddrResp) Reset()                    { *m = CheckEVMAddrResp{} }
func (m *CheckEVMAddrResp) String() string            { return proto.CompactTextString(m) }
func (*CheckEVMAddrResp) ProtoMessage()               {}
func (*CheckEVMAddrResp) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{10} }

func (m *CheckEVMAddrResp) GetContract() bool {
	if m != nil {
		return m.Contract
	}
	return false
}

func (m *CheckEVMAddrResp) GetContractAddr() string {
	if m != nil {
		return m.ContractAddr
	}
	return ""
}

func (m *CheckEVMAddrResp) GetContractName() string {
	if m != nil {
		return m.ContractName
	}
	return ""
}

func (m *CheckEVMAddrResp) GetAliasName() string {
	if m != nil {
		return m.AliasName
	}
	return ""
}

type EstimateEVMGasReq struct {
	To     string `protobuf:"bytes,1,opt,name=to" json:"to,omitempty"`
	Code   []byte `protobuf:"bytes,2,opt,name=code,proto3" json:"code,omitempty"`
	Caller string `protobuf:"bytes,3,opt,name=caller" json:"caller,omitempty"`
	Amount uint64 `protobuf:"varint,4,opt,name=amount" json:"amount,omitempty"`
}

func (m *EstimateEVMGasReq) Reset()                    { *m = EstimateEVMGasReq{} }
func (m *EstimateEVMGasReq) String() string            { return proto.CompactTextString(m) }
func (*EstimateEVMGasReq) ProtoMessage()               {}
func (*EstimateEVMGasReq) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{11} }

func (m *EstimateEVMGasReq) GetTo() string {
	if m != nil {
		return m.To
	}
	return ""
}

func (m *EstimateEVMGasReq) GetCode() []byte {
	if m != nil {
		return m.Code
	}
	return nil
}

func (m *EstimateEVMGasReq) GetCaller() string {
	if m != nil {
		return m.Caller
	}
	return ""
}

func (m *EstimateEVMGasReq) GetAmount() uint64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

type EstimateEVMGasResp struct {
	Gas uint64 `protobuf:"varint,1,opt,name=gas" json:"gas,omitempty"`
}

func (m *EstimateEVMGasResp) Reset()                    { *m = EstimateEVMGasResp{} }
func (m *EstimateEVMGasResp) String() string            { return proto.CompactTextString(m) }
func (*EstimateEVMGasResp) ProtoMessage()               {}
func (*EstimateEVMGasResp) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{12} }

func (m *EstimateEVMGasResp) GetGas() uint64 {
	if m != nil {
		return m.Gas
	}
	return 0
}

type EvmDebugReq struct {
	// 0 query, 1 set, -1 clear
	Optype int32 `protobuf:"varint,1,opt,name=optype" json:"optype,omitempty"`
}

func (m *EvmDebugReq) Reset()                    { *m = EvmDebugReq{} }
func (m *EvmDebugReq) String() string            { return proto.CompactTextString(m) }
func (*EvmDebugReq) ProtoMessage()               {}
func (*EvmDebugReq) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{13} }

func (m *EvmDebugReq) GetOptype() int32 {
	if m != nil {
		return m.Optype
	}
	return 0
}

type EvmDebugResp struct {
	DebugStatus string `protobuf:"bytes,1,opt,name=debugStatus" json:"debugStatus,omitempty"`
}

func (m *EvmDebugResp) Reset()                    { *m = EvmDebugResp{} }
func (m *EvmDebugResp) String() string            { return proto.CompactTextString(m) }
func (*EvmDebugResp) ProtoMessage()               {}
func (*EvmDebugResp) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{14} }

func (m *EvmDebugResp) GetDebugStatus() string {
	if m != nil {
		return m.DebugStatus
	}
	return ""
}

func init() {
	proto.RegisterType((*EVMContractObject)(nil), "types.EVMContractObject")
	proto.RegisterType((*EVMContractData)(nil), "types.EVMContractData")
	proto.RegisterType((*EVMContractState)(nil), "types.EVMContractState")
	proto.RegisterType((*EVMContractAction)(nil), "types.EVMContractAction")
	proto.RegisterType((*ReceiptEVMContract)(nil), "types.ReceiptEVMContract")
	proto.RegisterType((*EVMStateChangeItem)(nil), "types.EVMStateChangeItem")
	proto.RegisterType((*EVMContractDataCmd)(nil), "types.EVMContractDataCmd")
	proto.RegisterType((*EVMContractStateCmd)(nil), "types.EVMContractStateCmd")
	proto.RegisterType((*ReceiptEVMContractCmd)(nil), "types.ReceiptEVMContractCmd")
	proto.RegisterType((*CheckEVMAddrReq)(nil), "types.CheckEVMAddrReq")
	proto.RegisterType((*CheckEVMAddrResp)(nil), "types.CheckEVMAddrResp")
	proto.RegisterType((*EstimateEVMGasReq)(nil), "types.EstimateEVMGasReq")
	proto.RegisterType((*EstimateEVMGasResp)(nil), "types.EstimateEVMGasResp")
	proto.RegisterType((*EvmDebugReq)(nil), "types.EvmDebugReq")
	proto.RegisterType((*EvmDebugResp)(nil), "types.EvmDebugResp")
}

func init() { proto.RegisterFile("evmcontract.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 716 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xcc, 0x55, 0x41, 0x6f, 0xd3, 0x4a,
	0x10, 0x96, 0x63, 0x27, 0x8d, 0x27, 0x79, 0xaf, 0xed, 0xbe, 0xf7, 0xfa, 0xac, 0x8a, 0x43, 0xb4,
	0xa2, 0x10, 0x21, 0x11, 0xa1, 0x72, 0x41, 0x3d, 0x20, 0x55, 0xa9, 0x55, 0x90, 0x08, 0xa0, 0xad,
	0x94, 0xfb, 0xd6, 0x5e, 0x52, 0xd3, 0xd8, 0x6b, 0xbc, 0xeb, 0x4a, 0xbd, 0xf2, 0x1b, 0xb8, 0x20,
	0x71, 0x00, 0xf1, 0xcf, 0x38, 0xf1, 0x33, 0xd0, 0xee, 0x3a, 0xce, 0xda, 0x4d, 0x6f, 0x15, 0xe2,
	0x94, 0xf9, 0x66, 0x67, 0x3d, 0xdf, 0xcc, 0x7c, 0xb3, 0x81, 0x5d, 0x76, 0x95, 0x46, 0x3c, 0x93,
	0x05, 0x8d, 0xe4, 0x24, 0x2f, 0xb8, 0xe4, 0xa8, 0x2b, 0xaf, 0x73, 0x26, 0xf0, 0x47, 0x07, 0x76,
	0xc3, 0xf9, 0x6c, 0x5a, 0x1d, 0xbe, 0x39, 0x7f, 0xcf, 0x22, 0x89, 0x10, 0x78, 0x34, 0x8e, 0x8b,
	0xc0, 0x19, 0x39, 0x63, 0x9f, 0x68, 0x1b, 0x3d, 0x02, 0x2f, 0xa6, 0x92, 0x06, 0x9d, 0x91, 0x33,
	0x1e, 0x1c, 0xee, 0x4d, 0xf4, 0xfd, 0x89, 0x75, 0xf7, 0x84, 0x4a, 0x4a, 0x74, 0x0c, 0x7a, 0x0c,
	0x5d, 0x21, 0xa9, 0x64, 0x81, 0xab, 0x83, 0xff, 0xbf, 0x19, 0x7c, 0xa6, 0x8e, 0x89, 0x89, 0xc2,
	0x9f, 0x1d, 0xd8, 0x6e, 0x7d, 0x08, 0x05, 0xb0, 0x15, 0x15, 0x8c, 0x4a, 0xbe, 0x62, 0xb1, 0x82,
	0x8a, 0x5c, 0x46, 0x53, 0xa6, 0x89, 0xf8, 0x44, 0xdb, 0xe8, 0x5f, 0xe8, 0xd2, 0x65, 0x42, 0x85,
	0x4e, 0xe8, 0x13, 0x03, 0xea, 0x32, 0x3c, 0xab, 0x0c, 0x04, 0x5e, 0xc4, 0x63, 0x16, 0x74, 0x47,
	0xce, 0x78, 0x48, 0xb4, 0x8d, 0xf6, 0xa1, 0xaf, 0x7e, 0x5f, 0x50, 0x71, 0x11, 0xf4, 0xb4, 0xbf,
	0xc6, 0xf8, 0x87, 0x03, 0x3b, 0x6d, 0xde, 0x2a, 0x5d, 0xc6, 0xb3, 0x88, 0x69, 0x6a, 0x1e, 0x31,
	0x40, 0x7d, 0x46, 0x94, 0x49, 0x94, 0xc4, 0x2c, 0xd6, 0xe4, 0xfa, 0xa4, 0xc6, 0x68, 0x04, 0x03,
	0x21, 0x79, 0x41, 0x17, 0x26, 0x8b, 0xab, 0xb3, 0xd8, 0x2e, 0xf4, 0x1c, 0xb6, 0x2a, 0x18, 0x78,
	0x23, 0x77, 0x3c, 0x38, 0xbc, 0x7f, 0x4b, 0xd7, 0x26, 0x67, 0x26, 0x2c, 0xcc, 0x64, 0x71, 0x4d,
	0x56, 0x97, 0xf6, 0x8f, 0x60, 0x68, 0x1f, 0xa0, 0x1d, 0x70, 0x2f, 0xd9, 0x75, 0xd5, 0x3c, 0x65,
	0x2a, 0xd6, 0x57, 0x74, 0x59, 0x9a, 0xce, 0x0d, 0x89, 0x01, 0x47, 0x9d, 0x67, 0x0e, 0xfe, 0xd6,
	0x54, 0xc1, 0x71, 0x24, 0x13, 0x9e, 0xa1, 0x3d, 0xe8, 0xd1, 0x94, 0x97, 0x99, 0xac, 0xca, 0xac,
	0x90, 0xaa, 0x73, 0x41, 0xc5, 0xab, 0x24, 0x4d, 0xa4, 0xfe, 0x94, 0x47, 0x6a, 0x5c, 0x9d, 0xbd,
	0x2d, 0x92, 0xc8, 0x0c, 0xff, 0x2f, 0x52, 0xe3, 0xba, 0xf5, 0x9e, 0xd5, 0xfa, 0x7a, 0x70, 0xdd,
	0xd6, 0xe0, 0x32, 0x2e, 0x99, 0x1e, 0x86, 0x1a, 0x31, 0x97, 0x0c, 0x7f, 0x75, 0x00, 0x11, 0x16,
	0xb1, 0x24, 0x97, 0x16, 0x55, 0x45, 0x32, 0xa2, 0xcb, 0x25, 0x5b, 0xc9, 0xa4, 0x42, 0x08, 0xc3,
	0x70, 0xa5, 0xf8, 0xd7, 0x6b, 0xb5, 0x34, 0x7c, 0x76, 0xcc, 0xb1, 0xd2, 0x89, 0xdb, 0x8c, 0x51,
	0x3e, 0xa5, 0xc3, 0x52, 0xb0, 0xf8, 0x94, 0x0a, 0xcd, 0xdb, 0x23, 0x2b, 0xa8, 0x1a, 0x5c, 0x30,
	0x59, 0x09, 0x49, 0x99, 0xf8, 0x1d, 0xa0, 0x70, 0x3e, 0xd3, 0x43, 0x9a, 0x5e, 0xd0, 0x6c, 0xc1,
	0x5e, 0x4a, 0x96, 0x6e, 0x18, 0xc4, 0x3e, 0xf4, 0xf3, 0x82, 0xcd, 0xad, 0x59, 0xd4, 0x58, 0x73,
	0x2a, 0x8b, 0x82, 0x65, 0xd2, 0x9c, 0x1b, 0xa5, 0x34, 0x7c, 0xf8, 0x8b, 0xa3, 0x13, 0xd9, 0xfb,
	0x32, 0x4d, 0xe3, 0xdf, 0xb2, 0x32, 0xfe, 0x2d, 0x2b, 0xe3, 0x5b, 0x2b, 0xf3, 0xd3, 0x81, 0x7f,
	0xda, 0xa2, 0x55, 0xfc, 0xee, 0x64, 0x6b, 0xfc, 0xe6, 0xd6, 0x1c, 0xb7, 0xb7, 0xe6, 0xe1, 0x2d,
	0x5b, 0x33, 0x4d, 0xe3, 0xbb, 0x59, 0x1c, 0xdf, 0x5e, 0x9c, 0xef, 0x0e, 0xfc, 0x77, 0x53, 0x94,
	0xaa, 0xd8, 0x3f, 0x42, 0x97, 0xbe, 0xd1, 0xe5, 0x01, 0x6c, 0x4f, 0x2f, 0x58, 0x74, 0x19, 0xce,
	0x67, 0xea, 0x2e, 0x61, 0x1f, 0x36, 0xbd, 0xf0, 0xf8, 0x93, 0x03, 0x3b, 0xcd, 0x38, 0x91, 0x9b,
	0x41, 0x9b, 0xbc, 0x3a, 0xb8, 0x4f, 0x6a, 0x7c, 0x83, 0x67, 0x67, 0x03, 0xcf, 0x76, 0xbd, 0xee,
	0x86, 0x7a, 0xef, 0x81, 0xaf, 0xd5, 0xa7, 0x03, 0x8c, 0xf2, 0xd6, 0x0e, 0xbc, 0x80, 0xdd, 0x50,
	0xc8, 0x24, 0xa5, 0x92, 0x85, 0xf3, 0xd9, 0x29, 0x15, 0x8a, 0xff, 0xdf, 0xd0, 0x91, 0xbc, 0x62,
	0xdf, 0x91, 0xbc, 0xd6, 0x68, 0xc7, 0x7a, 0x5b, 0xd6, 0x23, 0x70, 0x1b, 0x23, 0x58, 0xbf, 0x6b,
	0x9e, 0xfd, 0xae, 0xe1, 0x07, 0x80, 0xda, 0x89, 0x44, 0xae, 0xda, 0xb9, 0xa0, 0xa2, 0xd2, 0xac,
	0x32, 0xf1, 0x01, 0x0c, 0xc2, 0xab, 0xf4, 0x84, 0x9d, 0x97, 0x0b, 0x45, 0x65, 0x0f, 0x7a, 0x3c,
	0x57, 0xa2, 0xd3, 0x31, 0x5d, 0x52, 0x21, 0xfc, 0x04, 0x86, 0xeb, 0x30, 0x91, 0x2b, 0x31, 0xc7,
	0x0a, 0x28, 0x39, 0x96, 0xa2, 0xe2, 0x6e, 0xbb, 0xce, 0x7b, 0xfa, 0xaf, 0xf9, 0xe9, 0xaf, 0x00,
	0x00, 0x00, 0xff, 0xff, 0xc4, 0x9f, 0x0e, 0xf7, 0xaf, 0x07, 0x00, 0x00,
}
