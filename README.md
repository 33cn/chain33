[![pipeline status](https://api.travis-ci.org/33cn/chain33.svg?branch=master)](https://travis-ci.org/33cn/chain33/)
[![Go Report Card](https://goreportcard.com/badge/github.com/33cn/chain33)](https://goreportcard.com/report/github.com/33cn/chain33)
 
# Chain33 区块链开发框架

高度模块化, 遵循 KISS原则的区块链开发框架

官方网站 和 文档: https://chain.33.cn

官方插件库: https://github.com/33cn/plugin


## Building from source

环境要求: Go (version 1.9 or later)

编译:

```shell
go get -v -u github.com/33cn/chain33
cd $GOPATH/github.com/33cn/chain33
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
