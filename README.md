[![API Reference](
https://camo.githubusercontent.com/915b7be44ada53c290eb157634330494ebe3e30a/68747470733a2f2f676f646f632e6f72672f6769746875622e636f6d2f676f6c616e672f6764646f3f7374617475732e737667
)](https://godoc.org/github.com/33cn/chain33)
[![pipeline status](https://api.travis-ci.org/33cn/chain33.svg?branch=master)](https://travis-ci.org/33cn/chain33/)
[![Go Report Card](https://goreportcard.com/badge/github.com/33cn/chain33)](https://goreportcard.com/report/github.com/33cn/chain33)
 [![Windows Build Status](https://ci.appveyor.com/api/projects/status/github/33cn/chain33?svg=true&branch=master&passingText=Windows%20-%20OK&failingText=Windows%20-%20failed&pendingText=Windows%20-%20pending)](https://ci.appveyor.com/project/33cn/chain33)
[![codecov](https://codecov.io/gh/33cn/chain33/branch/master/graph/badge.svg)](https://codecov.io/gh/33cn/chain33) [![Join the chat at https://gitter.im/33cn/Lobby](https://badges.gitter.im/33cn/Lobby.svg)](https://gitter.im/33cn/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

# Chain33 区块链开发框架

高度模块化, 遵循 KISS原则的区块链开发框架

官方网站 和 文档: https://chain.33.cn

官方插件库: https://github.com/33cn/plugin

典型案例: https://github.com/bityuan/bityuan

## Building from source

环境要求: Go (version 1.9 or later)

编译:

```shell
git clone https://github.com/33cn/chain33.git $GOPATH/src/github.com/33cn/chain33
cd $GOPATH/src/github.com/33cn/chain33
make
```

测试：

```shell
$ make test
```

## 运行

通过这个命令可以运行一个单节点到环境，可以用于开发测试

```shell
$ chain33 -f chain33.toml
```

## 贡献代码

我们先说一下代码贡献的细节流程，这些流程可以不看，用户可以直接看我们贡献代码简化流程

### 细节过程

* 如果有什么想法，建立 issues, 和我们来讨论。
* 首先点击 右上角的 fork 图标， 把chain33 fork 到自己的分支 比如我的是 vipwzw/chain33
* `git clone https://github.com/vipwzw/chain33.git $GOPATH/src/github.com/33cn/chain33`

```
注意：这里要 clone 到 $GOPATH/src/github.com/33cn/chain33, 否则go 包路径会找不到
```

* 添加 `33cn/chain33` 远端分支： `git remote add upstream https://github.com/33cn/chain33.git`  我已经把这个加入了 Makefile 可以直接 运行 `make addupstream` 

* 保持 `33cn/chain33` 和 `vipwzw/chain33` master 分支的同步，可以直接跑 `make sync` , 或者执行下面的命令

```
git fetch upstream
git checkout master
git merge upstream/master
```
```
注意：不要去修改 master 分支，这样，master 分支永远和upstream/master 保持同步
```

* 从最新的33cn/chain33代码建立分支开始开发

```
git fetch upstream
git checkout master
git merge upstream/master
git branch -a "fixbug_ci"
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

* 首先点击 右上角的 fork 图标， 把chain33 fork 到自己的分支 比如我的是 vipwzw/chain33
* `git clone https://github.com/vipwzw/chain33.git $GOPATH/src/github.com/33cn/chain33`

```
注意：这里要 clone 到 $GOPATH/src/github.com/33cn/chain33, 否则go 包路径会找不到
```

```
make addupstream
```

#### 开始开发： 这个分支名称自己设置

```
make branch b=mydevbranchname
```

#### 开发完成: push 

```
make push b=mydevbranchname m="这个提交的信息"
```

如果m不设置，那么不会执行 git commit 的命令

#### 修改别人提交的pull request

```
make pull name=fork_user_name b=pull_requst_branch
```

在本地创建分支后，开始修改别人的代码，最后提交一下代码就可以

```
make pushpull name=fork_user_name b=pull_requst_branch
```

在pull request 到页面上就可以看到你的修改

## License

```
BSD 2-Clause License

Copyright (c) 2018, 33.cn
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

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
