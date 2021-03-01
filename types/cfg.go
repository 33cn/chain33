// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

// Config 配置信息
type Config struct {
	Title          string       `json:"title,omitempty"`
	Version        string       `json:"version,omitempty"`
	Log            *Log         `json:"log,omitempty"`
	Store          *Store       `json:"store,omitempty"`
	Consensus      *Consensus   `json:"consensus,omitempty"`
	Mempool        *Mempool     `json:"memPool,omitempty"`
	BlockChain     *BlockChain  `json:"blockChain,omitempty"`
	Wallet         *Wallet      `json:"wallet,omitempty"`
	P2P            *P2P         `json:"p2p,omitempty"`
	RPC            *RPC         `json:"rpc,omitempty"`
	Exec           *Exec        `json:"exec,omitempty"`
	TestNet        bool         `json:"testNet,omitempty"`
	FixTime        bool         `json:"fixTime,omitempty"`
	TxHeight       bool         `json:"txHeight,omitempty"`
	Pprof          *Pprof       `json:"pprof,omitempty"`
	Fork           *ForkList    `json:"fork,omitempty"`
	Health         *HealthCheck `json:"health,omitempty"`
	CoinSymbol     string       `json:"coinSymbol,omitempty"`
	EnableParaFork bool         `json:"enableParaFork,omitempty"`
	Metrics        *Metrics     `json:"metrics,omitempty"`
	ChainID        int32        `json:"chainID,omitempty"`
	AddrVer        byte         `json:"addrVer,omitempty"`
}

// ForkList fork列表配置
type ForkList struct {
	System map[string]int64
	Sub    map[string]map[string]int64
}

// Log 日志配置
type Log struct {
	// 日志级别，支持debug(dbug)/info/warn/error(eror)/crit
	Loglevel        string `json:"loglevel,omitempty"`
	LogConsoleLevel string `json:"logConsoleLevel,omitempty"`
	// 日志文件名，可带目录，所有生成的日志文件都放到此目录下
	LogFile string `json:"logFile,omitempty"`
	// 单个日志文件的最大值（单位：兆）
	MaxFileSize uint32 `json:"maxFileSize,omitempty"`
	// 最多保存的历史日志文件个数
	MaxBackups uint32 `json:"maxBackups,omitempty"`
	// 最多保存的历史日志消息（单位：天）
	MaxAge uint32 `json:"maxAge,omitempty"`
	// 日志文件名是否使用本地事件（否则使用UTC时间）
	LocalTime bool `json:"localTime,omitempty"`
	// 历史日志文件是否压缩（压缩格式为gz）
	Compress bool `json:"compress,omitempty"`
	// 是否打印调用源文件和行号
	CallerFile bool `json:"callerFile,omitempty"`
	// 是否打印调用方法
	CallerFunction bool `json:"callerFunction,omitempty"`
}

// Mempool 配置
type Mempool struct {
	// mempool队列名称，可配，timeline，score，price
	Name string `json:"name,omitempty"`
	// mempool缓存容量大小，默认10240
	PoolCacheSize int64 `json:"poolCacheSize,omitempty"`
	ForceAccept   bool  `json:"forceAccept,omitempty"`
	// 每个账户在mempool中得最大交易数量，默认100
	MaxTxNumPerAccount int64 `json:"maxTxNumPerAccount,omitempty"`
	MaxTxLast          int64 `json:"maxTxLast,omitempty"`
	IsLevelFee         bool  `json:"isLevelFee,omitempty"`
	// 最小单元交易费，这个没有默认值，必填，一般是100000
	MinTxFeeRate int64 `json:"minTxFeeRate,omitempty"`
	// 最大单元交易费, 默认1e7
	MaxTxFeeRate int64 `json:"maxTxFeeRate,omitempty"`
	// 单笔最大交易费, 默认1e9
	MaxTxFee int64 `json:"maxTxFee,omitempty"`
	// 目前execCheck效率较低，支持关闭交易execCheck，提升性能
	DisableExecCheck bool `json:"disableExecCheck,omitempty"`
}

// Consensus 配置
type Consensus struct {
	// 共识名称 ：solo, ticket, raft, tendermint, para
	Name string `json:"name,omitempty"`
	// 创世区块时间(UTC时间)
	GenesisBlockTime int64 `json:"genesisBlockTime,omitempty"`
	// 是否开启挖矿,开启挖矿才能创建区块
	Minerstart bool `json:"minerstart,omitempty"`
	// 创世交易地址
	Genesis     string `json:"genesis,omitempty"`
	HotkeyAddr  string `json:"hotkeyAddr,omitempty"`
	ForceMining bool   `json:"forceMining,omitempty"`
	// 配置挖矿的合约名单
	MinerExecs []string `json:"minerExecs,omitempty"`
	// 最优区块选择
	EnableBestBlockCmp bool `json:"enableBestBlockCmp,omitempty"`
}

