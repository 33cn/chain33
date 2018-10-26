package version

const version = "5.3.0"

var (
	WalletVerKey     = []byte("WalletVerKey")
	BlockChainVerKey = []byte("BlockChainVerKey")
	LocalDBMeta      = []byte("LocalDBMeta")
	MavlTreeVerKey   = []byte("MavlTreeVerKey")
	GitCommit        string
)

func GetLocalDBKeyList() [][]byte {
	return [][]byte{
		WalletVerKey, BlockChainVerKey, LocalDBMeta, MavlTreeVerKey,
	}
}

func GetVersion() string {
	if GitCommit != "" {
		return version + "-" + GitCommit
	}
	return version
}

//数据库版本解析
/*
格式: v1.v2.v3
如果: v1 升级了， 那么意味着localdb 需要 重新 reindex
*/
func GetLocalDBVersion() string {
	return "0.0.0"
}

//v0.1.2
//更新内容：
// 1.p2p 修改为在nat结束后，在启动peer的stream，ping,version 等功能

//v0.1.3
//硬分叉

//v0.1.4
//更新内容
// p2p  增加节点下载速度监控功能，改进下载模块
// p2p  增加p2p serverStart 功能，及时自身节点在外网或者可以穿透网络，也不会对外提供服务，但不影响挖矿，数据同步

//数据库版本相关接口
// blcokchain db
//	ver=1:增加地址参与交易量的记录，
// wallet db:
//	ver=1:增加rescan的功能，自动将wallet账户相关的tx交易信息重新扫描从blockchian模块
// state mavltree db

//v5.3.0
//hard fork for bug
