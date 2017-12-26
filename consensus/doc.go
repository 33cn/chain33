package consensus

//共识相关的模块
//模块功能：模块主要的功能是实现共识排序的功能，包括完整的共识的实现。
/**
1. solo
提供单节点排序功能，适用于开发过程，便于调试。
多节点情况下部署solo,只有主节点处理交易排序打包，其余节点不作处理。
主节点打包完之后，广播给其它节点。

2. raft
实现功能：
	1  建立raft集群，建立成功后集群中只有一个leader节点，多个follower节点。leader节点和follower节点
	之间通过http方式通信。
	2  选定主节点，validator从mempool中获取交易列表，排除重复后打包。 再通过raft复制到其余的从节点。
	确认可以commit后，把block写入blockchain.
	3   leader,follower间设定心跳包来确认leader是否还存活，超时则触发leader切换。leader切换过程中共识
	不会出错。  -- 验证开发中
	4   节点增加，删除。共识不受影响  -- 验证开发中

运行方式（3个节点）:
	运行环境： ubuntu
	GO版本：	  go1.8.4
	步骤：
		//  编译版本
		cd $GOPATH/src/code.aliyun.com/chain33
		git clone git@code.aliyun.com:chain33/chain33.git
		cd chain33
		go build

		//	从管理机上拷贝可执行文件到各个节点
		scp chain33 ubuntu@172.31.8.229:/home/ubuntu/
		scp chain33 ubuntu@172.31.15.241:/home/ubuntu/
		scp chain33 ubuntu@172.31.4.182:/home/ubuntu/

		// 在各个节点上执行
		./chain33 --id 1 --cluster http://172.31.8.229:12379,http://172.31.15.241:12379,http://172.31.4.182:12379 --validator
		./chain33 --id 2 --cluster http://172.31.8.229:12379,http://172.31.15.241:12379,http://172.31.4.182:12379
		./chain33 --id 3 --cluster http://172.31.8.229:12379,http://172.31.15.241:12379,http://172.31.4.182:12379

	问题记录及解决：
		1. 多节点之间时间不同步导致的告警：rafthttp: the clock difference against peer 3 is too high [53.958607849s > 1s]
		需要同步ntp时间


3. bft

*/
//接口设计：
