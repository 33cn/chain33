// Code generated by protoc-gen-go.
// source: config.proto
// DO NOT EDIT!

package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Config struct {
	Title      string      `protobuf:"bytes,1,opt,name=title" json:"title,omitempty"`
	Log        *Log        `protobuf:"bytes,2,opt,name=log" json:"log,omitempty"`
	Store      *Store      `protobuf:"bytes,3,opt,name=store" json:"store,omitempty"`
	Consensus  *Consensus  `protobuf:"bytes,5,opt,name=consensus" json:"consensus,omitempty"`
	MemPool    *MemPool    `protobuf:"bytes,6,opt,name=memPool" json:"memPool,omitempty"`
	BlockChain *BlockChain `protobuf:"bytes,7,opt,name=blockChain" json:"blockChain,omitempty"`
	Wallet     *Wallet     `protobuf:"bytes,8,opt,name=wallet" json:"wallet,omitempty"`
	P2P        *P2P        `protobuf:"bytes,9,opt,name=p2p" json:"p2p,omitempty"`
	Rpc        *Rpc        `protobuf:"bytes,10,opt,name=rpc" json:"rpc,omitempty"`
	Exec       *Exec       `protobuf:"bytes,11,opt,name=exec" json:"exec,omitempty"`
	TestNet    bool        `protobuf:"varint,12,opt,name=testNet" json:"testNet,omitempty"`
	FixTime    bool        `protobuf:"varint,13,opt,name=fixTime" json:"fixTime,omitempty"`
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

func (m *Config) GetLog() *Log {
	if m != nil {
		return m.Log
	}
	return nil
}

func (m *Config) GetStore() *Store {
	if m != nil {
		return m.Store
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

func (m *Config) GetExec() *Exec {
	if m != nil {
		return m.Exec
	}
	return nil
}

func (m *Config) GetTestNet() bool {
	if m != nil {
		return m.TestNet
	}
	return false
}

func (m *Config) GetFixTime() bool {
	if m != nil {
		return m.FixTime
	}
	return false
}

type Log struct {
	// 日志级别，支持debug(dbug)/info/warn/error(eror)/crit
	Loglevel        string `protobuf:"bytes,1,opt,name=loglevel" json:"loglevel,omitempty"`
	LogConsoleLevel string `protobuf:"bytes,2,opt,name=logConsoleLevel" json:"logConsoleLevel,omitempty"`
	// 日志文件名，可带目录，所有生成的日志文件都放到此目录下
	LogFile string `protobuf:"bytes,3,opt,name=logFile" json:"logFile,omitempty"`
	// 单个日志文件的最大值（单位：兆）
	MaxFileSize uint32 `protobuf:"varint,4,opt,name=maxFileSize" json:"maxFileSize,omitempty"`
	// 最多保存的历史日志文件个数
	MaxBackups uint32 `protobuf:"varint,5,opt,name=maxBackups" json:"maxBackups,omitempty"`
	// 最多保存的历史日志消息（单位：天）
	MaxAge uint32 `protobuf:"varint,6,opt,name=maxAge" json:"maxAge,omitempty"`
	// 日志文件名是否使用本地事件（否则使用UTC时间）
	LocalTime bool `protobuf:"varint,7,opt,name=localTime" json:"localTime,omitempty"`
	// 历史日志文件是否压缩（压缩格式为gz）
	Compress bool `protobuf:"varint,8,opt,name=compress" json:"compress,omitempty"`
	// 是否打印调用源文件和行号
	CallerFile bool `protobuf:"varint,9,opt,name=callerFile" json:"callerFile,omitempty"`
	// 是否打印调用方法
	CallerFunction bool `protobuf:"varint,10,opt,name=callerFunction" json:"callerFunction,omitempty"`
}

func (m *Log) Reset()                    { *m = Log{} }
func (m *Log) String() string            { return proto.CompactTextString(m) }
func (*Log) ProtoMessage()               {}
func (*Log) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{1} }

func (m *Log) GetLoglevel() string {
	if m != nil {
		return m.Loglevel
	}
	return ""
}

func (m *Log) GetLogConsoleLevel() string {
	if m != nil {
		return m.LogConsoleLevel
	}
	return ""
}

func (m *Log) GetLogFile() string {
	if m != nil {
		return m.LogFile
	}
	return ""
}

func (m *Log) GetMaxFileSize() uint32 {
	if m != nil {
		return m.MaxFileSize
	}
	return 0
}

func (m *Log) GetMaxBackups() uint32 {
	if m != nil {
		return m.MaxBackups
	}
	return 0
}

func (m *Log) GetMaxAge() uint32 {
	if m != nil {
		return m.MaxAge
	}
	return 0
}

func (m *Log) GetLocalTime() bool {
	if m != nil {
		return m.LocalTime
	}
	return false
}

func (m *Log) GetCompress() bool {
	if m != nil {
		return m.Compress
	}
	return false
}

func (m *Log) GetCallerFile() bool {
	if m != nil {
		return m.CallerFile
	}
	return false
}

func (m *Log) GetCallerFunction() bool {
	if m != nil {
		return m.CallerFunction
	}
	return false
}

type MemPool struct {
	PoolCacheSize int64 `protobuf:"varint,1,opt,name=poolCacheSize" json:"poolCacheSize,omitempty"`
	MinTxFee      int64 `protobuf:"varint,2,opt,name=minTxFee" json:"minTxFee,omitempty"`
	ForceAccept   bool  `protobuf:"varint,3,opt,name=forceAccept" json:"forceAccept,omitempty"`
}

func (m *MemPool) Reset()                    { *m = MemPool{} }
func (m *MemPool) String() string            { return proto.CompactTextString(m) }
func (*MemPool) ProtoMessage()               {}
func (*MemPool) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{2} }

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

func (m *MemPool) GetForceAccept() bool {
	if m != nil {
		return m.ForceAccept
	}
	return false
}

type Consensus struct {
	Name              string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Genesis           string `protobuf:"bytes,2,opt,name=genesis" json:"genesis,omitempty"`
	Minerstart        bool   `protobuf:"varint,3,opt,name=minerstart" json:"minerstart,omitempty"`
	GenesisBlockTime  int64  `protobuf:"varint,4,opt,name=genesisBlockTime" json:"genesisBlockTime,omitempty"`
	HotkeyAddr        string `protobuf:"bytes,5,opt,name=hotkeyAddr" json:"hotkeyAddr,omitempty"`
	ForceMining       bool   `protobuf:"varint,6,opt,name=forceMining" json:"forceMining,omitempty"`
	NodeId            int64  `protobuf:"varint,7,opt,name=NodeId" json:"NodeId,omitempty"`
	PeersURL          string `protobuf:"bytes,8,opt,name=PeersURL" json:"PeersURL,omitempty"`
	ClientAddr        string `protobuf:"bytes,9,opt,name=ClientAddr" json:"ClientAddr,omitempty"`
	RaftApiPort       int64  `protobuf:"varint,15,opt,name=raftApiPort" json:"raftApiPort,omitempty"`
	IsNewJoinNode     bool   `protobuf:"varint,16,opt,name=isNewJoinNode" json:"isNewJoinNode,omitempty"`
	ReadOnlyPeersURL  string `protobuf:"bytes,17,opt,name=readOnlyPeersURL" json:"readOnlyPeersURL,omitempty"`
	AddPeersURL       string `protobuf:"bytes,18,opt,name=addPeersURL" json:"addPeersURL,omitempty"`
	DefaultSnapCount  int64  `protobuf:"varint,19,opt,name=defaultSnapCount" json:"defaultSnapCount,omitempty"`
	WriteBlockSeconds int64  `protobuf:"varint,20,opt,name=writeBlockSeconds" json:"writeBlockSeconds,omitempty"`
	HeartbeatTick     int32  `protobuf:"varint,21,opt,name=heartbeatTick" json:"heartbeatTick,omitempty"`
}

func (m *Consensus) Reset()                    { *m = Consensus{} }
func (m *Consensus) String() string            { return proto.CompactTextString(m) }
func (*Consensus) ProtoMessage()               {}
func (*Consensus) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{3} }

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

