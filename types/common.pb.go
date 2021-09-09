// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.23.0
// 	protoc        v3.9.1
// source: common.proto

package types

import (
	reflect "reflect"
	sync "sync"

	proto "github.com/golang/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type Reply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	IsOk bool   `protobuf:"varint,1,opt,name=isOk,proto3" json:"isOk,omitempty"`
	Msg  []byte `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
}

func (x *Reply) Reset() {
	*x = Reply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Reply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Reply) ProtoMessage() {}

func (x *Reply) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Reply.ProtoReflect.Descriptor instead.
func (*Reply) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{0}
}

func (x *Reply) GetIsOk() bool {
	if x != nil {
		return x.IsOk
	}
	return false
}

func (x *Reply) GetMsg() []byte {
	if x != nil {
		return x.Msg
	}
	return nil
}

type ReqString struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data string `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *ReqString) Reset() {
	*x = ReqString{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReqString) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReqString) ProtoMessage() {}

func (x *ReqString) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReqString.ProtoReflect.Descriptor instead.
func (*ReqString) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{1}
}

func (x *ReqString) GetData() string {
	if x != nil {
		return x.Data
	}
	return ""
}

type ReplyString struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data string `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *ReplyString) Reset() {
	*x = ReplyString{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReplyString) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReplyString) ProtoMessage() {}

func (x *ReplyString) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReplyString.ProtoReflect.Descriptor instead.
func (*ReplyString) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{2}
}

func (x *ReplyString) GetData() string {
	if x != nil {
		return x.Data
	}
	return ""
}

type ReplyStrings struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Datas []string `protobuf:"bytes,1,rep,name=datas,proto3" json:"datas,omitempty"`
}

func (x *ReplyStrings) Reset() {
	*x = ReplyStrings{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReplyStrings) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReplyStrings) ProtoMessage() {}

func (x *ReplyStrings) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReplyStrings.ProtoReflect.Descriptor instead.
func (*ReplyStrings) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{3}
}

func (x *ReplyStrings) GetDatas() []string {
	if x != nil {
		return x.Datas
	}
	return nil
}

type ReqInt struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height int64 `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
}

func (x *ReqInt) Reset() {
	*x = ReqInt{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReqInt) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReqInt) ProtoMessage() {}

func (x *ReqInt) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReqInt.ProtoReflect.Descriptor instead.
func (*ReqInt) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{4}
}

func (x *ReqInt) GetHeight() int64 {
	if x != nil {
		return x.Height
	}
	return 0
}

type Int64 struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data int64 `protobuf:"varint,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *Int64) Reset() {
	*x = Int64{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Int64) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Int64) ProtoMessage() {}

func (x *Int64) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Int64.ProtoReflect.Descriptor instead.
func (*Int64) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{5}
}

func (x *Int64) GetData() int64 {
	if x != nil {
		return x.Data
	}
	return 0
}

type ReqHash struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Hash    []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	Upgrade bool   `protobuf:"varint,2,opt,name=upgrade,proto3" json:"upgrade,omitempty"`
}

func (x *ReqHash) Reset() {
	*x = ReqHash{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReqHash) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReqHash) ProtoMessage() {}

func (x *ReqHash) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReqHash.ProtoReflect.Descriptor instead.
func (*ReqHash) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{6}
}

func (x *ReqHash) GetHash() []byte {
	if x != nil {
		return x.Hash
	}
	return nil
}

func (x *ReqHash) GetUpgrade() bool {
	if x != nil {
		return x.Upgrade
	}
	return false
}

type ReplyHash struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
}

func (x *ReplyHash) Reset() {
	*x = ReplyHash{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReplyHash) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReplyHash) ProtoMessage() {}

func (x *ReplyHash) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReplyHash.ProtoReflect.Descriptor instead.
func (*ReplyHash) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{7}
}

func (x *ReplyHash) GetHash() []byte {
	if x != nil {
		return x.Hash
	}
	return nil
}

type ReqNil struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ReqNil) Reset() {
	*x = ReqNil{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReqNil) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReqNil) ProtoMessage() {}

func (x *ReqNil) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReqNil.ProtoReflect.Descriptor instead.
func (*ReqNil) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{8}
}

type ReqBytes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *ReqBytes) Reset() {
	*x = ReqBytes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReqBytes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReqBytes) ProtoMessage() {}

func (x *ReqBytes) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReqBytes.ProtoReflect.Descriptor instead.
func (*ReqBytes) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{9}
}

func (x *ReqBytes) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type ReqHashes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Hashes [][]byte `protobuf:"bytes,1,rep,name=hashes,proto3" json:"hashes,omitempty"`
}

func (x *ReqHashes) Reset() {
	*x = ReqHashes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReqHashes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReqHashes) ProtoMessage() {}

func (x *ReqHashes) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReqHashes.ProtoReflect.Descriptor instead.
func (*ReqHashes) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{10}
}

func (x *ReqHashes) GetHashes() [][]byte {
	if x != nil {
		return x.Hashes
	}
	return nil
}

type ReplyHashes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Hashes [][]byte `protobuf:"bytes,1,rep,name=hashes,proto3" json:"hashes,omitempty"`
}

func (x *ReplyHashes) Reset() {
	*x = ReplyHashes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[11]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReplyHashes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReplyHashes) ProtoMessage() {}

func (x *ReplyHashes) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[11]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReplyHashes.ProtoReflect.Descriptor instead.
func (*ReplyHashes) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{11}
}

func (x *ReplyHashes) GetHashes() [][]byte {
	if x != nil {
		return x.Hashes
	}
	return nil
}

type KeyValue struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key   []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *KeyValue) Reset() {
	*x = KeyValue{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[12]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *KeyValue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*KeyValue) ProtoMessage() {}

func (x *KeyValue) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[12]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use KeyValue.ProtoReflect.Descriptor instead.
func (*KeyValue) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{12}
}

func (x *KeyValue) GetKey() []byte {
	if x != nil {
		return x.Key
	}
	return nil
}

func (x *KeyValue) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

type TxHash struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Hash string `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
}

