# dapp ci test cases compose rule/ ci test cases 编写规则

## 框架介绍
说明：这里dapp关键字指的是各插件名字，比如relay,paracross
每个dapp的testcase和跟CI测试相关的文件比如Dockerfile等部署在自己目录下的cmd/build目录下
例如paracross在plugin/dapp/paracross/cmd/build下面

cmd目录下有两个文件Makefile和build.sh 负责在make时候把build里面的内容copy到系统的build/ci目录下的“dapp”目录准备测试
例如paracross copy到chain33/build/ci/paracross目录下

通过make docker-compose DAPP="dapp" 来运行测试 (此处dapp是各dapp的名字 比如paracross)
系统会依据build/ci/dapp目录生成一个临时的dapp-ci目录，把dapp和系统的bin和配置信息等一起copy进去来运行测试
测试执行完后相应dockers不会自动删除，还可以通过docker exec build_chain33_1 /root/chain33-cli [cmd...] 来查看信息
通过make docker-compose-down DAPP="dapp" 来删除此dapp的测试资源，释放docker

也可以通过dapp参数关键字all来run所有的dapp， all模式会自动删除pass的dapp的资源

 1. make docker-compose [PROJ=xx] [DAPP=xx]
    1. 如果PROJ 和DAPP都不设置如 make docker-compose, 只会run 系统的test case，不会run任何dapp
    1. 如果PROJ不设置，系统会缺省采用build关键字作为docker-compose的service工程名，如果设置以设置为准，
       不同PROJ可以实现docker compose 并行
    1. 如果DAPP不设置，则不run任何dapp，如果设置，则只run 指定的dapp，run结束后需要手动 make docker-compose-down DAPP=xx释放
    1. 如果DAPP=all 或者ALL， 则run 所有提供testcase的dapp
 1. make docker-compose down [PROJ=xx] [DAPP=xx] 
    负责clean make docker-compose 或make fork-test 创建了的docker资源， PROJ 和DAPP规则同上
 1. make fork-test [PROJ=xx] [DAPP=xx]   分叉测试
    1. 规则同make docker-compose     


## dapp/cmd/build下的文件信息
build下的文件都是和CI 测试相关的
 1. Dockerfile， 如果本dapp对系统的Dockerfile没有改动，可以不提供，使用系统默认的，如果有改动使用自己的，名字可以保持不变，系统不会覆盖，
    也可以对某个docker改变Dockerfile，使用自己命名的比如Dockerfile-xxx，需要设置到自己的docker-compose yml文件里面
    需要说明的是Dockerfile不能继承，只能替换
 1. docker-compose yml文件是组织docker service编排的，chain33至少需要启动2个docker才能挖矿，如果对Dockerfile 命名有修改，可以在
    这里指定，docker-compose文件可以和系统文件继承， 也就是dapp里面可以只写修改的部分，如和系统有重叠，以dapp里面的为准。
    docker-compose yml文件可以对docker service做各种定制
    如果对docker-compose没有修改，也可以不提供
    如果和系统docker-compose.yml文件组合使用，dapp的compose文件必须符合docker-compose-$dapp.yml的命名    
 1. testcase.sh 测试用例写在这个文件里，规定一定是testcase.sh这个文件名，不然系统找不到。
    当前对testcase提供三个step：
    1. init 是docker 启动前对配置文件需要的修改
    1. config： docker启动后对dapp做需要的配置，比如转账，设置钱包等
    1. test： ci测试用例部分
    testcase一定要提供一个以自己dapp名字命名的function作为入口函数，三个step不是必须的
    ```
     function paracross() {
         if [ "${2}" == "init" ]; then
             para_init
         elif [ "${2}" == "config" ]; then
             para_transfer
             para_set_wallet
         elif [ "${2}" == "test" ]; then
             para_test 
         fi
     
     }
     ```    
 1. fork-test.sh 分叉测试用例，规定一定是这个文件名，不然系统找不到
    fork-test dapp函数入口可以和testcase共用，通过source 把testcase.sh import进来
    fork test 也提供了5个step供测试配置
    1. forkInit: docker启动前对文件参数的修改
    1. forkConfig: docker service启动后做系统的配置，如转账
    1. forkAGroupRun: 分叉测试分成两个测试group A和B 交替挖矿来模拟实际分叉场景，这里是Agroup先运行时候自己用例的配置
    1. forkBGroupRun: B group运行时候dapp的设置
    1. forkChekRst: 分叉测试结束后查看系统回滚后的结果
    ```
     function privacy() {
         if [ "${2}" == "forkInit" ]; then
             privacy_init
         elif [ "${2}" == "forkConfig" ]; then
             initPriAccount
         elif [ "${2}" == "forkAGroupRun" ]; then
             genFirstChainPritx
             genFirstChainPritxType4
         elif [ "${2}" == "forkBGroupRun" ]; then
             genSecondChainPritx
             genSecondChainPritxType4
         elif [ "${2}" == "forkCheckRst" ]; then
             checkPriResult
         fi
    
     }
     ```
 


