package common

const version = "v0.1.4"

func GetVersion() string {
	return version
}

//v0.1.2
//更新内容：
// 1.p2p 修改为在nat结束后，在启动peer的stream，ping,version 等功能

//v0.1.3
//硬分叉

//v0.1.4
//更新内容
// p2p  增加节点下载速度监控功能
