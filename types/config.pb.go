// Code generated by protoc-gen-go. DO NOT EDIT.
// source: config.proto

package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Config struct {
	Title           string      `protobuf:"bytes,1,opt,name=title" json:"title,omitempty"`
	Loglevel        string      `protobuf:"bytes,2,opt,name=loglevel" json:"loglevel,omitempty"`
	LogConsoleLevel string      `protobuf:"bytes,10,opt,name=logConsoleLevel" json:"logConsoleLevel,omitempty"`
	LogFile         string      `protobuf:"bytes,9,opt,name=logFile" json:"logFile,omitempty"`
	Store           *Store      `protobuf:"bytes,3,opt,name=store" json:"store,omitempty"`
	LocalStore      *LocalStore `protobuf:"bytes,11,opt,name=localStore" json:"localStore,omitempty"`
	Consensus       *Consensus  `protobuf:"bytes,4,opt,name=consensus" json:"consensus,omitempty"`
	MemPool         *MemPool    `protobuf:"bytes,5,opt,name=memPool" json:"memPool,omitempty"`
	BlockChain      *BlockChain `protobuf:"bytes,6,opt,name=blockChain" json:"blockChain,omitempty"`
	Wallet          *Wallet     `protobuf:"bytes,7,opt,name=wallet" json:"wallet,omitempty"`
	P2P             *P2P        `protobuf:"bytes,8,opt,name=p2p" json:"p2p,omitempty"`
	Rpc             *Rpc        `protobuf:"bytes,12,opt,name=rpc" json:"rpc,omitempty"`
}

func (m *Config) Reset()                    { *m = Config{} }
func (m *Config) String() string            { return proto.CompactTextString(m) }
func (*Config) ProtoMessage()               {}
func (*Config) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{0} }

func (m *Config) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *Config) GetLoglevel() string {
	if m != nil {
		return m.Loglevel
	}
	return ""
}

func (m *Config) GetLogConsoleLevel() string {
	if m != nil {
		return m.LogConsoleLevel
	}
	return ""
}

func (m *Config) GetLogFile() string {
	if m != nil {
		return m.LogFile
	}
	return ""
}

func (m *Config) GetStore() *Store {
	if m != nil {
		return m.Store
	}
	return nil
}

func (m *Config) GetLocalStore() *LocalStore {
	if m != nil {
		return m.LocalStore
	}
	return nil
}

func (m *Config) GetConsensus() *Consensus {
	if m != nil {
		return m.Consensus
	}
	return nil
}

func (m *Config) GetMemPool() *MemPool {
	if m != nil {
		return m.MemPool
	}
	return nil
}

func (m *Config) GetBlockChain() *BlockChain {
	if m != nil {
		return m.BlockChain
	}
	return nil
}

func (m *Config) GetWallet() *Wallet {
	if m != nil {
		return m.Wallet
	}
	return nil
}

func (m *Config) GetP2P() *P2P {
	if m != nil {
		return m.P2P
	}
	return nil
}

func (m *Config) GetRpc() *Rpc {
	if m != nil {
		return m.Rpc
	}
	return nil
}

type MemPool struct {
	PoolCacheSize int64 `protobuf:"varint,1,opt,name=poolCacheSize" json:"poolCacheSize,omitempty"`
	MinTxFee      int64 `protobuf:"varint,2,opt,name=minTxFee" json:"minTxFee,omitempty"`
}

func (m *MemPool) Reset()                    { *m = MemPool{} }
func (m *MemPool) String() string            { return proto.CompactTextString(m) }
func (*MemPool) ProtoMessage()               {}
func (*MemPool) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{1} }

func (m *MemPool) GetPoolCacheSize() int64 {
	if m != nil {
		return m.PoolCacheSize
	}
	return 0
}

func (m *MemPool) GetMinTxFee() int64 {
	if m != nil {
		return m.MinTxFee
	}
	return 0
}

