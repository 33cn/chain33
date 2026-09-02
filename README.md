[![Go Reference](https://pkg.go.dev/badge/github.com/33cn/chain33.svg)](https://pkg.go.dev/github.com/33cn/chain33)
[![pipeline status](https://github.com/33cn/chain33/actions/workflows/build.yml/badge.svg)](https://github.com/33cn/chain33/actions/)
[![Go Report Card](https://goreportcard.com/badge/github.com/33cn/chain33)](https://goreportcard.com/report/github.com/33cn/chain33)
[![Windows Build Status](https://ci.appveyor.com/api/projects/status/github/33cn/chain33?svg=true&branch=master&passingText=Windows%20-%20OK&failingText=Windows%20-%20failed&pendingText=Windows%20-%20pending)](https://ci.appveyor.com/project/33cn/chain33)
[![Release](https://img.shields.io/github/v/release/33cn/chain33)](https://github.com/33cn/chain33/releases)
[![code coverage](https://img.shields.io/badge/coverage-65.5%25-yellowgreen)](https://github.com/33cn/chain33/actions/workflows/build.yml) *(不含 protobuf 自动代码)*

# Chain33 区块链开发框架

高度模块化, 遵循 KISS原则的区块链开发框架

官方网站和文档: https://chain.33.cn

官方插件库: https://github.com/33cn/plugin

典型案例: https://github.com/bityuan/bityuan

chain33背后故事: [chain33诞生记](https://mp.weixin.qq.com/s/9g5ZFDKJi9uzR_NFxfeuAA)

视频教程: [视频教程](https://chain.33.cn/document/289)

# 感谢

[腾讯玄武安全实验室](https://github.com/33cn/chain33/issues?utf8=%E2%9C%93&q=label%3A%E8%85%BE%E8%AE%AF%E7%8E%84%E6%AD%A6%E5%AE%9E%E9%AA%8C%E5%AE%A4)

## 模块架构

Chain33 采用分层模块化设计, 核心模块按层次划分:

| 层次 | 模块 | 说明 |
|------|------|------|
| 协议与数据 | `types/` | protobuf 自动生成的类型定义(交易、区块、地址等) |
| 区块链核心 | `blockchain/` `store/` `mempool/` `account/` | 区块同步与索引、持久化存储、交易池、账户与执行账户 |
| 共识与执行 | `consensus/` `executor/` | 共识接口与交易执行引擎(具体共识算法见 `system/consensus/`) |
| 网络与通信 | `p2p/` `rpc/` `queue/` `client/` | P2P 网络、grpc/jsonrpc/ethrpc 三套 RPC、进程内消息队列 |
| 应用层 | `wallet/` `system/` | 钱包与 BIP 助记词、可插拔系统扩展(dapp/address/crypto/store 等接口与注册点) |
| 基础设施 | `common/` `util/` `pluginmgr/` `metrics/` | 通用工具库、辅助组件、插件管理器、监控指标 |
| 运行时与命令行 | `cmd/` | 二进制入口: chain33 主进程、cli、autotest、execblock 等 |

## Building from source

环境要求: Go 1.21+

编译:

```shell
git clone https://github.com/33cn/chain33.git $GOPATH/src/github.com/33cn/chain33
cd $GOPATH/src/github.com/33cn/chain33
make
```

```
注意: 代码必须放在 $GOPATH/src/github.com/33cn/chain33, 否则 go 包路径会找不到。
国内用户需要配置代理获取依赖包, mod 功能已在 Makefile 中默认开启:
export GOPROXY=https://mirrors.aliyun.com/goproxy
```

测试:

```shell
$ make test
```

## 运行

通过这个命令可以运行一个单节点到环境, 可以用于开发测试

```shell
$ chain33 -f chain33.toml
```

## 使用chain33 开发插件注意点

* 不可以使用 master 分支, 要使用 发布分支

## 贡献代码

我们先说一下代码贡献的细节流程, 这些流程可以不看, 用户可以直接看我们贡献代码简化流程

### 细节过程

* 如果有什么想法, 建立 issues, 和我们来讨论。
* 首先点击 右上角的 fork 图标, 把chain33 fork 到自己的分支 比如我的是 vipwzw/chain33
* `git clone https://github.com/vipwzw/chain33.git $GOPATH/src/github.com/33cn/chain33`

```
注意: 这里要 clone 到 $GOPATH/src/github.com/33cn/chain33, 否则go 包路径会找不到
```

* 添加 `33cn/chain33` 远端分支: `git remote add upstream https://github.com/33cn/chain33.git`  我已经把这个加入了 Makefile 可以直接 运行 `make addupstream`

* 保持 `33cn/chain33` 和 `vipwzw/chain33` master 分支的同步, 可以直接跑 `make sync` , 或者执行下面的命令

```
git fetch upstream
git checkout master
git merge upstream/master
```
```
注意: 不要去修改 master 分支, 这样, master 分支永远和upstream/master 保持同步
```

* 从最新的33cn/chain33代码建立分支开始开发

```
git fetch upstream
git checkout master
git merge upstream/master
git checkout -b "fixbug_ci"
```

* 开发完成后, push 到 `vipwzw/chain33`

```
git fetch upstream
git checkout master
git merge upstream/master
git checkout fixbug_ci
git merge master
git push origin fixbug_ci
```

然后在界面上进行pull request

### 简化流程

#### 准备阶段

* 首先点击 右上角的 fork 图标, 把chain33 fork 到自己的分支 比如我的是 vipwzw/chain33
* `git clone https://github.com/vipwzw/chain33.git $GOPATH/src/github.com/33cn/chain33`

```
注意: 这里要 clone 到 $GOPATH/src/github.com/33cn/chain33, 否则go 包路径会找不到
```

```
make addupstream
```

#### 开始开发: 这个分支名称自己设置

```
make branch b=mydevbranchname
```

#### 开发完成: push

```
make push b=mydevbranchname m="这个提交的信息"
```

如果m不设置, 那么不会执行 git commit 的命令

## 修改别人的pull request

比如我要修改 name=libangzhu branch chain33-p2p-listenPort 的pr

##### step1: 拉取要修改的分支

```
make pull name=libangzhu b=chain33-p2p-listenPort
```

然后修改代码, 修改完成后, 并且在本地commit

###### step2: push已经修改好的内容

```
make pullpush name=libangzhu b=chain33-p2p-listenPort
```

## License

```
BSD 3-Clause License

Copyright (c) 2018, 33.cn
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of the copyright holder nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```
