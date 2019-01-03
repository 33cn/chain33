// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

// Config 配置信息
type Config struct {
	Title      string       `protobuf:"bytes,1,opt,name=title" json:"title,omitempty"`
	Version    string       `protobuf:"bytes,1,opt,name=version" json:"version,omitempty"`
	Log        *Log         `protobuf:"bytes,2,opt,name=log" json:"log,omitempty"`
	Store      *Store       `protobuf:"bytes,3,opt,name=store" json:"store,omitempty"`
	Consensus  *Consensus   `protobuf:"bytes,5,opt,name=consensus" json:"consensus,omitempty"`
	Mempool    *Mempool     `protobuf:"bytes,6,opt,name=mempool" json:"memPool,omitempty"`
	BlockChain *BlockChain  `protobuf:"bytes,7,opt,name=blockChain" json:"blockChain,omitempty"`
	Wallet     *Wallet      `protobuf:"bytes,8,opt,name=wallet" json:"wallet,omitempty"`
	P2P        *P2P         `protobuf:"bytes,9,opt,name=p2p" json:"p2p,omitempty"`
	RPC        *RPC         `protobuf:"bytes,10,opt,name=rpc" json:"rpc,omitempty"`
	Exec       *Exec        `protobuf:"bytes,11,opt,name=exec" json:"exec,omitempty"`
	TestNet    bool         `protobuf:"varint,12,opt,name=testNet" json:"testNet,omitempty"`
	FixTime    bool         `protobuf:"varint,13,opt,name=fixTime" json:"fixTime,omitempty"`
	Pprof      *Pprof       `protobuf:"bytes,14,opt,name=pprof" json:"pprof,omitempty"`
	Fork       *ForkList    `protobuf:"bytes,15,opt,name=fork" json:"fork,omitempty"`
	Health     *HealthCheck `protobuf:"bytes,16,opt,name=health" json:"health,omitempty"`
}

// ForkList fork列表配置
type ForkList struct {
	System map[string]int64            `protobuf:"bytes,1,rep,name=system" json:"system,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value"`
	Sub    map[string]map[string]int64 `protobuf:"bytes,2,rep,name=sub" json:"sub,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value"`
}

// Log 日志配置
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

// Mempool 配置
type Mempool struct {
	Name               string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	PoolCacheSize      int64  `protobuf:"varint,1,opt,name=poolCacheSize" json:"poolCacheSize,omitempty"`
	MinTxFee           int64  `protobuf:"varint,2,opt,name=minTxFee" json:"minTxFee,omitempty"`
	ForceAccept        bool   `protobuf:"varint,3,opt,name=forceAccept" json:"forceAccept,omitempty"`
	MaxTxNumPerAccount int64  `protobuf:"varint,4,opt,name=maxTxNumPerAccount" json:"maxTxNumPerAccount,omitempty"`
	MaxTxLast          int64  `protobuf:"varint,4,opt,name=maxTxLast" json:"maxTxLast,omitempty"`
}

// Consensus 配置
type Consensus struct {
	Name                        string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	GenesisBlockTime            int64  `protobuf:"varint,2,opt,name=genesisBlockTime" json:"genesisBlockTime,omitempty"`
	Minerstart                  bool   `protobuf:"varint,3,opt,name=minerstart" json:"minerstart,omitempty"`
	Genesis                     string `protobuf:"bytes,4,opt,name=genesis" json:"genesis,omitempty"`
	HotkeyAddr                  string `protobuf:"bytes,5,opt,name=hotkeyAddr" json:"hotkeyAddr,omitempty"`
	ForceMining                 bool   `protobuf:"varint,6,opt,name=forceMining" json:"forceMining,omitempty"`
	WriteBlockSeconds           int64  `protobuf:"varint,20,opt,name=writeBlockSeconds" json:"writeBlockSeconds,omitempty"`
	ParaRemoteGrpcClient        string `protobuf:"bytes,22,opt,name=paraRemoteGrpcClient" json:"paraRemoteGrpcClient,omitempty"`
	StartHeight                 int64  `protobuf:"varint,23,opt,name=startHeight" json:"startHeight,omitempty"`
	EmptyBlockInterval          int64  `protobuf:"varint,24,opt,name=emptyBlockInterval" json:"emptyBlockInterval,omitempty"`
	AuthAccount                 string `protobuf:"bytes,25,opt,name=authAccount" json:"authAccount,omitempty"`
	WaitBlocks4CommitMsg        int32  `protobuf:"varint,26,opt,name=waitBlocks4CommitMsg" json:"waitBlocks4CommitMsg,omitempty"`
	SearchHashMatchedBlockDepth int32  `protobuf:"varint,27,opt,name=searchHashMatchedBlockDepth" json:"searchHashMatchedBlockDepth,omitempty"`
}

// Wallet 配置
type Wallet struct {
	MinFee   int64  `protobuf:"varint,1,opt,name=minFee" json:"minFee,omitempty"`
	Driver   string `protobuf:"bytes,2,opt,name=driver" json:"driver,omitempty"`
	DbPath   string `protobuf:"bytes,3,opt,name=dbPath" json:"dbPath,omitempty"`
	DbCache  int32  `protobuf:"varint,4,opt,name=dbCache" json:"dbCache,omitempty"`
	SignType string `protobuf:"bytes,5,opt,name=signType" json:"signType,omitempty"`
}

