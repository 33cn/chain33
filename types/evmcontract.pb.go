// Code generated by protoc-gen-go.
// source: evmcontract.proto
// DO NOT EDIT!

package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// 合约对象信息
type EVMContractObject struct {
	Addr  string            `protobuf:"bytes,1,opt,name=addr" json:"addr,omitempty"`
	Data  *EVMContractData  `protobuf:"bytes,2,opt,name=data" json:"data,omitempty"`
	State *EVMContractState `protobuf:"bytes,3,opt,name=state" json:"state,omitempty"`
}

func (m *EVMContractObject) Reset()                    { *m = EVMContractObject{} }
func (m *EVMContractObject) String() string            { return proto.CompactTextString(m) }
func (*EVMContractObject) ProtoMessage()               {}
func (*EVMContractObject) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{0} }

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
func (*EVMContractData) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{1} }

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
func (*EVMContractState) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{2} }

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
func (*EVMContractAction) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{3} }

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
func (*ReceiptEVMContract) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{4} }

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
func (*EVMStateChangeItem) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{5} }

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
func (*EVMContractDataCmd) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{6} }

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
func (*EVMContractStateCmd) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{7} }

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
func (*ReceiptEVMContractCmd) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{8} }

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
func (*CheckEVMAddrReq) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{9} }

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
func (*CheckEVMAddrResp) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{10} }

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
func (*EstimateEVMGasReq) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{11} }

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
func (*EstimateEVMGasResp) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{12} }

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
func (*EvmDebugReq) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{13} }

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
func (*EvmDebugResp) Descriptor() ([]byte, []int) { return fileDescriptor5, []int{14} }

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

func init() { proto.RegisterFile("evmcontract.proto", fileDescriptor5) }

