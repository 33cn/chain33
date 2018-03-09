package main

/*
QLANG使用说明

1. 下载qlang
git clone git@github.com:qiniu/qlang.git

2. 编译安装
在qlang.io/cmd/qlang路径下
go build
go install
安装以后，可以使用qlang直接进入命令行模式，也可以编写X.ql文件脚本，用qlang X.ql执行脚本

3. 利用qlang脚本测试黑名单功能
a. qlang.go监听HTTP本地地址8081端口，接收到消息以后，找出关键KEY（采用不完整的分析，第一个string数据需要固定描述）。注：不依赖数据结构，除了关键KEY，其他数据不处理
b. qlang.go中通过RPC接口与链进行通信，通过NORM执行器进行读写操作
c. blacklist.ql 尝试写数据
d. get.ql尝试读数据,这两个脚本可以放在linux虚拟机上运行的

*/