func (x *TxHash) Reset() {
	*x = TxHash{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[13]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TxHash) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TxHash) ProtoMessage() {}

func (x *TxHash) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[13]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TxHash.ProtoReflect.Descriptor instead.
func (*TxHash) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{13}
}

func (x *TxHash) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

type TimeStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	NtpTime   string `protobuf:"bytes,1,opt,name=ntpTime,proto3" json:"ntpTime,omitempty"`
	LocalTime string `protobuf:"bytes,2,opt,name=localTime,proto3" json:"localTime,omitempty"`
	Diff      int64  `protobuf:"varint,3,opt,name=diff,proto3" json:"diff,omitempty"`
}

func (x *TimeStatus) Reset() {
	*x = TimeStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[14]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TimeStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TimeStatus) ProtoMessage() {}

func (x *TimeStatus) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[14]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TimeStatus.ProtoReflect.Descriptor instead.
func (*TimeStatus) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{14}
}

func (x *TimeStatus) GetNtpTime() string {
	if x != nil {
		return x.NtpTime
	}
	return ""
}

func (x *TimeStatus) GetLocalTime() string {
	if x != nil {
		return x.LocalTime
	}
	return ""
}

func (x *TimeStatus) GetDiff() int64 {
	if x != nil {
		return x.Diff
	}
	return 0
}

type ReqKey struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (x *ReqKey) Reset() {
	*x = ReqKey{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[15]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReqKey) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReqKey) ProtoMessage() {}

func (x *ReqKey) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[15]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReqKey.ProtoReflect.Descriptor instead.
func (*ReqKey) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{15}
}

func (x *ReqKey) GetKey() []byte {
	if x != nil {
		return x.Key
	}
	return nil
}

type ReqRandHash struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ExecName string `protobuf:"bytes,1,opt,name=execName,proto3" json:"execName,omitempty"`
	Height   int64  `protobuf:"varint,2,opt,name=height,proto3" json:"height,omitempty"`
	BlockNum int64  `protobuf:"varint,3,opt,name=blockNum,proto3" json:"blockNum,omitempty"`
	Hash     []byte `protobuf:"bytes,4,opt,name=hash,proto3" json:"hash,omitempty"`
}

func (x *ReqRandHash) Reset() {
	*x = ReqRandHash{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[16]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReqRandHash) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReqRandHash) ProtoMessage() {}

func (x *ReqRandHash) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[16]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReqRandHash.ProtoReflect.Descriptor instead.
func (*ReqRandHash) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{16}
}

func (x *ReqRandHash) GetExecName() string {
	if x != nil {
		return x.ExecName
	}
	return ""
}

func (x *ReqRandHash) GetHeight() int64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *ReqRandHash) GetBlockNum() int64 {
	if x != nil {
		return x.BlockNum
	}
	return 0
}

func (x *ReqRandHash) GetHash() []byte {
	if x != nil {
		return x.Hash
	}
	return nil
}