var fileDescriptor5 = []byte{
	// 741 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xcc, 0x55, 0x41, 0x6f, 0x13, 0x3b,
	0x10, 0xd6, 0x26, 0x9b, 0x34, 0x3b, 0xc9, 0x7b, 0x6d, 0xfd, 0x1e, 0x25, 0xaa, 0x38, 0x44, 0x56,
	0x0b, 0x11, 0x12, 0x01, 0x35, 0x17, 0xd4, 0x03, 0x52, 0x49, 0xa3, 0x82, 0x44, 0x00, 0xb9, 0x52,
	0x0e, 0xdc, 0x1c, 0xaf, 0x49, 0x96, 0x66, 0xd7, 0xcb, 0xda, 0xa9, 0xd4, 0x2b, 0xbf, 0x81, 0x0b,
	0x12, 0x07, 0x10, 0xff, 0x8c, 0x13, 0x3f, 0x03, 0xd9, 0xde, 0x6c, 0x9c, 0x6d, 0x7a, 0xab, 0x10,
	0xa7, 0xcc, 0x37, 0x1e, 0xef, 0x7c, 0x33, 0xf3, 0x8d, 0x03, 0xbb, 0xfc, 0x32, 0x66, 0x22, 0x51,
	0x19, 0x65, 0xaa, 0x97, 0x66, 0x42, 0x09, 0x54, 0x53, 0x57, 0x29, 0x97, 0xf8, 0x93, 0x07, 0xbb,
	0xc3, 0xf1, 0x68, 0x90, 0x1f, 0xbe, 0x99, 0x7c, 0xe0, 0x4c, 0x21, 0x04, 0x3e, 0x0d, 0xc3, 0xac,
	0xed, 0x75, 0xbc, 0x6e, 0x40, 0x8c, 0x8d, 0x1e, 0x82, 0x1f, 0x52, 0x45, 0xdb, 0x95, 0x8e, 0xd7,
	0x6d, 0x1e, 0xed, 0xf5, 0xcc, 0xfd, 0x9e, 0x73, 0xf7, 0x94, 0x2a, 0x4a, 0x4c, 0x0c, 0x7a, 0x04,
	0x35, 0xa9, 0xa8, 0xe2, 0xed, 0xaa, 0x09, 0xbe, 0x7b, 0x3d, 0xf8, 0x5c, 0x1f, 0x13, 0x1b, 0x85,
	0xbf, 0x78, 0xb0, 0x5d, 0xfa, 0x10, 0x6a, 0xc3, 0x16, 0xcb, 0x38, 0x55, 0x62, 0xc9, 0x62, 0x09,
	0x35, 0xb9, 0x84, 0xc6, 0xdc, 0x10, 0x09, 0x88, 0xb1, 0xd1, 0xff, 0x50, 0xa3, 0xf3, 0x88, 0x4a,
	0x93, 0x30, 0x20, 0x16, 0x14, 0x65, 0xf8, 0x4e, 0x19, 0x08, 0x7c, 0x26, 0x42, 0xde, 0xae, 0x75,
	0xbc, 0x6e, 0x8b, 0x18, 0x1b, 0xed, 0x43, 0x43, 0xff, 0xbe, 0xa0, 0x72, 0xd6, 0xae, 0x1b, 0x7f,
	0x81, 0xf1, 0x4f, 0x0f, 0x76, 0xca, 0xbc, 0x75, 0xba, 0x44, 0x24, 0x8c, 0x1b, 0x6a, 0x3e, 0xb1,
	0x40, 0x7f, 0x46, 0x2e, 0x22, 0x16, 0x85, 0x3c, 0x34, 0xe4, 0x1a, 0xa4, 0xc0, 0xa8, 0x03, 0x4d,
	0xa9, 0x44, 0x46, 0xa7, 0x36, 0x4b, 0xd5, 0x64, 0x71, 0x5d, 0xe8, 0x19, 0x6c, 0xe5, 0xb0, 0xed,
	0x77, 0xaa, 0xdd, 0xe6, 0xd1, 0xc1, 0x0d, 0x5d, 0xeb, 0x9d, 0xdb, 0xb0, 0x61, 0xa2, 0xb2, 0x2b,
	0xb2, 0xbc, 0xb4, 0x7f, 0x0c, 0x2d, 0xf7, 0x00, 0xed, 0x40, 0xf5, 0x82, 0x5f, 0xe5, 0xcd, 0xd3,
	0xa6, 0x66, 0x7d, 0x49, 0xe7, 0x0b, 0xdb, 0xb9, 0x16, 0xb1, 0xe0, 0xb8, 0xf2, 0xd4, 0xc3, 0xdf,
	0xd7, 0x55, 0x70, 0xc2, 0x54, 0x24, 0x12, 0xb4, 0x07, 0x75, 0x1a, 0x8b, 0x45, 0xa2, 0xf2, 0x32,
	0x73, 0xa4, 0xeb, 0x9c, 0x52, 0xf9, 0x2a, 0x8a, 0x23, 0x65, 0x3e, 0xe5, 0x93, 0x02, 0xe7, 0x67,
	0x6f, 0xb3, 0x88, 0xd9, 0xe1, 0xff, 0x43, 0x0a, 0x5c, 0xb4, 0xde, 0x77, 0x5a, 0x5f, 0x0c, 0xae,
	0x56, 0x1a, 0x5c, 0x22, 0x14, 0x37, 0xc3, 0xd0, 0x23, 0x16, 0x8a, 0xe3, 0x6f, 0x1e, 0x20, 0xc2,
	0x19, 0x8f, 0x52, 0xe5, 0x50, 0xd5, 0x24, 0x19, 0x9d, 0xcf, 0xf9, 0x52, 0x26, 0x39, 0x42, 0x18,
	0x5a, 0x4b, 0xc5, 0xbf, 0x5e, 0xa9, 0x65, 0xcd, 0xe7, 0xc6, 0x9c, 0x68, 0x9d, 0x54, 0xd7, 0x63,
	0xb4, 0x4f, 0xeb, 0x70, 0x21, 0x79, 0x78, 0x46, 0xa5, 0xe1, 0xed, 0x93, 0x25, 0xd4, 0x0d, 0xce,
	0xb8, 0xca, 0x85, 0xa4, 0x4d, 0xfc, 0x1e, 0xd0, 0x70, 0x3c, 0x32, 0x43, 0x1a, 0xcc, 0x68, 0x32,
	0xe5, 0x2f, 0x15, 0x8f, 0x37, 0x0c, 0x62, 0x1f, 0x1a, 0x69, 0xc6, 0xc7, 0xce, 0x2c, 0x0a, 0x6c,
	0x38, 0x2d, 0xb2, 0x8c, 0x27, 0xca, 0x9e, 0x5b, 0xa5, 0xac, 0xf9, 0xf0, 0x57, 0xcf, 0x24, 0x72,
	0xf7, 0x65, 0x10, 0x87, 0x7f, 0x64, 0x65, 0x82, 0x1b, 0x56, 0x26, 0x70, 0x56, 0xe6, 0x97, 0x07,
	0xff, 0x95, 0x45, 0xab, 0xf9, 0xdd, 0xca, 0xd6, 0x04, 0xeb, 0x5b, 0x73, 0x52, 0xde, 0x9a, 0x07,
	0x37, 0x6c, 0xcd, 0x20, 0x0e, 0x6f, 0x67, 0x71, 0x02, 0x77, 0x71, 0x7e, 0x78, 0x70, 0xe7, 0xba,
	0x28, 0x75, 0xb1, 0x7f, 0x85, 0x2e, 0x03, 0xab, 0xcb, 0x43, 0xd8, 0x1e, 0xcc, 0x38, 0xbb, 0x18,
	0x8e, 0x47, 0xfa, 0x2e, 0xe1, 0x1f, 0x37, 0xbd, 0xf0, 0xf8, 0xb3, 0x07, 0x3b, 0xeb, 0x71, 0x32,
	0xb5, 0x83, 0xb6, 0x79, 0x4d, 0x70, 0x83, 0x14, 0xf8, 0x1a, 0xcf, 0xca, 0x06, 0x9e, 0xe5, 0x7a,
	0xab, 0x1b, 0xea, 0xbd, 0x07, 0x81, 0x51, 0x9f, 0x09, 0xb0, 0xca, 0x5b, 0x39, 0xf0, 0x14, 0x76,
	0x87, 0x52, 0x45, 0x31, 0x55, 0x7c, 0x38, 0x1e, 0x9d, 0x51, 0xa9, 0xf9, 0xff, 0x0b, 0x15, 0x25,
	0x72, 0xf6, 0x15, 0x25, 0x0a, 0x8d, 0x56, 0x9c, 0xb7, 0x65, 0x35, 0x82, 0xea, 0xda, 0x08, 0x56,
	0xef, 0x9a, 0xef, 0xbe, 0x6b, 0xf8, 0x3e, 0xa0, 0x72, 0x22, 0x99, 0xea, 0x76, 0x4e, 0xa9, 0xcc,
	0x35, 0xab, 0x4d, 0x7c, 0x08, 0xcd, 0xe1, 0x65, 0x7c, 0xca, 0x27, 0x8b, 0xa9, 0xa6, 0xb2, 0x07,
	0x75, 0x91, 0x6a, 0xd1, 0x99, 0x98, 0x1a, 0xc9, 0x11, 0x7e, 0x02, 0xad, 0x55, 0x98, 0x4c, 0xb5,
	0x98, 0x43, 0x0d, 0xb4, 0x1c, 0x17, 0x32, 0xe7, 0xee, 0xba, 0x9e, 0x1f, 0xbc, 0xc3, 0xd3, 0x48,
	0xcd, 0xe9, 0xa4, 0xd7, 0xef, 0xf7, 0x58, 0xf2, 0x98, 0xcd, 0x68, 0x94, 0xf4, 0xfb, 0xc5, 0xaf,
	0x51, 0xf6, 0xa4, 0x6e, 0xfe, 0xc0, 0xfb, 0xbf, 0x03, 0x00, 0x00, 0xff, 0xff, 0xb8, 0x62, 0x60,
	0x5a, 0xd5, 0x07, 0x00, 0x00,
}
