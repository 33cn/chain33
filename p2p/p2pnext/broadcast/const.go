package broadcast

//TTL
const (
	DefaultLtTxBroadCastTTL  = 3
	DefaultMaxTxBroadCastTTL = 25
)

// P2pCacheTxSize p2pcache size of transaction
const (
	//接收的交易哈希过滤缓存设为mempool最大接收交易量
	TxRecvFilterCacheNum = 10240
	BlockFilterCacheNum  = 50
	//发送过滤主要用于发送时冗余检测, 发送完即可以被删除, 维护较小缓存数
	TxSendFilterCacheNum  = 500
	BlockCacheNum         = 10
	MaxBlockCacheByteSize = 100 * 1024 * 1024
)