type Consensus struct {
	Name             string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Genesis          string `protobuf:"bytes,2,opt,name=genesis" json:"genesis,omitempty"`
	Minerstart       bool   `protobuf:"varint,3,opt,name=minerstart" json:"minerstart,omitempty"`
	GenesisBlockTime int64  `protobuf:"varint,4,opt,name=genesisBlockTime" json:"genesisBlockTime,omitempty"`
	HotkeyAddr       string `protobuf:"bytes,5,opt,name=hotkeyAddr" json:"hotkeyAddr,omitempty"`
}

func (m *Consensus) Reset()                    { *m = Consensus{} }
func (m *Consensus) String() string            { return proto.CompactTextString(m) }
func (*Consensus) ProtoMessage()               {}
func (*Consensus) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{2} }

func (m *Consensus) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Consensus) GetGenesis() string {
	if m != nil {
		return m.Genesis
	}
	return ""
}

func (m *Consensus) GetMinerstart() bool {
	if m != nil {
		return m.Minerstart
	}
	return false
}

func (m *Consensus) GetGenesisBlockTime() int64 {
	if m != nil {
		return m.GenesisBlockTime
	}
	return 0
}

func (m *Consensus) GetHotkeyAddr() string {
	if m != nil {
		return m.HotkeyAddr
	}
	return ""
}

type Wallet struct {
	MinFee   int64  `protobuf:"varint,1,opt,name=minFee" json:"minFee,omitempty"`
	Driver   string `protobuf:"bytes,2,opt,name=driver" json:"driver,omitempty"`
	DbPath   string `protobuf:"bytes,3,opt,name=dbPath" json:"dbPath,omitempty"`
	SignType string `protobuf:"bytes,4,opt,name=signType" json:"signType,omitempty"`
}

func (m *Wallet) Reset()                    { *m = Wallet{} }
func (m *Wallet) String() string            { return proto.CompactTextString(m) }
func (*Wallet) ProtoMessage()               {}
func (*Wallet) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{3} }

func (m *Wallet) GetMinFee() int64 {
	if m != nil {
		return m.MinFee
	}
	return 0
}

func (m *Wallet) GetDriver() string {
	if m != nil {
		return m.Driver
	}
	return ""
}

func (m *Wallet) GetDbPath() string {
	if m != nil {
		return m.DbPath
	}
	return ""
}

func (m *Wallet) GetSignType() string {
	if m != nil {
		return m.SignType
	}
	return ""
}

type Store struct {
	Name   string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Driver string `protobuf:"bytes,2,opt,name=driver" json:"driver,omitempty"`
	DbPath string `protobuf:"bytes,3,opt,name=dbPath" json:"dbPath,omitempty"`
}

func (m *Store) Reset()                    { *m = Store{} }
func (m *Store) String() string            { return proto.CompactTextString(m) }
func (*Store) ProtoMessage()               {}
func (*Store) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{4} }

func (m *Store) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Store) GetDriver() string {
	if m != nil {
		return m.Driver
	}
	return ""
}

func (m *Store) GetDbPath() string {
	if m != nil {
		return m.DbPath
	}
	return ""
}

type LocalStore struct {
	Driver string `protobuf:"bytes,1,opt,name=driver" json:"driver,omitempty"`
	DbPath string `protobuf:"bytes,2,opt,name=dbPath" json:"dbPath,omitempty"`
}

func (m *LocalStore) Reset()                    { *m = LocalStore{} }
func (m *LocalStore) String() string            { return proto.CompactTextString(m) }
func (*LocalStore) ProtoMessage()               {}
func (*LocalStore) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{5} }

func (m *LocalStore) GetDriver() string {
	if m != nil {
		return m.Driver
	}
	return ""
}

func (m *LocalStore) GetDbPath() string {
	if m != nil {
		return m.DbPath
	}
	return ""
}

