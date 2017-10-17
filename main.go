package main

//说明：
//main 函数会加载各个模块，组合成区块链程序
//主循环由消息队列驱动。
//消息队列本身可插拔，可以支持各种队列
//同时共识模式也是可以插拔的。
//rpc 服务也是可以插拔的

import (
	"queue"
	"rpc"
)

func main() {

	q := queue.New("channel") //topic=channal
	api := rpc.NewServer("jsonrpc", ":8801")
	api.SetQueue(q)

	go func(Q *queue.Queue) {
		//jsonrpc, grpc, channel 三种模式
		client := rpc.NewClient("channel", "")
		//同步接口
		client.SetQueue(q)

	}(q)
	q.Start()
}

