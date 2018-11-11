// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

3. pbft:

   目前实现功能：
   可以从mempool中读取交易，进行多节点pbft共识，共识完成后执行块。nodeid是1的节点默认是primary节点，其余节点不执行打包只参与共识。
   节点之间通信方式为tcp。目前暂不支持增加、删除节点，需要先将所有节点都写在配置文件中。
   //TODO ：1、链的可持久化，每128个块增加检查点，清理之前的log，防止过于臃肿
            2、增加视图变更，即在不影响共识过程的情况下变更primary节点

   pbft文件夹中各个模块的主要功能：
   pbft.go：定义pbft节点的结构，整个共识过程需要用到的函数
   block.go: 让pbftclient继承baseclient，执行打包等任务然后通过chan传递给节点进行共识
   controllor.go: 通过config文件生成新的pbft节点
   pbft_test.go：单元测试文件

   如何使用：
   单节点测试：直接运行pbft_test即可，会写50个区块进行测试
   多节点测试：
   windows和linux均可
   develop分支请先用protoc重新生成pb.go文件
   首先下载pbftbranch，然后创建code.aliyun.com/chain33文件夹，将解压出来的chain33文件夹放入下面，进入chain33文件夹
   执行go build命令生成可执行文件
   因为我是在一台电脑上进行测试，因此只需要在本机上拷贝文件即可，若需要在不同的电脑上进行测试，拷贝到不同电脑上即可
   修改个节点的chain33.toml即可，主要修改方式如下：
   name = "pbft"
   genesis = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
   minerstart = true
   genesisBlockTime = 1514533394
   hotkeyAddr = "12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
   NodeId = 1  //当前节点的id，每个节点需要不同，1是默认primary节点
   PeersURL = "127.0.0.1:8890,127.0.0.1:8891"  //所有节点的ip地址，用逗号隔开，我是一台电脑所以用的本地地址只是port不同
   ClientAddr = "127.0.0.1:8890"  //当前节点的ip地址
   //TODO 动态增加或者删除节点
  然后chain33命令启动所有节点后，再向mempool写入交易validator

4.tendermint:
实现功能：
	1  建立tendermint集群，如果要实现拜占庭容忍至少1个节点的能力，集群至少包含4个validator节点，建立成功后集群中的proposer将轮流从validator中选择，每次选中的proposer将成为提议者和写入区块者，提议(proposal)包含从mempool中取到的交易信息，所有validator节点参与对提议(proposal)投票。
	2  投票分为两个阶段——prevote和precommit，首先每个validator对proposal进行第一阶段投票，当每个validator收到的投票结果比例大于2/3节点个数时，每个validator根据这个结果进行第二阶段投票，当每个validator根据收到的投票结果为nil的比例大于2/3节点个数时，将开始新一轮的提议，投票，
       当收到的投票结果相同且不为nil的比例大于2/3节点个数时，进入commit阶段。该轮的proposer写区块，非proposer等待接收广播的新区块。之后进入新高度的共识。
	3  整个共识过程中如果小于1/3的validator出现一些异常，不影响剩下validator继续共识。
	4  动态增加，删除validator节点，修改validator节点权限，共识不受影响
	5  仅支持创建集群时静态配置validator，不支持通过停止所有validator，修改genesis.json配置文件来增加，删除validator节点，修改validator节点权限的操作。

运行方式（4个节点）:
	运行环境： ubuntu
	GO版本：	  go1.9.2
	步骤：
		//  1.编译版本
		cd $GOPATH/src/gitlab.33.cn/chain33
		git clone git@gitlab.33.cn:chain33/chain33.git
		cd chain33
		go build

		//	2.从管理机上拷贝可执行文件到各个节点
		scp chain33 ubuntu@192.168.0.117:/home/ubuntu/
		scp chain33 ubuntu@192.168.0.119:/home/ubuntu/
		scp chain33 ubuntu@192.168.0.120:/home/ubuntu/
		scp chain33 ubuntu@192.168.0.121:/home/ubuntu/

		// 3.在各个节点上，依次修改chain33.toml中[consensus]项
			name="tendermint"
			minerstart=false
			genesis="14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
			genesisBlockTime=1514533394
			hotkeyAddr="12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
			timeoutPropose=5000
			timeoutProposeDelta=500
			timeoutPrevote=3000
			timeoutPrevoteDelta=500
			timeoutPrecommit=3000
			timeoutPrecommitDelta=500
			timeoutCommit=3000
			skipTimeoutCommit=false
			createEmptyBlocks=false
			createEmptyBlocksInterval=0
			seeds=["192.168.0.117:46656","192.168.0.119:46656","192.168.0.120:46656","192.168.0.121:46656"]
       // 4.然后依次启动种子节点，tendermint集群相关节点
       ./chain33 或者写个启动脚本
       // 5.动态增删节点
             动态加入节点
		    	1.）将新生成的priv_validator.json文件和其他Validator节点的genesis.json和chain33.toml拷贝到待加入集群的节点工作目录下，然后启动chain33。
					注意：可以修改chain33.toml文件的[p2p]项和[consensus]项中的seeds参数，只要是集群节点就可以。
		    	2.）将新节点的priv_validator.json文件中的pub_key记录下来，向集群任一节点发送交易，执行器为valnode，构建payload为ValNodeAction结构体的序列化，
					该结构体参数Value为ValNodeAction_Node结构体地址，Ty为1，ValNodeAction_Node结构体有两个参数，PubKey为刚才记录下的字符串通过hex.DecodeString()编码生成的[]byte，
					Power最大不能超过总power值的1/3。该笔交易执行成功即添加完成，新节点将会参与新区块的共识。
					注意：新增节点的Power值一定不能超过总power值的1/3，否则就是该笔交易执行成功了，该节点也不能成功加入。
			动态删除节点
		    	1.）同加入节点一样，向任意节点发送一笔交易，交易格式与加入节点一样，其中pub_key为要删除的节点的pub_key，Power设置为0，该节点将不会参与新区块的共识。
		    动态修改validator的Power值
				1.）同加入节点一样，向任意节点发送一笔交易，交易格式与加入节点一样，其中pub_key为要修改的节点的pub_key,Power为要设置的值，
                    注意：修改的Power值一定不能超过总power值的1/3，否则就是该笔交易执行成功了，该节点的Power值也不会更改。
*/
//接口设计：
