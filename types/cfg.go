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
	CoinSymbol string       `protobuf:"bytes,16,opt,name=coinSymbol" json:"coinSymbol,omitempty"`
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
	// mempool队列名称，可配，timeline，score，price
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// mempool缓存容量大小，默认10240
	PoolCacheSize int64 `protobuf:"varint,2,opt,name=poolCacheSize" json:"poolCacheSize,omitempty"`
	// 最小得交易手续费用，这个没有默认值，必填，一般是100000
	MinTxFee    int64 `protobuf:"varint,3,opt,name=minTxFee" json:"minTxFee,omitempty"`
	MaxTxFee    int64 `protobuf:"varint,3,opt,name=maxTxFee" json:"maxTxFee,omitempty"`
	ForceAccept bool  `protobuf:"varint,4,opt,name=forceAccept" json:"forceAccept,omitempty"`
	// 每个账户在mempool中得最大交易数量，默认100
	MaxTxNumPerAccount int64 `protobuf:"varint,5,opt,name=maxTxNumPerAccount" json:"maxTxNumPerAccount,omitempty"`
	MaxTxLast          int64 `protobuf:"varint,6,opt,name=maxTxLast" json:"maxTxLast,omitempty"`
}

// Consensus 配置
type Consensus struct {
	// 共识名称 ：solo, ticket, raft, tendermint, para
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// 创世区块时间(UTC时间)
	GenesisBlockTime int64 `protobuf:"varint,2,opt,name=genesisBlockTime" json:"genesisBlockTime,omitempty"`
	// 是否开启挖矿,开启挖矿才能创建区块
	Minerstart bool `protobuf:"varint,3,opt,name=minerstart" json:"minerstart,omitempty"`
	// 创世交易地址
	Genesis     string `protobuf:"bytes,4,opt,name=genesis" json:"genesis,omitempty"`
	HotkeyAddr  string `protobuf:"bytes,5,opt,name=hotkeyAddr" json:"hotkeyAddr,omitempty"`
	ForceMining bool   `protobuf:"varint,6,opt,name=forceMining" json:"forceMining,omitempty"`
	// 配置挖矿的合约名单
	MinerExecs []string `protobuf:"bytes,7,rep,name=minerExecs" json:"minerExecs,omitempty"`
}

// Wallet 配置
type Wallet struct {
	// 交易发送最低手续费，单位0.00000001BTY(1e-8),默认100000，即0.001BTY
	MinFee int64 `protobuf:"varint,1,opt,name=minFee" json:"minFee,omitempty"`
	// walletdb驱动名
	Driver string `protobuf:"bytes,2,opt,name=driver" json:"driver,omitempty"`
	// walletdb路径
	DbPath string `protobuf:"bytes,3,opt,name=dbPath" json:"dbPath,omitempty"`
	// walletdb缓存大小
	DbCache int32 `protobuf:"varint,4,opt,name=dbCache" json:"dbCache,omitempty"`
	// 钱包发送交易签名方式
	SignType string `protobuf:"bytes,5,opt,name=signType" json:"signType,omitempty"`
}

// Store 配置
type Store struct {
	// 数据存储格式名称，目前支持mavl,kvdb,kvmvcc,mpt
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// 数据存储驱动类别，目前支持leveldb,goleveldb,memdb,gobadgerdb,ssdb,pegasus
	Driver string `protobuf:"bytes,2,opt,name=driver" json:"driver,omitempty"`
	// 数据文件存储路径
	DbPath string `protobuf:"bytes,3,opt,name=dbPath" json:"dbPath,omitempty"`
	// Cache大小
	DbCache int32 `protobuf:"varint,4,opt,name=dbCache" json:"dbCache,omitempty"`
	// 数据库版本
	LocalDBVersion string `protobuf:"bytes,5,opt,name=localdbVersion" json:"localdbVersion,omitempty"`
	// 数据库版本
	StoreDBVersion string `protobuf:"bytes,5,opt,name=storedbVersion" json:"storedbVersion,omitempty"`
}