type BlockChain struct {
	DefCacheSize        int64  `protobuf:"varint,1,opt,name=defCacheSize" json:"defCacheSize,omitempty"`
	MaxFetchBlockNum    int64  `protobuf:"varint,2,opt,name=maxFetchBlockNum" json:"maxFetchBlockNum,omitempty"`
	TimeoutSeconds      int64  `protobuf:"varint,3,opt,name=timeoutSeconds" json:"timeoutSeconds,omitempty"`
	BatchBlockNum       int64  `protobuf:"varint,4,opt,name=batchBlockNum" json:"batchBlockNum,omitempty"`
	Driver              string `protobuf:"bytes,5,opt,name=driver" json:"driver,omitempty"`
	DbPath              string `protobuf:"bytes,6,opt,name=dbPath" json:"dbPath,omitempty"`
	IsStrongConsistency bool   `protobuf:"varint,7,opt,name=isStrongConsistency" json:"isStrongConsistency,omitempty"`
	SingleMode          bool   `protobuf:"varint,8,opt,name=singleMode" json:"singleMode,omitempty"`
}

func (m *BlockChain) Reset()                    { *m = BlockChain{} }
func (m *BlockChain) String() string            { return proto.CompactTextString(m) }
func (*BlockChain) ProtoMessage()               {}
func (*BlockChain) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{6} }

func (m *BlockChain) GetDefCacheSize() int64 {
	if m != nil {
		return m.DefCacheSize
	}
	return 0
}

func (m *BlockChain) GetMaxFetchBlockNum() int64 {
	if m != nil {
		return m.MaxFetchBlockNum
	}
	return 0
}

func (m *BlockChain) GetTimeoutSeconds() int64 {
	if m != nil {
		return m.TimeoutSeconds
	}
	return 0
}

func (m *BlockChain) GetBatchBlockNum() int64 {
	if m != nil {
		return m.BatchBlockNum
	}
	return 0
}

func (m *BlockChain) GetDriver() string {
	if m != nil {
		return m.Driver
	}
	return ""
}

func (m *BlockChain) GetDbPath() string {
	if m != nil {
		return m.DbPath
	}
	return ""
}

func (m *BlockChain) GetIsStrongConsistency() bool {
	if m != nil {
		return m.IsStrongConsistency
	}
	return false
}

func (m *BlockChain) GetSingleMode() bool {
	if m != nil {
		return m.SingleMode
	}
	return false
}

type P2P struct {
	SeedPort     int32    `protobuf:"varint,1,opt,name=seedPort" json:"seedPort,omitempty"`
	DbPath       string   `protobuf:"bytes,2,opt,name=dbPath" json:"dbPath,omitempty"`
	GrpcLogFile  string   `protobuf:"bytes,3,opt,name=grpcLogFile" json:"grpcLogFile,omitempty"`
	IsSeed       bool     `protobuf:"varint,4,opt,name=isSeed" json:"isSeed,omitempty"`
	Seeds        []string `protobuf:"bytes,5,rep,name=seeds" json:"seeds,omitempty"`
	Enable       bool     `protobuf:"varint,6,opt,name=enable" json:"enable,omitempty"`
	MsgCacheSize int32    `protobuf:"varint,7,opt,name=msgCacheSize" json:"msgCacheSize,omitempty"`
	Version      int32    `protobuf:"varint,8,opt,name=version" json:"version,omitempty"`
	VerMix       int32    `protobuf:"varint,9,opt,name=verMix" json:"verMix,omitempty"`
	VerMax       int32    `protobuf:"varint,10,opt,name=verMax" json:"verMax,omitempty"`
}

func (m *P2P) Reset()                    { *m = P2P{} }
func (m *P2P) String() string            { return proto.CompactTextString(m) }
func (*P2P) ProtoMessage()               {}
func (*P2P) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{7} }

func (m *P2P) GetSeedPort() int32 {
	if m != nil {
		return m.SeedPort
	}
	return 0
}

func (m *P2P) GetDbPath() string {
	if m != nil {
		return m.DbPath
	}
	return ""
}

func (m *P2P) GetGrpcLogFile() string {
	if m != nil {
		return m.GrpcLogFile
	}
	return ""
}

func (m *P2P) GetIsSeed() bool {
	if m != nil {
		return m.IsSeed
	}
	return false
}

func (m *P2P) GetSeeds() []string {
	if m != nil {
		return m.Seeds
	}
	return nil
}

func (m *P2P) GetEnable() bool {
	if m != nil {
		return m.Enable
	}
	return false
}

func (m *P2P) GetMsgCacheSize() int32 {
	if m != nil {
		return m.MsgCacheSize
	}
	return 0
}

