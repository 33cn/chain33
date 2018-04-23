package consensus

//共识相关的模块
//模块功能：模块主要的功能是实现共识排序的功能，包括完整的共识的实现。
/**
1. solo
提供单节点排序功能，适用于开发过程，便于调试。
多节点情况下部署solo,只有主节点处理交易排序打包，其余节点不作处理。
主节点打包完之后，广播给其它节点。

2. raft
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
  然后chain33命令启动所有节点后，再向mempool写入交易

*/
//接口设计：