// BlockChain 配置
type BlockChain struct {
	// 缓存区块的个数
	DefCacheSize int64 `protobuf:"varint,1,opt,name=defCacheSize" json:"defCacheSize,omitempty"`
	// 同步区块时一次最多申请获取的区块个数
	MaxFetchBlockNum int64 `protobuf:"varint,2,opt,name=maxFetchBlockNum" json:"maxFetchBlockNum,omitempty"`
	// 向对端节点请求同步区块的时间间隔
	TimeoutSeconds int64 `protobuf:"varint,3,opt,name=timeoutSeconds" json:"timeoutSeconds,omitempty"`
	BatchBlockNum  int64 `protobuf:"varint,4,opt,name=batchBlockNum" json:"batchBlockNum,omitempty"`
	// 使用的数据库类型
	Driver string `protobuf:"bytes,5,opt,name=driver" json:"driver,omitempty"`
	// 数据库文件目录
	DbPath string `protobuf:"bytes,6,opt,name=dbPath" json:"dbPath,omitempty"`
	// 数据库缓存大小
	DbCache             int32 `protobuf:"varint,7,opt,name=dbCache" json:"dbCache,omitempty"`
	IsStrongConsistency bool  `protobuf:"varint,8,opt,name=isStrongConsistency" json:"isStrongConsistency,omitempty"`
	// 是否为单节点
	SingleMode bool `protobuf:"varint,9,opt,name=singleMode" json:"singleMode,omitempty"`
	// 同步区块批量写数据库时，是否需要立即写磁盘，非固态硬盘的电脑可以设置为false，以提高性能
	Batchsync bool `protobuf:"varint,10,opt,name=batchsync" json:"batchsync,omitempty"`
	// 是否记录添加或者删除区块的序列，若节点作为主链节点，为平行链节点提供服务，需要设置为true
	IsRecordBlockSequence bool `protobuf:"varint,11,opt,name=isRecordBlockSequence" json:"isRecordBlockSequence,omitempty"`
	// 是否为平行链节点
	IsParaChain        bool `protobuf:"varint,12,opt,name=isParaChain" json:"isParaChain,omitempty"`
	EnableTxQuickIndex bool `protobuf:"varint,13,opt,name=enableTxQuickIndex" json:"enableTxQuickIndex,omitempty"`
}

// P2P 配置
type P2P struct {
	// P2P服务监听端口号
	Port int32 `protobuf:"varint,1,opt,name=port" json:"port,omitempty"`
	// 使用的数据库类型
	Driver string `protobuf:"bytes,2,opt,name=driver" json:"driver,omitempty"`
	// 数据库文件目录
	DbPath string `protobuf:"bytes,3,opt,name=dbPath" json:"dbPath,omitempty"`
	// 数据库缓存大小
	DbCache int32 `protobuf:"varint,4,opt,name=dbCache" json:"dbCache,omitempty"`
	// GRPC请求日志文件
	GrpcLogFile string `protobuf:"bytes,5,opt,name=grpcLogFile" json:"grpcLogFile,omitempty"`
	// 是否为种子节点
	IsSeed bool `protobuf:"varint,6,opt,name=isSeed" json:"isSeed,omitempty"`
	// 是否作为服务端，对外提供服务
	ServerStart bool `protobuf:"varint,7,opt,name=serverStart" json:"serverStart,omitempty"`
	// 种子节点，格式为ip:port，多个节点以逗号分隔，如seeds=
	Seeds []string `protobuf:"bytes,8,rep,name=seeds" json:"seeds,omitempty"`
	// 是否启动P2P服务
	Enable       bool  `protobuf:"varint,9,opt,name=enable" json:"enable,omitempty"`
	MsgCacheSize int32 `protobuf:"varint,10,opt,name=msgCacheSize" json:"msgCacheSize,omitempty"`
	// 版本号
	Version int32 `protobuf:"varint,11,opt,name=version" json:"version,omitempty"`
	VerMin  int32 `protobuf:"varint,12,opt,name=verMin" json:"verMin,omitempty"`
	VerMax  int32 `protobuf:"varint,13,opt,name=verMax" json:"verMax,omitempty"`
	// 是否使用内置的种子节点
	InnerSeedEnable bool `protobuf:"varint,14,opt,name=innerSeedEnable" json:"innerSeedEnable,omitempty"`
	// 最多的接入节点个数
	InnerBounds int32 `protobuf:"varint,15,opt,name=innerBounds" json:"innerBounds,omitempty"`
	// 是否使用Github获取种子节点
	UseGithub bool `protobuf:"varint,16,opt,name=useGithub" json:"useGithub,omitempty"`
}

