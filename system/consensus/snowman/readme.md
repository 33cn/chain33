
## 区块最终性共识

基于avlanche snowman共识引擎实现

### 使用配置
涉及blockchain, consensus 模块主要配置项

```toml

[blockchain]
# 启动高度, 作为初始共识高度
blockFinalizeEnableHeight=0
# 区块最终化启动后等待高度差, 达到启动高度后继续等待高度值, 最低1, 默认128
blockFinalizeGapHeight=128


[consensus]
# 插件名称, 不配置则关闭最终化共识
finalizer="snowman"

# 共识参数, 通常无需配置, 采用默认即可
[consensus.sub.snowman]
# 单次投票参与节点数
k=20
# 单次投票通过最低票数, 需大于k/2
alpha=15 
# 达成共识所需的连续投票成功次数
betaVirtuous=15

```


### 底层实现

- 基于avalanchego v1.10.9, 并针对chain33做了简单改动适配, 主要包括整合区块哈希, 节点ID等数据格式保持一致, 移除跨平台编译不兼容依赖, 日志调整
- replace github.com/ava-labs/avalanchego => github.com/bysomeone/avalanchego

#### 模块简介

> blockchain/finalize.go
> 
snowman共识结果信息维护, 并根据本地主链打包情况进行状态校验, 分叉重置等处理

> consensus/snowman
> 
snowman共识引擎适配实现, 以下为主要文件介绍

- snowman.go 共识入口, chain33 consensus finalizer插件实现, 事件接收分发调度
- vm.go 链相关的功能适配, 衔接底层共识和blockchain交互, 涉及区块打包, 查询, 共识
- validator.go 验证者功能适配, 主要实现节点抽样
- sender.go 网络发送功能适配, 衔接底层p2p通信, 涉及共识交互
- config.go 配置, 日志等底层接口依赖适配

> dht/snow

底层p2p通信交互实现

- EventHandler 处理本地消息事件
- StreamHandler 处理网络消息事件
