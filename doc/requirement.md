**【需求描述】**

# 1. 功能性需求：

## [一期]

### 角色： 普通用户，管理员

#### 普通用户的功能：

1.  注册一个账户
2.  申请开通写权限(授权写多少数据：比如1e8 Bytes)，账户里面有个计数器，可以写多少数据。
3.  通过rpc接口写一笔交易，扣除相应计数器的数目
4.  提供一个交易hash，可以得到一个merkle proof

#### 管理员

1.  给某个用户授权开通写权限与写数据的量
2.  禁用某个用户的写权限

# 2. 非功能性需求

## 性能需求
每个模块要求 5w tps  
整体要求3w tps

## 规范性要求：
*   编写语言：golang
*   golang编程：doc/golang
*   [代码规范](http://docscn.studygolang.com/doc/effective_go.html)
*   [代码review](https://github.com/golang/go/wiki/CodeReviewComments#error-strings)
*   工具：
    >> 格式化工具：所有的代码提交的过程中，需要执行：  
    >> go fmt
    >> 并且需要执行检查工具：  
    >> golint  
    >> go vet  
    >> 修复其中的 warning 错误
    >> go 包管理工具(govendor)  
    >> go的扩展包放在 vendor 中，引入包需要经过批准才能引入。更新包也需要批准。

* rpc:
    >> 所有的rpc 采用grpc，编码采用protocol buffer

* 质量要求：
    >> 模块必须有设计文档 和 接口文档
    >> 单元测试覆盖率 到达到 80% 以上
    >> 提交的代码必须通过主管代码审核
dummy push