// RPC 配置
type RPC struct {
	// jrpc绑定地址
	JrpcBindAddr string `protobuf:"bytes,1,opt,name=jrpcBindAddr" json:"jrpcBindAddr,omitempty"`
	// grpc绑定地址
	GrpcBindAddr string `protobuf:"bytes,2,opt,name=grpcBindAddr" json:"grpcBindAddr,omitempty"`
	// 白名单列表，允许访问的IP地址，默认是“*”，允许所有IP访问
	Whitlist  []string `protobuf:"bytes,3,rep,name=whitlist" json:"whitlist,omitempty"`
	Whitelist []string `protobuf:"bytes,4,rep,name=whitelist" json:"whitelist,omitempty"`
	// jrpc方法请求白名单，默认是“*”，允许访问所有RPC方法
	JrpcFuncWhitelist []string `protobuf:"bytes,5,rep,name=jrpcFuncWhitelist" json:"jrpcFuncWhitelist,omitempty"`
	// grpc方法请求白名单，默认是“*”，允许访问所有RPC方法
	GrpcFuncWhitelist []string `protobuf:"bytes,6,rep,name=grpcFuncWhitelist" json:"grpcFuncWhitelist,omitempty"`
	// jrpc方法请求黑名单，禁止调用黑名单里配置的rpc方法，一般和白名单配合使用，默认是空
	JrpcFuncBlacklist []string `protobuf:"bytes,7,rep,name=jrpcFuncBlacklist" json:"jrpcFuncBlacklist,omitempty"`
	// grpc方法请求黑名单，禁止调用黑名单里配置的rpc方法，一般和白名单配合使用，默认是空
	GrpcFuncBlacklist []string `protobuf:"bytes,8,rep,name=grpcFuncBlacklist" json:"grpcFuncBlacklist,omitempty"`
	// 是否开启https
	EnableTLS   bool `protobuf:"varint,10,opt,name=enableTLS" json:"enableTLS,omitempty"`
	EnableTrace bool `protobuf:"varint,10,opt,name=enableTrace" json:"enableTrace,omitempty"`
	// 证书文件，证书和私钥文件可以用cli工具生成
	CertFile string `protobuf:"varint,11,opt,name=certFile" json:"certFile,omitempty"`
	// 私钥文件
	KeyFile string `protobuf:"varint,12,opt,name=keyFile" json:"keyFile,omitempty"`
}

// Exec 配置
type Exec struct {
	// 执行器执行所需最小费用,低于Mempool和Wallet设置的MinFee,在minExecFee = 0 的情况下，isFree = true才会生效
	MinExecFee int64 `protobuf:"varint,1,opt,name=minExecFee" json:"minExecFee,omitempty"`
	// 执行器执行所需最大费用,这个值必须大于mempool 和 wallet 的值
	MaxExecFee int64 `protobuf:"varint,1,opt,name=maxExecFee" json:"maxExecFee,omitempty"`
	// 执行器执行是否免费
	IsFree bool `protobuf:"varint,2,opt,name=isFree" json:"isFree,omitempty"`
	// 是否开启stat插件
	EnableStat bool `protobuf:"varint,3,opt,name=enableStat" json:"enableStat,omitempty"`
	// 是否开启MVCC插件
	EnableMVCC       bool     `protobuf:"varint,4,opt,name=enableMVCC" json:"enableMVCC,omitempty"`
	DisableAddrIndex bool     `protobuf:"varint,7,opt,name=disableAddrIndex" json:"disableAddrIndex,omitempty"`
	Alias            []string `protobuf:"bytes,5,rep,name=alias" json:"alias,omitempty"`
	// 是否保存token交易信息
	SaveTokenTxList bool `protobuf:"varint,6,opt,name=saveTokenTxList" json:"saveTokenTxList,omitempty"`
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