// Store 配置
type Store struct {
	Name           string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Driver         string `protobuf:"bytes,2,opt,name=driver" json:"driver,omitempty"`
	DbPath         string `protobuf:"bytes,3,opt,name=dbPath" json:"dbPath,omitempty"`
	DbCache        int32  `protobuf:"varint,4,opt,name=dbCache" json:"dbCache,omitempty"`
	LocalDBVersion string `protobuf:"bytes,5,opt,name=localdbVersion" json:"localdbVersion,omitempty"`
}

// BlockChain 配置
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
	EnableTxQuickIndex    bool   `protobuf:"varint,13,opt,name=enableTxQuickIndex" json:"enableTxQuickIndex,omitempty"`
}

// P2P 配置
type P2P struct {
	Port            int32    `protobuf:"varint,1,opt,name=port" json:"port,omitempty"`
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
	VerMin          int32    `protobuf:"varint,12,opt,name=verMin" json:"verMin,omitempty"`
	VerMax          int32    `protobuf:"varint,13,opt,name=verMax" json:"verMax,omitempty"`
	InnerSeedEnable bool     `protobuf:"varint,14,opt,name=innerSeedEnable" json:"innerSeedEnable,omitempty"`
	InnerBounds     int32    `protobuf:"varint,15,opt,name=innerBounds" json:"innerBounds,omitempty"`
	UseGithub       bool     `protobuf:"varint,16,opt,name=useGithub" json:"useGithub,omitempty"`
}

// RPC 配置
type RPC struct {
	JrpcBindAddr      string   `protobuf:"bytes,1,opt,name=jrpcBindAddr" json:"jrpcBindAddr,omitempty"`
	GrpcBindAddr      string   `protobuf:"bytes,2,opt,name=grpcBindAddr" json:"grpcBindAddr,omitempty"`
	Whitlist          []string `protobuf:"bytes,3,rep,name=whitlist" json:"whitlist,omitempty"`
	Whitelist         []string `protobuf:"bytes,4,rep,name=whitelist" json:"whitelist,omitempty"`
	JrpcFuncWhitelist []string `protobuf:"bytes,5,rep,name=jrpcFuncWhitelist" json:"jrpcFuncWhitelist,omitempty"`
	GrpcFuncWhitelist []string `protobuf:"bytes,6,rep,name=grpcFuncWhitelist" json:"grpcFuncWhitelist,omitempty"`
	JrpcFuncBlacklist []string `protobuf:"bytes,7,rep,name=jrpcFuncBlacklist" json:"jrpcFuncBlacklist,omitempty"`
	GrpcFuncBlacklist []string `protobuf:"bytes,8,rep,name=grpcFuncBlacklist" json:"grpcFuncBlacklist,omitempty"`
	MainnetJrpcAddr   string   `protobuf:"bytes,9,opt,name=mainnetJrpcAddr" json:"mainnetJrpcAddr,omitempty"`
	EnableTLS         bool     `protobuf:"varint,10,opt,name=enableTLS" json:"enableTLS,omitempty"`
	CertFile          string   `protobuf:"varint,11,opt,name=certFile" json:"certFile,omitempty"`
	KeyFile           string   `protobuf:"varint,12,opt,name=keyFile" json:"keyFile,omitempty"`
}

// Exec 配置
type Exec struct {
	MinExecFee       int64    `protobuf:"varint,1,opt,name=minExecFee" json:"minExecFee,omitempty"`
	IsFree           bool     `protobuf:"varint,2,opt,name=isFree" json:"isFree,omitempty"`
	EnableStat       bool     `protobuf:"varint,3,opt,name=enableStat" json:"enableStat,omitempty"`
	EnableMVCC       bool     `protobuf:"varint,4,opt,name=enableMVCC" json:"enableMVCC,omitempty"`
	DisableAddrIndex bool     `protobuf:"varint,7,opt,name=disableAddrIndex" json:"disableAddrIndex,omitempty"`
	Alias            []string `protobuf:"bytes,5,rep,name=alias" json:"alias,omitempty"`
	SaveTokenTxList  bool     `protobuf:"varint,6,opt,name=saveTokenTxList" json:"saveTokenTxList,omitempty"`
}

// Pprof 配置
type Pprof struct {
	ListenAddr string `protobuf:"bytes,1,opt,name=listenAddr" json:"listenAddr,omitempty"`
}

// HealthCheck 配置
type HealthCheck struct {
	ListenAddr     string `protobuf:"bytes,1,opt,name=listenAddr" json:"listenAddr,omitempty"`
	CheckInterval  uint32 `protobuf:"varint,2,opt,name=checkInterval" json:"checkInterval,omitempty"`
	UnSyncMaxTimes uint32 `protobuf:"varint,3,opt,name=unSyncMaxTimes" json:"unSyncMaxTimes,omitempty"`
}