// Wallet 配置
type Wallet struct {
	// 交易发送最低手续费，单位0.00000001BTY(1e-8),默认100000，即0.001BTY
	MinFee int64 `json:"minFee,omitempty"`
	// walletdb驱动名
	Driver string `json:"driver,omitempty"`
	// walletdb路径
	DbPath string `json:"dbPath,omitempty"`
	// walletdb缓存大小
	DbCache int32 `json:"dbCache,omitempty"`
	// 钱包发送交易签名方式
	SignType string `json:"signType,omitempty"`
	CoinType string `json:"coinType,omitempty"`
}

// Store 配置
type Store struct {
	// 数据存储格式名称，目前支持mavl,kvdb,kvmvcc,mpt
	Name string `json:"name,omitempty"`
	// 数据存储驱动类别，目前支持leveldb,goleveldb,memdb,gobadgerdb,ssdb,pegasus
	Driver string `json:"driver,omitempty"`
	// 数据文件存储路径
	DbPath string `json:"dbPath,omitempty"`
	// Cache大小
	DbCache int32 `json:"dbCache,omitempty"`
	// 数据库版本
	LocalDBVersion string `json:"localdbVersion,omitempty"`
	// 数据库版本
	StoreDBVersion string `json:"storedbVersion,omitempty"`
}

// BlockChain 配置
type BlockChain struct {
	// 分片存储中每个大块包含的区块数
	ChunkblockNum int64 `json:"chunkblockNum,omitempty"`
	// 缓存区块的个数
	DefCacheSize int64 `json:"defCacheSize,omitempty"`
	// 同步区块时一次最多申请获取的区块个数
	MaxFetchBlockNum int64 `json:"maxFetchBlockNum,omitempty"`
	// 向对端节点请求同步区块的时间间隔
	TimeoutSeconds int64 `json:"timeoutSeconds,omitempty"`
	BatchBlockNum  int64 `json:"batchBlockNum,omitempty"`
	// 使用的数据库类型
	Driver string `json:"driver,omitempty"`
	// 数据库文件目录
	DbPath string `json:"dbPath,omitempty"`
	// 数据库缓存大小
	DbCache             int32 `json:"dbCache,omitempty"`
	IsStrongConsistency bool  `json:"isStrongConsistency,omitempty"`
	// 是否为单节点
	SingleMode bool `json:"singleMode,omitempty"`
	// 同步区块批量写数据库时，是否需要立即写磁盘，非固态硬盘的电脑可以设置为false，以提高性能
	Batchsync bool `json:"batchsync,omitempty"`
	// 是否记录添加或者删除区块的序列，若节点作为主链节点，为平行链节点提供服务，需要设置为true
	IsRecordBlockSequence bool `json:"isRecordBlockSequence,omitempty"`
	// 是否为平行链节点
	IsParaChain        bool `json:"isParaChain,omitempty"`
	EnableTxQuickIndex bool `json:"enableTxQuickIndex,omitempty"`
	// 升级storedb是否重新执行localdb
	EnableReExecLocal bool `json:"enableReExecLocal,omitempty"`
	// 区块回退
	RollbackBlock int64 `json:"rollbackBlock,omitempty"`
	// 回退是否保存区块
	RollbackSave bool `json:"rollbackSave,omitempty"`
	// 最新区块上链超时时间，单位秒。
	OnChainTimeout int64 `json:"onChainTimeout,omitempty"`
	// 使能精简localdb
	EnableReduceLocaldb bool `json:"enableReduceLocaldb,omitempty"`
	// 关闭分片存储,默认开启分片存储为false;平行链不需要分片需要修改此默认参数为true
	DisableShard bool `protobuf:"varint,19,opt,name=disableShard" json:"disableShard,omitempty"`
	// 使能从P2pStore中获取数据
	EnableFetchP2pstore bool `json:"enableFetchP2pstore,omitempty"`
	// 使能假设已删除已归档数据后,获取数据情况
	EnableIfDelLocalChunk bool `json:"enableIfDelLocalChunk,omitempty"`
	// 使能注册推送区块、区块头或交易回执
	EnablePushSubscribe bool `json:"EnablePushSubscribe,omitempty"`
	// 当前活跃区块的缓存数量
	MaxActiveBlockNum int `json:"maxActiveBlockNum,omitempty"`
	// 当前活跃区块的缓存大小M为单位
	MaxActiveBlockSize int `json:"maxActiveBlockSize,omitempty"`

	//HighAllowPackHeight 允许打包的High区块高度
	HighAllowPackHeight int64 `json:"highAllowPackHeight,omitempty"`

	//LowAllowPackHeight 允许打包的low区块高度
	LowAllowPackHeight int64 `json:"lowAllowPackHeight,omitempty"`
}