func (m *P2P) GetVersion() int32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *P2P) GetVerMix() int32 {
	if m != nil {
		return m.VerMix
	}
	return 0
}

func (m *P2P) GetVerMax() int32 {
	if m != nil {
		return m.VerMax
	}
	return 0
}

type Rpc struct {
	JrpcBindAddr string   `protobuf:"bytes,1,opt,name=jrpcBindAddr" json:"jrpcBindAddr,omitempty"`
	GrpcBindAddr string   `protobuf:"bytes,2,opt,name=grpcBindAddr" json:"grpcBindAddr,omitempty"`
	Whitlist     []string `protobuf:"bytes,3,rep,name=whitlist" json:"whitlist,omitempty"`
}

func (m *Rpc) Reset()                    { *m = Rpc{} }
func (m *Rpc) String() string            { return proto.CompactTextString(m) }
func (*Rpc) ProtoMessage()               {}
func (*Rpc) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{8} }

func (m *Rpc) GetJrpcBindAddr() string {
	if m != nil {
		return m.JrpcBindAddr
	}
	return ""
}

func (m *Rpc) GetGrpcBindAddr() string {
	if m != nil {
		return m.GrpcBindAddr
	}
	return ""
}

func (m *Rpc) GetWhitlist() []string {
	if m != nil {
		return m.Whitlist
	}
	return nil
}

func init() {
	proto.RegisterType((*Config)(nil), "types.Config")
	proto.RegisterType((*MemPool)(nil), "types.MemPool")
	proto.RegisterType((*Consensus)(nil), "types.Consensus")
	proto.RegisterType((*Wallet)(nil), "types.Wallet")
	proto.RegisterType((*Store)(nil), "types.Store")
	proto.RegisterType((*LocalStore)(nil), "types.LocalStore")
	proto.RegisterType((*BlockChain)(nil), "types.BlockChain")
	proto.RegisterType((*P2P)(nil), "types.P2P")
	proto.RegisterType((*Rpc)(nil), "types.Rpc")
}

func init() { proto.RegisterFile("config.proto", fileDescriptor3) }

