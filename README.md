[![API Reference](
https://camo.githubusercontent.com/915b7be44ada53c290eb157634330494ebe3e30a/68747470733a2f2f676f646f632e6f72672f6769746875622e636f6d2f676f6c616e672f6764646f3f7374617475732e737667
)](https://godoc.org/github.com/33cn/chain33)
[![pipeline status](https://api.travis-ci.org/33cn/chain33.svg?branch=master)](https://travis-ci.org/33cn/chain33/)
[![Go Report Card](https://goreportcard.com/badge/github.com/33cn/chain33)](https://goreportcard.com/report/github.com/33cn/chain33)
 
# Chain33 Blockchain development framework

A highly modularized blockchain development framework according to the KISS principle 

Official website: https://chain.33.cn

Official plugin: https://github.com/33cn/plugin

The growth of chain33:[chain33诞生记](https://mp.weixin.qq.com/s/9g5ZFDKJi9uzR_NFxfeuAA)

## Building from source

Environment requirement: Go (version 1.9 or later)

Debug:

```shell
git clone https://github.com/33cn/chain33.git $GOPATH/src/github.com/33cn/chain33
cd $GOPATH/src/github.com/33cn/chain33
make
```

Testing：

```shell
$ make test
```

## Run:


Run single node with below command on your development environment.

```shell
$ chain33 -f chain33.toml
```

## Please aware of below when using chain33 plugins
1.Don't use master branch, please use publish branch.<br>
2.Don't re-create vendor dependency, we will support download vender folder for yourself in the future,
but currently not available.

## Contributions
Below is detailed contribution procedure. This can be skipped and directly see our 
simplified contribution flow in second part.

### 1. detailed procedure
*1.If you have any suggetions or bug, please create issues and discuss with us.

*2.Please fork `33cn/chain` to your own branch, like `vipwzw/chain33` via click up right `fork` button.
```
git clone https://github.com/vipwzw/chain33.git $GOPATH/src/github.com/33cn/chain33
```
Notice: Here you will need to clone to $GOPATH/src/github.com/33cn/chain33 or Go-lang package can't find the path.

*3.Add remote branch `33cn/chain33`: `git remote add upstream https://github.com/33cn/chain33.git. `
We have added `Makefile` to this and command `make addupstream` can be used.

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


