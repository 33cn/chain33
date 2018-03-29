**【需求描述】**

# 1. 功能性需求：

## [一期]

#### 完成区块链框架：

1. 框架可以用于所有种类的区块链
2. 区块链的基本功能，框架都要提供：P2P blockchain 共识 mempool 存储 等
3. 在这个框架上开发比特元

# 2. 非功能性需求

## 性能需求
每个模块要求 5w tps （没有merkle tree的情况下） 
整体要求3w tps （没有merkle tree 的情况下）

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