# Solo共识


## 性能测试
> solo_test文件提供了常用的性能测试

### 标准参数
>go test参数

- run, 指定单元测试名称, 如果名称不存在即屏蔽所有单元测试, 如-run=null
- bench, 指定基准参数名称, 如果名称不存在即屏蔽所有基准测试, 如-bench=none
- benchtime, 指定基准测试时长, 默认基准测试只跑1s
- memprofile, 指定go pprof 内存分析文件, 默认不输出该文件
- cpuprofile, 指定go pprof cpu分析文件, 默认不输出该文件

### BenchmarkSolo
> 基于solo共识,测试单节点区块打包性能, 支持存证/转账交易

#### 参数
> 自定义参数, 输入不同条件测试单节点打包性能

- port, 监听grpc端口, 默认9902
- maxtx, 性能测试时区块内交易数, 默认1w笔
- sign, 是否开启交易签名验签, 默认false不开启
- fee, 是否开启收取交易费, 默认false
- txindex, 是否开启交易哈希查询数据库索引, 默认false
- txtype, 设置性能测试交易类型, 支持存证(none)/转账(coins), 默认值为none
- txsize, 设置存证交易的大小, 默认32字节
- accnum, 设置转账交易参数的账户数, 默认10 


#### 示例
```bash 
# 测试节点在10秒时长打包的区块数, 区块内交易为1w笔, 交易类型为存证
go test -run=null -bench=Solo -benchtime=10s

# 指定存证交易大小
go test -run=null -bench=Solo -benchtime=10s -txsize=200

# 开启交易验签
go test -run=null -bench=Solo -benchtime=10s -sign

# 开启交易费
go test -run=null -bench=Solo -benchtime=10s -fee

# 指定区块交易数为2w
go test -run=null -bench=Solo -benchtime=10s -maxtx=20000 

# 开启交易查询索引
go test -run=null -bench=Solo -benchtime=10s -txindex

# 测试转账交易
go test -run=null -bench=Solo -benchtime=10s -txtype=coins

# 指定转账交易账户数
go test -run=null -bench=Solo -benchtime=10s -txtype=coins -accnum=1000

```



### BenchmarkCheckSign
>测试机器验签性能

#### 示例
```bash
go test -run=null -bench=CheckSign
```

## 测试工具发布

> 将当前目录下单元测试编译为二进制文件, 方便在不同环境下进行测试

### 编译

```bash

#当前目录下, 指定输出可执行文件 solo.test
go test -c -o solo.test

``` 

### 执行
> 执行命令和go test类似, 标准参数需要增加test.前缀, 如-test.benchtime等, 自定义参数不需要

```bash

#Solo性能测试
./solo.test  -test.run=none -test.bench=Solo -test.benchtime=10s -txtype=coins -sign

```
  