//*
//当前软件版本信息
type VersionInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Title   string `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	App     string `protobuf:"bytes,2,opt,name=app,proto3" json:"app,omitempty"`
	Chain33 string `protobuf:"bytes,3,opt,name=chain33,proto3" json:"chain33,omitempty"`
	LocalDb string `protobuf:"bytes,4,opt,name=localDb,proto3" json:"localDb,omitempty"`
	ChainID int32  `protobuf:"varint,5,opt,name=chainID,proto3" json:"chainID,omitempty"`
}

func (x *VersionInfo) Reset() {
	*x = VersionInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_common_proto_msgTypes[17]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VersionInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VersionInfo) ProtoMessage() {}

func (x *VersionInfo) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[17]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VersionInfo.ProtoReflect.Descriptor instead.
func (*VersionInfo) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{17}
}

func (x *VersionInfo) GetTitle() string {
	if x != nil {
		return x.Title
	}
	return ""
}

func (x *VersionInfo) GetApp() string {
	if x != nil {
		return x.App
	}
	return ""
}

func (x *VersionInfo) GetChain33() string {
	if x != nil {
		return x.Chain33
	}
	return ""
}

func (x *VersionInfo) GetLocalDb() string {
	if x != nil {
		return x.LocalDb
	}
	return ""
}

func (x *VersionInfo) GetChainID() int32 {
	if x != nil {
		return x.ChainID
	}
	return 0
}

var File_common_proto protoreflect.FileDescriptor

var file_common_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05,
	0x74, 0x79, 0x70, 0x65, 0x73, 0x22, 0x2d, 0x0a, 0x05, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x12,
	0x0a, 0x04, 0x69, 0x73, 0x4f, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x04, 0x69, 0x73,
	0x4f, 0x6b, 0x12, 0x10, 0x0a, 0x03, 0x6d, 0x73, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x03, 0x6d, 0x73, 0x67, 0x22, 0x1f, 0x0a, 0x09, 0x52, 0x65, 0x71, 0x53, 0x74, 0x72, 0x69, 0x6e,
	0x67, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x21, 0x0a, 0x0b, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x53, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x24, 0x0a, 0x0c, 0x52, 0x65, 0x70, 0x6c,
	0x79, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x64, 0x61, 0x74, 0x61,
	0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x64, 0x61, 0x74, 0x61, 0x73, 0x22, 0x20,
	0x0a, 0x06, 0x52, 0x65, 0x71, 0x49, 0x6e, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67,
	0x68, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74,
	0x22, 0x1b, 0x0a, 0x05, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74,
	0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x37, 0x0a,
	0x07, 0x52, 0x65, 0x71, 0x48, 0x61, 0x73, 0x68, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x12, 0x18, 0x0a, 0x07,
	0x75, 0x70, 0x67, 0x72, 0x61, 0x64, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x75,
	0x70, 0x67, 0x72, 0x61, 0x64, 0x65, 0x22, 0x1f, 0x0a, 0x09, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x48,
	0x61, 0x73, 0x68, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x22, 0x08, 0x0a, 0x06, 0x52, 0x65, 0x71, 0x4e, 0x69,
	0x6c, 0x22, 0x1e, 0x0a, 0x08, 0x52, 0x65, 0x71, 0x42, 0x79, 0x74, 0x65, 0x73, 0x12, 0x12, 0x0a,
	0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74,
	0x61, 0x22, 0x23, 0x0a, 0x09, 0x52, 0x65, 0x71, 0x48, 0x61, 0x73, 0x68, 0x65, 0x73, 0x12, 0x16,
	0x0a, 0x06, 0x68, 0x61, 0x73, 0x68, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x06,
	0x68, 0x61, 0x73, 0x68, 0x65, 0x73, 0x22, 0x25, 0x0a, 0x0b, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x48,
	0x61, 0x73, 0x68, 0x65, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x61, 0x73, 0x68, 0x65, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x06, 0x68, 0x61, 0x73, 0x68, 0x65, 0x73, 0x22, 0x32, 0x0a,
	0x08, 0x4b, 0x65, 0x79, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x22, 0x1c, 0x0a, 0x06, 0x54, 0x78, 0x48, 0x61, 0x73, 0x68, 0x12, 0x12, 0x0a, 0x04, 0x68,
	0x61, 0x73, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x22,
	0x58, 0x0a, 0x0a, 0x54, 0x69, 0x6d, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x18, 0x0a,
	0x07, 0x6e, 0x74, 0x70, 0x54, 0x69, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x6e, 0x74, 0x70, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x6c, 0x6f, 0x63, 0x61, 0x6c,
	0x54, 0x69, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6c, 0x6f, 0x63, 0x61,
	0x6c, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x69, 0x66, 0x66, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x04, 0x64, 0x69, 0x66, 0x66, 0x22, 0x1a, 0x0a, 0x06, 0x52, 0x65, 0x71,
	0x4b, 0x65, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x03, 0x6b, 0x65, 0x79, 0x22, 0x71, 0x0a, 0x0b, 0x52, 0x65, 0x71, 0x52, 0x61, 0x6e, 0x64,
	0x48, 0x61, 0x73, 0x68, 0x12, 0x1a, 0x0a, 0x08, 0x65, 0x78, 0x65, 0x63, 0x4e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x65, 0x78, 0x65, 0x63, 0x4e, 0x61, 0x6d, 0x65,
	0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x62, 0x6c, 0x6f, 0x63,
	0x6b, 0x4e, 0x75, 0x6d, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x62, 0x6c, 0x6f, 0x63,
	0x6b, 0x4e, 0x75, 0x6d, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x22, 0x83, 0x01, 0x0a, 0x0b, 0x56, 0x65, 0x72,
	0x73, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x69, 0x74, 0x6c,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x69, 0x74, 0x6c, 0x65, 0x12, 0x10,
	0x0a, 0x03, 0x61, 0x70, 0x70, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x61, 0x70, 0x70,
	0x12, 0x18, 0x0a, 0x07, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x33, 0x33, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x33, 0x33, 0x12, 0x18, 0x0a, 0x07, 0x6c, 0x6f,
	0x63, 0x61, 0x6c, 0x44, 0x62, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6c, 0x6f, 0x63,
	0x61, 0x6c, 0x44, 0x62, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x49, 0x44, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x49, 0x44, 0x42, 0x1f,
	0x5a, 0x1d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x33, 0x33, 0x63,
	0x6e, 0x2f, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x33, 0x33, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_common_proto_rawDescOnce sync.Once
	file_common_proto_rawDescData = file_common_proto_rawDesc
)

func file_common_proto_rawDescGZIP() []byte {
	file_common_proto_rawDescOnce.Do(func() {
		file_common_proto_rawDescData = protoimpl.X.CompressGZIP(file_common_proto_rawDescData)
	})
	return file_common_proto_rawDescData
}

var file_common_proto_msgTypes = make([]protoimpl.MessageInfo, 18)
var file_common_proto_goTypes = []interface{}{
	(*Reply)(nil),        // 0: types.Reply
	(*ReqString)(nil),    // 1: types.ReqString
	(*ReplyString)(nil),  // 2: types.ReplyString
	(*ReplyStrings)(nil), // 3: types.ReplyStrings
	(*ReqInt)(nil),       // 4: types.ReqInt
	(*Int64)(nil),        // 5: types.Int64
	(*ReqHash)(nil),      // 6: types.ReqHash
	(*ReplyHash)(nil),    // 7: types.ReplyHash
	(*ReqNil)(nil),       // 8: types.ReqNil
	(*ReqBytes)(nil),     // 9: types.ReqBytes
	(*ReqHashes)(nil),    // 10: types.ReqHashes
	(*ReplyHashes)(nil),  // 11: types.ReplyHashes
	(*KeyValue)(nil),     // 12: types.KeyValue
	(*TxHash)(nil),       // 13: types.TxHash
	(*TimeStatus)(nil),   // 14: types.TimeStatus
	(*ReqKey)(nil),       // 15: types.ReqKey
	(*ReqRandHash)(nil),  // 16: types.ReqRandHash
	(*VersionInfo)(nil),  // 17: types.VersionInfo
}
var file_common_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_common_proto_init() }
func file_common_proto_init() {
	if File_common_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_common_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Reply); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReqString); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReplyString); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReplyStrings); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReqInt); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Int64); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReqHash); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReplyHash); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReqNil); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReqBytes); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReqHashes); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[11].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReplyHashes); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[12].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*KeyValue); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[13].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TxHash); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[14].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TimeStatus); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[15].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReqKey); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[16].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReqRandHash); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_common_proto_msgTypes[17].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VersionInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_common_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   18,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_common_proto_goTypes,
		DependencyIndexes: file_common_proto_depIdxs,
		MessageInfos:      file_common_proto_msgTypes,
	}.Build()
	File_common_proto = out.File
	file_common_proto_rawDesc = nil
	file_common_proto_goTypes = nil
	file_common_proto_depIdxs = nil
}