func (m *Consensus) GetForceMining() bool {
	if m != nil {
		return m.ForceMining
	}
	return false
}

func (m *Consensus) GetNodeId() int64 {
	if m != nil {
		return m.NodeId
	}
	return 0
}

func (m *Consensus) GetPeersURL() string {
	if m != nil {
		return m.PeersURL
	}
	return ""
}

func (m *Consensus) GetClientAddr() string {
	if m != nil {
		return m.ClientAddr
	}
	return ""
}

func (m *Consensus) GetRaftApiPort() int64 {
	if m != nil {
		return m.RaftApiPort
	}
	return 0
}

func (m *Consensus) GetIsNewJoinNode() bool {
	if m != nil {
		return m.IsNewJoinNode
	}
	return false
}

func (m *Consensus) GetReadOnlyPeersURL() string {
	if m != nil {
		return m.ReadOnlyPeersURL
	}
	return ""
}

func (m *Consensus) GetAddPeersURL() string {
	if m != nil {
		return m.AddPeersURL
	}
	return ""
}

func (m *Consensus) GetDefaultSnapCount() int64 {
	if m != nil {
		return m.DefaultSnapCount
	}
	return 0
}

func (m *Consensus) GetWriteBlockSeconds() int64 {
	if m != nil {
		return m.WriteBlockSeconds
	}
	return 0
}