// P2P 配置
type P2P struct {
	// 使用的数据库类型
	Driver string `json:"driver,omitempty"`
	// 数据库文件目录
	DbPath string `json:"dbPath,omitempty"`
	// 数据库缓存大小
	DbCache int32 `json:"dbCache,omitempty"`
	// GRPC请求日志文件
	GrpcLogFile string `json:"grpcLogFile,omitempty"`
	// 是否启动P2P服务
	Enable bool `json:"enable,omitempty"`
	//是否等待Pid
	WaitPid bool `json:"waitPid,omitempty"`
	//指定p2p类型, 支持gossip, dht
	Types []string `json:"types,omitempty"`
}

// RPC 配置
type RPC struct {
	// jrpc绑定地址
	JrpcBindAddr string `json:"jrpcBindAddr,omitempty"`
	// grpc绑定地址
	GrpcBindAddr string `json:"grpcBindAddr,omitempty"`
	// 白名单列表，允许访问的IP地址，默认是“*”，允许所有IP访问
	Whitlist  []string `json:"whitlist,omitempty"`
	Whitelist []string `json:"whitelist,omitempty"`
	// jrpc方法请求白名单，默认是“*”，允许访问所有RPC方法
	JrpcFuncWhitelist []string `json:"jrpcFuncWhitelist,omitempty"`
	// grpc方法请求白名单，默认是“*”，允许访问所有RPC方法
	GrpcFuncWhitelist []string `json:"grpcFuncWhitelist,omitempty"`
	// jrpc方法请求黑名单，禁止调用黑名单里配置的rpc方法，一般和白名单配合使用，默认是空
	JrpcFuncBlacklist []string `json:"jrpcFuncBlacklist,omitempty"`
	// grpc方法请求黑名单，禁止调用黑名单里配置的rpc方法，一般和白名单配合使用，默认是空
	GrpcFuncBlacklist []string `json:"grpcFuncBlacklist,omitempty"`
	// 是否开启https
	EnableTLS   bool `json:"enableTLS,omitempty"`
	EnableTrace bool `json:"enableTrace,omitempty"`
	// 证书文件，证书和私钥文件可以用cli工具生成
	CertFile string `json:"certFile,omitempty"`
	// 私钥文件
	KeyFile string `json:"keyFile,omitempty"`
	//basic auth 用户名
	JrpcUserName string `json:"jrpcUserName,omitempty"`
	//basic auth 用户密码
	JrpcUserPasswd string `json:"jrpcUserPasswd,omitempty"`
}

// Exec 配置
type Exec struct {
	// 是否开启stat插件
	EnableStat bool `json:"enableStat,omitempty"`
	// 是否开启MVCC插件
	EnableMVCC       bool     `json:"enableMVCC,omitempty"`
	DisableAddrIndex bool     `json:"disableAddrIndex,omitempty"`
	Alias            []string `json:"alias,omitempty"`
	// 是否保存token交易信息
	SaveTokenTxList bool `json:"saveTokenTxList,omitempty"`
}

// Pprof 配置
type Pprof struct {
	ListenAddr string `json:"listenAddr,omitempty"`
}

// HealthCheck 配置
type HealthCheck struct {
	ListenAddr     string `json:"listenAddr,omitempty"`
	CheckInterval  uint32 `json:"checkInterval,omitempty"`
	UnSyncMaxTimes uint32 `json:"unSyncMaxTimes,omitempty"`
}

// Metrics 相关测量配置信息
type Metrics struct {
	EnableMetrics bool   `json:"enableMetrics,omitempty"`
	DataEmitMode  string `json:"dataEmitMode,omitempty"`
	Duration      int64  `json:"duration,omitempty"`
	URL           string `json:"url,omitempty"`
	DatabaseName  string `json:"databaseName,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
}
