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
	2  选定主节点，leader从mempool中获取交易列表，排除重复后打包。将日志写到wal中，再通过raft复制到其余的从节点。
	3  leader,follower间设定心跳包来确认leader是否还存活，超时则触发leader切换。leader切换过程中共识
	   不会出错。
	4  节点增加，删除。共识不受影响
	5  配置文件中新增readOnlyPeers，增加只读节点，只读节点不参与共识，即当leader发生故障时，新的leader不会从只读节点中产生，
       而是在共识节点中产生。

运行方式（3个节点）:
	运行环境： ubuntu
	GO版本：	  go1.8.4
	步骤：
		//  1.编译版本
		cd $GOPATH/src/gitlab.33.cn/chain33
		git clone git@gitlab.33.cn:chain33/chain33.git
		cd chain33
		go build

		//	2.从管理机上拷贝可执行文件到各个节点
		scp chain33 ubuntu@172.31.8.229:/home/ubuntu/
		scp chain33 ubuntu@172.31.15.241:/home/ubuntu/
		scp chain33 ubuntu@172.31.4.182:/home/ubuntu/

		// 3.在各个节点上，依次修改chain33.toml
			name="raft"
			minerstart=true
			genesis="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
			nodeId=1
			raftApiPort=9121
			isNewJoinNode=false
			peersURL="http://127.0.0.1:9021"
			readOnlyPeersURL="http://172.31.2.210:9021,http://172.31.2.252:9021"
            addPeersURL=""
       // 4.然后依次启动种子节点，raft集群相关节点
       ./chain33 或者写个启动脚本

       // 5.动态增删节点
             动态加入节点
		    	1.）先在原有的集群随意一台机器（leader和follower都可以）上面执行
		    	    curl -L http://127.0.0.1:9121/4 -X POST -d http://172.31.4.185:9021
		    	    注意：前者为要加入的nodeId 值，后面http地址为要加入的peerUrl地址，将peerURL依次追加到addPeersURL中,
   	                      用逗号分隔,一次只能添加一个节点
		    	2.）然后在chain33.toml配置文件中，写好相关参数，启动chain33即可,
                    chain33.toml配置文件可依据前一个节点的配置
                    修改如下参数:
                     nodeId=x             //第几个节点
                    isNewJoinNode=true    //是否为新增节点
                    addPeersURL="xxxxx"   //新增节点的URL

			动态删除节点
		    	1.）在非删除节点执行如下curl命令即可
		    	    curl -L http://127.0.0.1:9121/4 -X  DELETE
		    	    注解，表示要删除第四个节点，删除操作为不可逆操作，一旦删除，就不会能重新添加进来.


	问题记录及解决：
		1. 多节点之间时间不同步导致的告警：rafthttp: the clock difference against peer 3 is too high [53.958607849s > 1s]
		需要同步ntp时间



*/
//接口设计：