func (m *Consensus) GetHeartbeatTick() int32 {
	if m != nil {
		return m.HeartbeatTick
	}
	return 0
}

type Wallet struct {
	MinFee         int64    `protobuf:"varint,1,opt,name=minFee" json:"minFee,omitempty"`
	Driver         string   `protobuf:"bytes,2,opt,name=driver" json:"driver,omitempty"`
	DbPath         string   `protobuf:"bytes,3,opt,name=dbPath" json:"dbPath,omitempty"`
	DbCache        int32    `protobuf:"varint,4,opt,name=dbCache" json:"dbCache,omitempty"`
	SignType       string   `protobuf:"bytes,5,opt,name=signType" json:"signType,omitempty"`
	ForceMining    bool     `protobuf:"varint,6,opt,name=forceMining" json:"forceMining,omitempty"`
	Minerdisable   bool     `protobuf:"varint,7,opt,name=minerdisable" json:"minerdisable,omitempty"`
	Minerwhitelist []string `protobuf:"bytes,8,rep,name=minerwhitelist" json:"minerwhitelist,omitempty"`
}

func (m *Wallet) Reset()                    { *m = Wallet{} }
func (m *Wallet) String() string            { return proto.CompactTextString(m) }
func (*Wallet) ProtoMessage()               {}
func (*Wallet) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{4} }

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

func (m *Wallet) GetDbCache() int32 {
	if m != nil {
		return m.DbCache
	}
	return 0
}

func (m *Wallet) GetSignType() string {
	if m != nil {
		return m.SignType
	}
	return ""
}

func (m *Wallet) GetForceMining() bool {
	if m != nil {
		return m.ForceMining
	}
	return false
}

func (m *Wallet) GetMinerdisable() bool {
	if m != nil {
		return m.Minerdisable
	}
	return false
}

func (m *Wallet) GetMinerwhitelist() []string {
	if m != nil {
		return m.Minerwhitelist
	}
	return nil
}

type Store struct {
	Name    string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Driver  string `protobuf:"bytes,2,opt,name=driver" json:"driver,omitempty"`
	DbPath  string `protobuf:"bytes,3,opt,name=dbPath" json:"dbPath,omitempty"`
	DbCache int32  `protobuf:"varint,4,opt,name=dbCache" json:"dbCache,omitempty"`
}

func (m *Store) Reset()                    { *m = Store{} }
func (m *Store) String() string            { return proto.CompactTextString(m) }
func (*Store) ProtoMessage()               {}
func (*Store) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{5} }

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

func (m *Store) GetDbCache() int32 {
	if m != nil {
		return m.DbCache
	}
	return 0
}