var fileDescriptor3 = []byte{
	// 734 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x55, 0xcf, 0x6e, 0xfb, 0x44,
	0x10, 0x56, 0xe2, 0x3a, 0x7f, 0xa6, 0x69, 0x29, 0x0b, 0x42, 0x16, 0x42, 0x28, 0xb2, 0x00, 0x45,
	0x1c, 0x22, 0x08, 0x57, 0x2e, 0x34, 0x52, 0x2f, 0x6d, 0x51, 0xb4, 0xa9, 0xc4, 0xd9, 0xb1, 0xa7,
	0xce, 0xd2, 0xf5, 0xae, 0xe5, 0xdd, 0xa6, 0x09, 0x2f, 0xc1, 0x53, 0x70, 0xe3, 0x35, 0x78, 0x2f,
	0xb4, 0xe3, 0x8d, 0x63, 0xf7, 0xcf, 0xe1, 0x77, 0xcb, 0xf7, 0xcd, 0x37, 0xeb, 0x99, 0xf9, 0x66,
	0x37, 0x30, 0x49, 0xb5, 0x7a, 0x14, 0xf9, 0xbc, 0xac, 0xb4, 0xd5, 0x2c, 0xb4, 0x87, 0x12, 0x4d,
	0xfc, 0x5f, 0x00, 0x83, 0x25, 0xf1, 0xec, 0x4b, 0x08, 0xad, 0xb0, 0x12, 0xa3, 0xde, 0xb4, 0x37,
	0x1b, 0xf3, 0x1a, 0xb0, 0xaf, 0x61, 0x24, 0x75, 0x2e, 0x71, 0x87, 0x32, 0xea, 0x53, 0xa0, 0xc1,
	0x6c, 0x06, 0x9f, 0x49, 0x9d, 0x2f, 0xb5, 0x32, 0x5a, 0xe2, 0x1d, 0x49, 0x80, 0x24, 0xaf, 0x69,
	0x16, 0xc1, 0x50, 0xea, 0xfc, 0x46, 0x48, 0x8c, 0xc6, 0xa4, 0x38, 0x42, 0x16, 0x43, 0x68, 0xac,
	0xae, 0x30, 0x0a, 0xa6, 0xbd, 0xd9, 0xf9, 0x62, 0x32, 0xa7, 0xba, 0xe6, 0x6b, 0xc7, 0xf1, 0x3a,
	0xc4, 0x7e, 0x06, 0x90, 0x3a, 0x4d, 0x24, 0x91, 0xd1, 0x39, 0x09, 0x3f, 0xf7, 0xc2, 0xbb, 0x26,
	0xc0, 0x5b, 0x22, 0x36, 0x87, 0x71, 0xaa, 0x95, 0x41, 0x65, 0x9e, 0x4d, 0x74, 0x46, 0x19, 0x57,
	0x3e, 0x63, 0x79, 0xe4, 0xf9, 0x49, 0xc2, 0x66, 0x30, 0x2c, 0xb0, 0x58, 0x69, 0x2d, 0xa3, 0x90,
	0xd4, 0x97, 0x5e, 0x7d, 0x5f, 0xb3, 0xfc, 0x18, 0x76, 0xc5, 0x6c, 0xa4, 0x4e, 0x9f, 0x96, 0xdb,
	0x44, 0xa8, 0x68, 0xd0, 0x29, 0xe6, 0xba, 0x09, 0xf0, 0x96, 0x88, 0x7d, 0x0f, 0x83, 0x97, 0x44,
	0x4a, 0xb4, 0xd1, 0x90, 0xe4, 0x17, 0x5e, 0xfe, 0x07, 0x91, 0xdc, 0x07, 0xd9, 0x37, 0x10, 0x94,
	0x8b, 0x32, 0x1a, 0x91, 0x06, 0xbc, 0x66, 0xb5, 0x58, 0x71, 0x47, 0xbb, 0x68, 0x55, 0xa6, 0xd1,
	0xa4, 0x13, 0xe5, 0x65, 0xca, 0x1d, 0x1d, 0xdf, 0xc2, 0xd0, 0x57, 0xca, 0xbe, 0x83, 0x8b, 0x52,
	0x6b, 0xb9, 0x4c, 0xd2, 0x2d, 0xae, 0xc5, 0x5f, 0xb5, 0x9f, 0x01, 0xef, 0x92, 0xce, 0xd7, 0x42,
	0xa8, 0x87, 0xfd, 0x0d, 0x22, 0xf9, 0x1a, 0xf0, 0x06, 0xc7, 0xff, 0xf4, 0x60, 0xdc, 0x4c, 0x89,
	0x31, 0x38, 0x53, 0x49, 0x71, 0x5c, 0x0b, 0xfa, 0xed, 0xfc, 0xcc, 0x51, 0xa1, 0x11, 0xc6, 0x2f,
	0xc5, 0x11, 0xb2, 0x6f, 0x01, 0x0a, 0xa1, 0xb0, 0x32, 0x36, 0xa9, 0x2c, 0x99, 0x3a, 0xe2, 0x2d,
	0x86, 0xfd, 0x08, 0x57, 0x5e, 0x4a, 0xc3, 0x7a, 0x10, 0x05, 0x92, 0x3f, 0x01, 0x7f, 0xc3, 0xbb,
	0xb3, 0xb6, 0xda, 0x3e, 0xe1, 0xe1, 0xb7, 0x2c, 0xab, 0xc8, 0x97, 0x31, 0x6f, 0x31, 0xb1, 0x84,
	0x41, 0x3d, 0x42, 0xf6, 0x15, 0x0c, 0x0a, 0xa1, 0x5c, 0x2f, 0x75, 0xb3, 0x1e, 0x39, 0x3e, 0xab,
	0xc4, 0x0e, 0x2b, 0x5f, 0xa6, 0x47, 0xc4, 0x6f, 0x56, 0x89, 0xdd, 0x52, 0x85, 0x8e, 0x27, 0xe4,
	0xa6, 0x62, 0x44, 0xae, 0x1e, 0x0e, 0x65, 0x5d, 0xd5, 0x98, 0x37, 0x38, 0xbe, 0x85, 0xb0, 0xde,
	0xad, 0xf7, 0x06, 0xf2, 0x89, 0x1f, 0x8a, 0x7f, 0x05, 0x38, 0x6d, 0x6e, 0x2b, 0xbb, 0xf7, 0x41,
	0x76, 0xbf, 0x93, 0xfd, 0x6f, 0x1f, 0xe0, 0xb4, 0x6b, 0x2c, 0x86, 0x49, 0x86, 0x8f, 0xaf, 0x0d,
	0xef, 0x70, 0x6e, 0xee, 0x45, 0xb2, 0xbf, 0x41, 0x9b, 0x6e, 0x29, 0xf3, 0xf7, 0xe7, 0xc2, 0xfb,
	0xfe, 0x86, 0x67, 0x3f, 0xc0, 0xa5, 0x15, 0x05, 0xea, 0x67, 0xbb, 0xc6, 0x54, 0xab, 0xcc, 0x50,
	0xf1, 0x01, 0x7f, 0xc5, 0xba, 0x4d, 0xdb, 0x24, 0xed, 0x03, 0x6b, 0x23, 0xbb, 0x64, 0xab, 0xb9,
	0xf0, 0x83, 0xe6, 0x06, 0x1d, 0x0f, 0x7e, 0x82, 0x2f, 0x84, 0x59, 0xdb, 0x4a, 0x2b, 0x7a, 0x43,
	0x84, 0xb1, 0xa8, 0xd2, 0x03, 0x5d, 0x9d, 0x11, 0x7f, 0x2f, 0xe4, 0xf6, 0xc4, 0x08, 0x95, 0x4b,
	0xbc, 0xd7, 0x19, 0xd2, 0xfd, 0x19, 0xf1, 0x16, 0x13, 0xff, 0xdd, 0x87, 0x60, 0xb5, 0x58, 0x91,
	0xbb, 0x88, 0xd9, 0x4a, 0x57, 0x96, 0x66, 0x14, 0xf2, 0x06, 0x7f, 0x34, 0x6a, 0x36, 0x85, 0xf3,
	0xbc, 0x2a, 0xd3, 0x3b, 0xff, 0x7a, 0xd5, 0x2e, 0xb6, 0x29, 0x97, 0x29, 0xcc, 0x1a, 0x31, 0xa3,
	0xf6, 0x47, 0xdc, 0x23, 0xf7, 0x9e, 0xba, 0xd3, 0x4d, 0x14, 0x4e, 0x03, 0xf7, 0x9e, 0x12, 0x70,
	0x6a, 0x54, 0xc9, 0x46, 0x22, 0x75, 0x3d, 0xe2, 0x1e, 0x39, 0x0f, 0x0b, 0x93, 0x9f, 0x3c, 0x1c,
	0x52, 0x7d, 0x1d, 0xce, 0xdd, 0xba, 0x1d, 0x56, 0x46, 0x68, 0x45, 0x4d, 0x86, 0xfc, 0x08, 0xdd,
	0xa9, 0x3b, 0xac, 0xee, 0xc5, 0x9e, 0x9e, 0xd7, 0x90, 0x7b, 0x74, 0xe4, 0x93, 0x3d, 0x3d, 0xcc,
	0x9e, 0x4f, 0xf6, 0xb1, 0x80, 0x80, 0x97, 0xa9, 0xfb, 0xe8, 0x9f, 0x55, 0x99, 0x5e, 0x0b, 0x95,
	0xd1, 0x15, 0xab, 0xb7, 0xaf, 0xc3, 0x39, 0x4d, 0xde, 0xd6, 0xd4, 0xe3, 0xe9, 0x70, 0x6e, 0xb0,
	0x2f, 0x5b, 0x61, 0xa5, 0x30, 0xee, 0xca, 0xbb, 0x6e, 0x1b, 0xbc, 0x19, 0xd0, 0xff, 0xcd, 0x2f,
	0xff, 0x07, 0x00, 0x00, 0xff, 0xff, 0xe4, 0xf4, 0xb1, 0x55, 0x7f, 0x06, 0x00, 0x00,
}
