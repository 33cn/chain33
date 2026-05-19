# Chain33 — Working with Claude

## 项目本质
- 区块链开发框架,Go 1.20+,KISS 原则、高度模块化
- 19 个顶层模块按 7 层组织(见下方目录速查)
- 当前阶段焦点:测试覆盖率(目标稳住 65%+)、CI 稳定性、模块重构

## 关键命令(日常用 Makefile)
- `make`            全量构建(build + cli + depends)
- `make test`       单元测试
- `make testq`      静音版单元测试
- `make race`       race detector(改并发代码必跑)
- `make coverage`   生成覆盖率报告
- `make linter`     vet + ineffassign + gosec
- `make fmt`        gofmt + protobuf + shell 全套
- `make build_ci`   CI 专用,本地别用

## Git 工作流
chain33 在 Makefile 里包了一套自动化 git 命令,日常优先用:
- 新分支:`make branch b=<name>`
- 推送:  `make push b=<name> m="<msg>"`
- 同步 upstream:`make sync`
- 拉取别人的 PR:`make pull name=<user> b=<branch>`

也可以直接用 git 命令(实际开发中两种都有),但有几条硬规则:
- **不要直接改 master**,master 永远跟 upstream/master 同步
- 不要 `git push --force` 到共享分支(尤其 master)
- 不要 `--no-verify` 跳过钩子

## 环境约定
- 路径必须是 `$GOPATH/src/github.com/33cn/chain33`(go 包路径硬编码)
- 国内拉依赖前:`export GOPROXY=https://mirrors.aliyun.com/goproxy`
- 提交前跑 `make fmt && make linter`

## 目录速查(按层次)

### 1. 协议与数据
- `types/`        protobuf 自动生成的类型(交易、区块、地址等)。**不计入覆盖率**;改 .proto 后跑 `make depends`,不要手改 .pb.go

### 2. 区块链核心
- `blockchain/`   区块链主流程:区块同步、索引、最终性、状态
- `store/`        持久化存储抽象
- `mempool/`      交易池
- `account/`      账户、执行账户、创世账户

### 3. 共识与执行
- `consensus/`    共识接口(具体算法在 system/consensus/)
- `executor/`     交易执行引擎

### 4. 网络与通信
- `p2p/`          P2P 网络(含 gg18 TSS 子模块)
- `rpc/`          RPC:grpc + jsonrpc + ethrpc 三套
- `queue/`        进程内消息队列(模块间通信主干)
- `client/`       queue protocol 客户端

### 5. 应用层
- `wallet/`       钱包、BIP 助记词、种子管理
- `system/`       **可插拔系统扩展**:dapp / address / consensus / crypto / store / mempool 的接口与实现注册点 — 新算法/插件加在这里

### 6. 基础设施
- `common/`       通用工具:crypto / address / db / difficulty
- `util/`         辅助:bitmap / commit / healthcheck / cli helpers
- `pluginmgr/`    插件管理器
- `metrics/`      监控指标(influxdb)

### 7. 运行时与命令行
- `cmd/`          二进制入口:chain33(主进程)/ cli / autotest / execblock / dht_crawler

## 测试纪律
- 测试用 testnode 做集成,放 `*_integration_test.go`,build tag `integration`
- 单元测试可以 mock(参考 wallet/common 下的 mock 基础设施;queue 与 client 各自有 mocks/ 目录)
- 新代码必须有测试,protobuf 自动生成的不计入覆盖率

## 工作流(用 Claude 时)
- 非平凡改动走 brainstorm → spec(`docs/superpowers/specs/`)→ plan → TDD → verify
- 简单 typo / 单行修复直接做
- 完成前必须跑 `make test && make linter`,凭实际输出宣布完成,不靠"应该会过"

## 防坑清单
- 改 .proto 后必须跑 `make depends` 重新生成,不要手改 .pb.go
- 不要把 `coverage_*.out` / `build/coverage/` 这种本地产物提交(已在 .gitignore)
- 不要碰 `.gitlab-ci.yml` / `.travis.yml` / `appveyor.yml` / `.github/workflows/` 配置文件,改 CI 要确认再问
- 不要在没有 race detector 的情况下宣布并发代码"修好了" — 跑 `make race`