type BlockChain struct {
	DefCacheSize          int64  `protobuf:"varint,1,opt,name=defCacheSize" json:"defCacheSize,omitempty"`
	MaxFetchBlockNum      int64  `protobuf:"varint,2,opt,name=maxFetchBlockNum" json:"maxFetchBlockNum,omitempty"`
	TimeoutSeconds        int64  `protobuf:"varint,3,opt,name=timeoutSeconds" json:"timeoutSeconds,omitempty"`
	BatchBlockNum         int64  `protobuf:"varint,4,opt,name=batchBlockNum" json:"batchBlockNum,omitempty"`
	Driver                string `protobuf:"bytes,5,opt,name=driver" json:"driver,omitempty"`
	DbPath                string `protobuf:"bytes,6,opt,name=dbPath" json:"dbPath,omitempty"`
	DbCache               int32  `protobuf:"varint,7,opt,name=dbCache" json:"dbCache,omitempty"`
	IsStrongConsistency   bool   `protobuf:"varint,8,opt,name=isStrongConsistency" json:"isStrongConsistency,omitempty"`
	SingleMode            bool   `protobuf:"varint,9,opt,name=singleMode" json:"singleMode,omitempty"`
	Batchsync             bool   `protobuf:"varint,10,opt,name=batchsync" json:"batchsync,omitempty"`
	IsRecordBlockSequence bool   `protobuf:"varint,11,opt,name=isRecordBlockSequence" json:"isRecordBlockSequence,omitempty"`
	IsParaChain           bool   `protobuf:"varint,12,opt,name=isParaChain" json:"isParaChain,omitempty"`
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

func (m *BlockChain) GetDbCache() int32 {
	if m != nil {
		return m.DbCache
	}
	return 0
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

func (m *BlockChain) GetBatchsync() bool {
	if m != nil {
		return m.Batchsync
	}
	return false
}

func (m *BlockChain) GetIsRecordBlockSequence() bool {
	if m != nil {
		return m.IsRecordBlockSequence
	}
	return false
}

func (m *BlockChain) GetIsParaChain() bool {
	if m != nil {
		return m.IsParaChain
	}
	return false
}

type P2P struct {
	SeedPort        int32    `protobuf:"varint,1,opt,name=seedPort" json:"seedPort,omitempty"`
	Driver          string   `protobuf:"bytes,2,opt,name=driver" json:"driver,omitempty"`
	DbPath          string   `protobuf:"bytes,3,opt,name=dbPath" json:"dbPath,omitempty"`
	DbCache         int32    `protobuf:"varint,4,opt,name=dbCache" json:"dbCache,omitempty"`
	GrpcLogFile     string   `protobuf:"bytes,5,opt,name=grpcLogFile" json:"grpcLogFile,omitempty"`
	IsSeed          bool     `protobuf:"varint,6,opt,name=isSeed" json:"isSeed,omitempty"`
	ServerStart     bool     `protobuf:"varint,7,opt,name=serverStart" json:"serverStart,omitempty"`
	Seeds           []string `protobuf:"bytes,8,rep,name=seeds" json:"seeds,omitempty"`
	Enable          bool     `protobuf:"varint,9,opt,name=enable" json:"enable,omitempty"`
	MsgCacheSize    int32    `protobuf:"varint,10,opt,name=msgCacheSize" json:"msgCacheSize,omitempty"`
	Version         int32    `protobuf:"varint,11,opt,name=version" json:"version,omitempty"`
	VerMix          int32    `protobuf:"varint,12,opt,name=verMix" json:"verMix,omitempty"`
	VerMax          int32    `protobuf:"varint,13,opt,name=verMax" json:"verMax,omitempty"`
	InnerSeedEnable bool     `protobuf:"varint,14,opt,name=innerSeedEnable" json:"innerSeedEnable,omitempty"`
	InnerBounds     int32    `protobuf:"varint,15,opt,name=innerBounds" json:"innerBounds,omitempty"`
	UseGithub       bool     `protobuf:"varint,16,opt,name=useGithub" json:"useGithub,omitempty"`
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

func (m *P2P) GetDriver() string {
	if m != nil {
		return m.Driver
	}
	return ""
}

func (m *P2P) GetDbPath() string {
	if m != nil {
		return m.DbPath
	}
	return ""
}

func (m *P2P) GetDbCache() int32 {
	if m != nil {
		return m.DbCache
	}
	return 0
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

func (m *P2P) GetServerStart() bool {
	if m != nil {
		return m.ServerStart
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

func (m *P2P) GetInnerSeedEnable() bool {
	if m != nil {
		return m.InnerSeedEnable
	}
	return false
}

func (m *P2P) GetInnerBounds() int32 {
	if m != nil {
		return m.InnerBounds
	}
	return 0
}

func (m *P2P) GetUseGithub() bool {
	if m != nil {
		return m.UseGithub
	}
	return false
}

type Rpc struct {
	JrpcBindAddr      string   `protobuf:"bytes,1,opt,name=jrpcBindAddr" json:"jrpcBindAddr,omitempty"`
	GrpcBindAddr      string   `protobuf:"bytes,2,opt,name=grpcBindAddr" json:"grpcBindAddr,omitempty"`
	Whitlist          []string `protobuf:"bytes,3,rep,name=whitlist" json:"whitlist,omitempty"`
	Whitelist         []string `protobuf:"bytes,4,rep,name=whitelist" json:"whitelist,omitempty"`
	JrpcFuncWhitelist []string `protobuf:"bytes,5,rep,name=jrpcFuncWhitelist" json:"jrpcFuncWhitelist,omitempty"`
	GrpcFuncWhitelist []string `protobuf:"bytes,6,rep,name=grpcFuncWhitelist" json:"grpcFuncWhitelist,omitempty"`
	JrpcFuncBlacklist []string `protobuf:"bytes,7,rep,name=jrpcFuncBlacklist" json:"jrpcFuncBlacklist,omitempty"`
	GrpcFuncBlacklist []string `protobuf:"bytes,8,rep,name=grpcFuncBlacklist" json:"grpcFuncBlacklist,omitempty"`
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

func (m *Rpc) GetWhitelist() []string {
	if m != nil {
		return m.Whitelist
	}
	return nil
}

func (m *Rpc) GetJrpcFuncWhitelist() []string {
	if m != nil {
		return m.JrpcFuncWhitelist
	}
	return nil
}

func (m *Rpc) GetGrpcFuncWhitelist() []string {
	if m != nil {
		return m.GrpcFuncWhitelist
	}
	return nil
}

func (m *Rpc) GetJrpcFuncBlacklist() []string {
	if m != nil {
		return m.JrpcFuncBlacklist
	}
	return nil
}

func (m *Rpc) GetGrpcFuncBlacklist() []string {
	if m != nil {
		return m.GrpcFuncBlacklist
	}
	return nil
}

type Exec struct {
	MinExecFee int64 `protobuf:"varint,1,opt,name=minExecFee" json:"minExecFee,omitempty"`
	IsFree     bool  `protobuf:"varint,2,opt,name=isFree" json:"isFree,omitempty"`
	EnableStat bool  `protobuf:"varint,3,opt,name=enableStat" json:"enableStat,omitempty"`
}

func (m *Exec) Reset()                    { *m = Exec{} }
func (m *Exec) String() string            { return proto.CompactTextString(m) }
func (*Exec) ProtoMessage()               {}
func (*Exec) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{9} }

func (m *Exec) GetMinExecFee() int64 {
	if m != nil {
		return m.MinExecFee
	}
	return 0
}

func (m *Exec) GetIsFree() bool {
	if m != nil {
		return m.IsFree
	}
	return false
}

func (m *Exec) GetEnableStat() bool {
	if m != nil {
		return m.EnableStat
	}
	return false
}

func init() {
	proto.RegisterType((*Config)(nil), "types.Config")
	proto.RegisterType((*Log)(nil), "types.Log")
	proto.RegisterType((*MemPool)(nil), "types.MemPool")
	proto.RegisterType((*Consensus)(nil), "types.Consensus")
	proto.RegisterType((*Wallet)(nil), "types.Wallet")
	proto.RegisterType((*Store)(nil), "types.Store")
	proto.RegisterType((*BlockChain)(nil), "types.BlockChain")
	proto.RegisterType((*P2P)(nil), "types.P2P")
	proto.RegisterType((*Rpc)(nil), "types.Rpc")
	proto.RegisterType((*Exec)(nil), "types.Exec")
}

func init() { proto.RegisterFile("config.proto", fileDescriptor3) }

var fileDescriptor3 = []byte{
	// 1251 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xac, 0x57, 0xed, 0x6e, 0x23, 0x35,
	0x17, 0x56, 0x3a, 0xf9, 0x74, 0xdb, 0xfd, 0x98, 0xdd, 0x7d, 0x35, 0x7a, 0xb5, 0x82, 0x28, 0x02,
	0x14, 0x21, 0x54, 0x41, 0xe1, 0x06, 0xda, 0x6a, 0x17, 0x81, 0xda, 0x12, 0x39, 0x45, 0xfb, 0x0f,
	0x69, 0xe2, 0x39, 0x9d, 0x98, 0x4e, 0xec, 0xc1, 0x76, 0xda, 0x94, 0xfb, 0xe1, 0x02, 0xf8, 0x8f,
	0x84, 0xb8, 0x00, 0xee, 0x86, 0x0b, 0x40, 0xe7, 0xd8, 0x99, 0x8f, 0xb4, 0x2b, 0xf1, 0x63, 0xff,
	0xe5, 0x79, 0xce, 0x33, 0xb6, 0xcf, 0xa7, 0x1d, 0x76, 0x20, 0xb4, 0xba, 0x96, 0xf9, 0x51, 0x69,
	0xb4, 0xd3, 0x71, 0xcf, 0xdd, 0x97, 0x60, 0x27, 0xbf, 0x45, 0xac, 0x7f, 0x46, 0x7c, 0xfc, 0x92,
	0xf5, 0x9c, 0x74, 0x05, 0x24, 0x9d, 0x71, 0x67, 0x3a, 0xe2, 0x1e, 0xc4, 0xaf, 0x59, 0x54, 0xe8,
	0x3c, 0xd9, 0x1b, 0x77, 0xa6, 0xfb, 0xc7, 0xec, 0x88, 0xbe, 0x3a, 0x3a, 0xd7, 0x39, 0x47, 0x3a,
	0x9e, 0xb0, 0x9e, 0x75, 0xda, 0x40, 0x12, 0x91, 0xfd, 0x20, 0xd8, 0xe7, 0xc8, 0x71, 0x6f, 0x8a,
	0x8f, 0xd8, 0x48, 0x68, 0x65, 0x41, 0xd9, 0xb5, 0x4d, 0x7a, 0xa4, 0x7b, 0x16, 0x74, 0x67, 0x5b,
	0x9e, 0xd7, 0x92, 0x78, 0xca, 0x06, 0x2b, 0x58, 0xcd, 0xb4, 0x2e, 0x92, 0x3e, 0xa9, 0x9f, 0x04,
	0xf5, 0x85, 0x67, 0xf9, 0xd6, 0x1c, 0x7f, 0xc5, 0xd8, 0xa2, 0xd0, 0xe2, 0xe6, 0x6c, 0x99, 0x4a,
	0x95, 0x0c, 0x48, 0xfc, 0x3c, 0x88, 0x4f, 0x2b, 0x03, 0x6f, 0x88, 0xe2, 0x4f, 0x59, 0xff, 0x2e,
	0x2d, 0x0a, 0x70, 0xc9, 0x90, 0xe4, 0x87, 0x41, 0xfe, 0x8e, 0x48, 0x1e, 0x8c, 0xe8, 0x75, 0x79,
	0x5c, 0x26, 0xa3, 0x96, 0xd7, 0xb3, 0xe3, 0x19, 0x47, 0x1a, 0xad, 0xa6, 0x14, 0x09, 0x6b, 0x59,
	0x79, 0x29, 0x38, 0xd2, 0xf1, 0xc7, 0xac, 0x0b, 0x1b, 0x10, 0xc9, 0x3e, 0x99, 0xf7, 0x83, 0xf9,
	0xcd, 0x06, 0x04, 0x27, 0x43, 0x9c, 0xb0, 0x81, 0x03, 0xeb, 0x2e, 0xc1, 0x25, 0x07, 0xe3, 0xce,
	0x74, 0xc8, 0xb7, 0x10, 0x2d, 0xd7, 0x72, 0x73, 0x25, 0x57, 0x90, 0x1c, 0x7a, 0x4b, 0x80, 0x93,
	0xbf, 0xf6, 0x58, 0x74, 0xae, 0xf3, 0xf8, 0xff, 0x6c, 0x58, 0xe8, 0xbc, 0x80, 0x5b, 0x28, 0x42,
	0x9e, 0x2a, 0x1c, 0x4f, 0xd9, 0xd3, 0x42, 0xe7, 0x18, 0x53, 0x5d, 0xc0, 0x39, 0x49, 0xf6, 0x48,
	0xb2, 0x4b, 0xe3, 0x3e, 0x85, 0xce, 0xdf, 0xca, 0xc2, 0x27, 0x6e, 0xc4, 0xb7, 0x30, 0x1e, 0xb3,
	0xfd, 0x55, 0xba, 0xc1, 0x9f, 0x73, 0xf9, 0x2b, 0x24, 0xdd, 0x71, 0x67, 0x7a, 0xc8, 0x9b, 0x54,
	0xfc, 0x11, 0x63, 0xab, 0x74, 0x73, 0x9a, 0x8a, 0x9b, 0x75, 0xe9, 0xf3, 0x79, 0xc8, 0x1b, 0x4c,
	0xfc, 0x3f, 0xd6, 0x5f, 0xa5, 0x9b, 0x93, 0x1c, 0x28, 0x7b, 0x87, 0x3c, 0xa0, 0xf8, 0x35, 0x1b,
	0x15, 0x5a, 0xa4, 0x05, 0x79, 0x37, 0x20, 0xef, 0x6a, 0x02, 0xfd, 0x12, 0x7a, 0x55, 0x1a, 0xb0,
	0x96, 0x32, 0x33, 0xe4, 0x15, 0xc6, 0x1d, 0x05, 0xa6, 0xc5, 0xd0, 0x81, 0x47, 0x64, 0x6d, 0x30,
	0xf1, 0x67, 0xec, 0x49, 0x40, 0x6b, 0x25, 0x9c, 0xd4, 0x8a, 0x32, 0x33, 0xe4, 0x3b, 0xec, 0x64,
	0xc5, 0x06, 0xa1, 0x84, 0xe2, 0x4f, 0xd8, 0x61, 0xa9, 0x75, 0x71, 0x96, 0x8a, 0xa5, 0x77, 0x14,
	0x63, 0x19, 0xf1, 0x36, 0x89, 0x87, 0x5a, 0x49, 0x75, 0xb5, 0x79, 0x0b, 0x40, 0x91, 0x8c, 0x78,
	0x85, 0x31, 0x50, 0xd7, 0xda, 0x08, 0x38, 0x11, 0x02, 0x4a, 0x47, 0x61, 0x1c, 0xf2, 0x26, 0x35,
	0xf9, 0xbd, 0xcb, 0x46, 0x55, 0x81, 0xc7, 0x31, 0xeb, 0xaa, 0x74, 0xb5, 0x6d, 0x2e, 0xfa, 0x8d,
	0x69, 0xc8, 0x41, 0x81, 0x95, 0x36, 0x24, 0x6a, 0x0b, 0x29, 0xc8, 0x52, 0x81, 0xb1, 0x2e, 0x35,
	0xdb, 0xc5, 0x1b, 0x4c, 0xfc, 0x39, 0x7b, 0x16, 0xa4, 0x54, 0xe7, 0x14, 0xd3, 0x2e, 0x9d, 0xf0,
	0x01, 0x8f, 0x6b, 0x2d, 0xb5, 0xbb, 0x81, 0xfb, 0x93, 0x2c, 0x33, 0x94, 0xb0, 0x11, 0x6f, 0x30,
	0x95, 0x27, 0x17, 0x52, 0x49, 0x95, 0x53, 0xd6, 0xb6, 0x9e, 0x78, 0x0a, 0x53, 0x7a, 0xa9, 0x33,
	0xf8, 0x2e, 0xa3, 0xbc, 0x45, 0x3c, 0x20, 0x8c, 0xcf, 0x0c, 0xc0, 0xd8, 0x1f, 0xf9, 0x39, 0x25,
	0x6d, 0xc4, 0x2b, 0x8c, 0xbb, 0x9e, 0x15, 0x12, 0x94, 0xa3, 0x5d, 0x47, 0x7e, 0xd7, 0x9a, 0xc1,
	0x5d, 0x4d, 0x7a, 0xed, 0x4e, 0x4a, 0x39, 0xd3, 0xc6, 0x25, 0x4f, 0x69, 0xe1, 0x26, 0x85, 0x39,
	0x92, 0xf6, 0x12, 0xee, 0xbe, 0xd7, 0x52, 0xe1, 0x86, 0xc9, 0x33, 0x3a, 0x59, 0x9b, 0xc4, 0x48,
	0x18, 0x48, 0xb3, 0x1f, 0x54, 0x71, 0x5f, 0x9d, 0xe5, 0x39, 0xed, 0xf6, 0x80, 0xc7, 0x3d, 0xd3,
	0x2c, 0xab, 0x64, 0x31, 0xc9, 0x9a, 0x14, 0xae, 0x96, 0xc1, 0x75, 0xba, 0x2e, 0xdc, 0x5c, 0xa5,
	0xe5, 0x99, 0x5e, 0x2b, 0x97, 0xbc, 0xf0, 0x71, 0xdd, 0xe5, 0xe3, 0x2f, 0xd8, 0xf3, 0x3b, 0x23,
	0x1d, 0x50, 0xa4, 0xe7, 0x20, 0xb4, 0xca, 0x6c, 0xf2, 0x92, 0xc4, 0x0f, 0x0d, 0xe8, 0xcd, 0x12,
	0x52, 0xe3, 0x16, 0x90, 0xba, 0x2b, 0x29, 0x6e, 0x92, 0x57, 0xe3, 0xce, 0xb4, 0xc7, 0xdb, 0xe4,
	0xe4, 0x9f, 0x0e, 0xeb, 0xfb, 0x51, 0x44, 0x7d, 0x24, 0x15, 0x96, 0x9e, 0xaf, 0xcd, 0x80, 0x90,
	0xcf, 0x8c, 0xbc, 0x05, 0x13, 0x6a, 0x26, 0x20, 0xe2, 0x17, 0xb3, 0xd4, 0x2d, 0x43, 0x4b, 0x07,
	0x84, 0x45, 0x96, 0x2d, 0xa8, 0xa6, 0xa9, 0x42, 0x7a, 0x7c, 0x0b, 0x31, 0x7d, 0x56, 0xe6, 0xea,
	0xea, 0xbe, 0x84, 0x50, 0x16, 0x15, 0xfe, 0x0f, 0x45, 0x31, 0x61, 0x07, 0x54, 0x90, 0x99, 0xb4,
	0xe9, 0xa2, 0xd8, 0xb6, 0x74, 0x8b, 0xc3, 0xce, 0x24, 0x7c, 0xb7, 0x94, 0x0e, 0x0a, 0x69, 0x71,
	0xea, 0x46, 0xd3, 0x11, 0xdf, 0x61, 0x27, 0xc0, 0x7a, 0x74, 0x65, 0x3c, 0xda, 0x25, 0x1f, 0xcc,
	0xe1, 0xc9, 0x1f, 0x11, 0x63, 0xf5, 0xbd, 0x80, 0x1e, 0x64, 0x70, 0xbd, 0x3b, 0x03, 0x5a, 0x1c,
	0x16, 0x04, 0x0e, 0x3f, 0x70, 0x62, 0x49, 0x5f, 0x5e, 0xae, 0x57, 0x61, 0x14, 0x3c, 0xe0, 0xd1,
	0x5b, 0x27, 0x57, 0xa0, 0xd7, 0x6e, 0x5b, 0x0d, 0x11, 0x29, 0x77, 0x58, 0x2c, 0x85, 0x45, 0xda,
	0x5c, 0xd0, 0x77, 0x6e, 0x9b, 0x6c, 0xb8, 0xdd, 0x7b, 0x8f, 0xdb, 0xfd, 0xf7, 0xb9, 0x3d, 0x68,
	0xe7, 0xf9, 0x4b, 0xf6, 0x42, 0xda, 0xb9, 0x33, 0x5a, 0xd1, 0x2d, 0x20, 0xad, 0x03, 0x25, 0xee,
	0xc3, 0x98, 0x7d, 0xcc, 0x84, 0xcd, 0x6b, 0xa5, 0xca, 0x0b, 0xb8, 0xc0, 0xbe, 0x0b, 0x13, 0xb7,
	0x66, 0x70, 0x96, 0xd3, 0x61, 0xed, 0xbd, 0x12, 0x61, 0xd8, 0xd6, 0x44, 0xfc, 0x0d, 0x7b, 0x25,
	0x2d, 0x07, 0xa1, 0x4d, 0x16, 0x5a, 0xe0, 0x97, 0x35, 0x28, 0x01, 0x74, 0x23, 0x0e, 0xf9, 0xe3,
	0x46, 0xac, 0x38, 0x69, 0x67, 0xa9, 0x49, 0xfd, 0x6d, 0xee, 0x6f, 0xc6, 0x26, 0x35, 0xf9, 0x3b,
	0x62, 0xd1, 0xec, 0x78, 0x46, 0x75, 0x0b, 0x90, 0xd1, 0xdc, 0xe8, 0x90, 0xab, 0x15, 0xfe, 0x80,
	0xdd, 0x31, 0x66, 0xfb, 0xb9, 0x29, 0xc5, 0x79, 0xb8, 0x27, 0x7d, 0x12, 0x9a, 0x14, 0xae, 0x29,
	0xed, 0x1c, 0x20, 0x0b, 0xed, 0x11, 0x10, 0x7e, 0x69, 0xc1, 0xdc, 0x82, 0x99, 0xd3, 0xf4, 0xf6,
	0x8d, 0xd1, 0xa4, 0xf0, 0xa9, 0x85, 0x27, 0xb6, 0xa1, 0x1d, 0x3c, 0xc0, 0xf5, 0x40, 0x51, 0x2f,
	0xf9, 0x88, 0x07, 0x44, 0x9d, 0x66, 0xf3, 0xba, 0x4e, 0x19, 0x1d, 0xb4, 0xc5, 0xa1, 0x1f, 0xb7,
	0x60, 0x2c, 0x5e, 0x7e, 0xfb, 0xde, 0x8f, 0x00, 0x71, 0xd5, 0x5b, 0x30, 0x17, 0x72, 0x43, 0x21,
	0xed, 0xf1, 0x80, 0xb6, 0x7c, 0xba, 0xa1, 0xa7, 0x46, 0xe0, 0xd3, 0x0d, 0xbe, 0x22, 0xa4, 0x52,
	0x60, 0xd0, 0x95, 0x37, 0xfe, 0x38, 0x4f, 0xe8, 0x38, 0xbb, 0x34, 0x65, 0x0c, 0xa9, 0x53, 0xbd,
	0xc6, 0x62, 0x7f, 0x4a, 0xcb, 0x34, 0x29, 0xac, 0x93, 0xb5, 0x85, 0x6f, 0xa5, 0x5b, 0xae, 0x17,
	0x61, 0x7c, 0xd7, 0xc4, 0xe4, 0xcf, 0x3d, 0x16, 0xf1, 0x52, 0xa0, 0x7f, 0x3f, 0x9b, 0x52, 0x9c,
	0x4a, 0x95, 0xd1, 0x65, 0xe1, 0x9b, 0xbf, 0xc5, 0xa1, 0x26, 0x6f, 0x6a, 0x7c, 0x76, 0x5b, 0x1c,
	0xd6, 0x05, 0x8e, 0x14, 0x9a, 0x33, 0x11, 0x05, 0xb6, 0xc2, 0x78, 0x92, 0x7a, 0x08, 0x75, 0xc9,
	0x58, 0x13, 0x38, 0xca, 0x71, 0x37, 0x7c, 0x29, 0xbc, 0xab, 0x54, 0x3d, 0x52, 0x3d, 0x34, 0xa0,
	0x3a, 0x7f, 0xa0, 0xee, 0x7b, 0x75, 0xfe, 0x98, 0x7a, 0xbb, 0xc4, 0x69, 0x91, 0x8a, 0x1b, 0x52,
	0x0f, 0xda, 0x6b, 0x57, 0x86, 0xe6, 0xda, 0xb5, 0x7a, 0xd8, 0x5e, 0xbb, 0x32, 0x4c, 0x7e, 0x62,
	0x5d, 0x7c, 0x57, 0x86, 0xe7, 0x02, 0xfe, 0xac, 0xef, 0x8b, 0x06, 0xe3, 0x2b, 0xf5, 0xad, 0x09,
	0xcf, 0x18, 0xaa, 0x54, 0x44, 0xf8, 0x9d, 0xaf, 0xb1, 0xb9, 0x4b, 0xab, 0x67, 0x46, 0xcd, 0x2c,
	0xfa, 0xf4, 0x5f, 0xe1, 0xeb, 0x7f, 0x03, 0x00, 0x00, 0xff, 0xff, 0x9f, 0xb4, 0xea, 0xbb, 0x3b,
	0x0c, 0x00, 0x00,
}
