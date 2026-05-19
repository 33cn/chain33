# Chain33 Claude 协作基础建设 — 设计文档

- **日期**:2026-05-19
- **作者**:king(与 Claude 共同 brainstorm)
- **状态**:待 review
- **范围**:为 chain33 仓库一次性建立使用 Claude(superpowers skill 系统)的基础设施

---

## 1. 目标与背景

chain33 是一个成熟的 Go 区块链开发框架(357 个 Go 文件、130 个测试文件、19 个顶层模块),最近 3 个月的团队焦点集中在测试覆盖率、CI 稳定性与模块重构。当前仓库**没有任何 Claude 相关配置**,导致:

- Claude 不了解 chain33 特有的协作约定(`make branch` / `make push` 包装、`$GOPATH` 硬编码路径、protobuf 不计覆盖率)
- 每次跑 `make test` / `go test` / `gofmt` 都要重复授权
- 没有标准位置存放设计文档与实施计划,大型改动的上下文容易丢

**本次目标**:用 superpowers 推荐的"标准型"配置,为 chain33 一次性建立 Claude 协作基础建设。

## 2. 范围

### 2.1 在范围内

1. 编写 `CLAUDE.md`(项目根目录),固化 chain33 的命令、约定、目录结构、防坑规则
2. 建立 `.claude/settings.json`(团队共享),把频繁、安全、可逆的命令加入允许列表
3. 调整 `.gitignore`,允许 `settings.json` 提交、保持其他个人配置忽略
4. 建立 `docs/superpowers/` 目录骨架(README + `specs/` + `plans/`),给后续工作流产出预留位置

### 2.2 非目标(本次不做)

- **不写 hooks**:hooks 是反应式自动化,设计错了反而打断工作流;等 1-2 个月暴露真实痛点再加
- **不写自定义 agents**(`.claude/agents/`):没有具体痛点之前写专属 agent 容易是空中楼阁
- **不写 `CONTRIBUTING-WITH-CLAUDE.md`**:多人协作指南容易跑偏成"我猜大家想要的样子",等 spec 流程真正跑过几次再总结
- **不改 Makefile**:`make branch` / `make push` 等自动化命令保持原样
- **不改 README.md**:README 是人类文档,与 CLAUDE.md 平行存在

## 3. 决策记录

brainstorm 阶段达成的关键决策(供未来回溯):

| 议题 | 决策 | 备注 |
|------|------|------|
| 规模选型 | 标准型(B) | 最小型(A)不够、完整脚手架(C)过早 |
| Git 工作流口径 | `make` 系列与原生 `git` 并存 | 实际开发中两种都用 |
| 目录速查呈现方式 | 按 7 层组织 19 个模块 | 用户明确要求层次化 |
| 单元测试 mock 政策 | 允许 mock | 已有 wallet/common、queue/mocks、client/mocks 等基础设施 |
| `make depends` 是否进允许列表 | 进 | protobuf 工作流的常规步骤,虽改源码但可控 |
| `.claude/settings.json` 是否提交 | 提交 | 团队共享配置统一,个人偏好放 `settings.local.json` |
| `make push` / `make sync` 是否进允许列表 | 不进 | 涉及远端,每次提示 |
| 通用 `make:*` 是否进允许列表 | 不进 | 太宽,会覆盖 push/sync/docker |

## 4. 详细设计

### 4.1 `CLAUDE.md`(项目根目录,新建)

完整内容如下:

````markdown
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
````

### 4.2 `.gitignore` 改动

第 31 行 `.claude*` 改为:

```diff
- .claude*
+ .claude/*
+ !.claude/settings.json
```

效果:
- `.claude/` 目录里的内容默认忽略
- `.claude/settings.json`(团队共享配置)允许提交
- `.claude/settings.local.json`(个人偏好、token、本地路径)依然忽略

### 4.3 `.claude/settings.json`(新建)

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "permissions": {
    "allow": [
      "Bash(make test:*)",
      "Bash(make testq:*)",
      "Bash(make race:*)",
      "Bash(make coverage:*)",
      "Bash(make coverhtml:*)",
      "Bash(make linter:*)",
      "Bash(make linter_test:*)",
      "Bash(make vet:*)",
      "Bash(make gosec:*)",
      "Bash(make ineffassign:*)",
      "Bash(make bench:*)",
      "Bash(make fmt:*)",
      "Bash(make depends:*)",
      "Bash(make build:*)",
      "Bash(make cli:*)",

      "Bash(go test:*)",
      "Bash(go vet:*)",
      "Bash(go build:*)",
      "Bash(go fmt:*)",
      "Bash(go list:*)",
      "Bash(go doc:*)",
      "Bash(go mod download)",
      "Bash(go mod verify)",
      "Bash(go mod tidy)",
      "Bash(gofmt:*)",

      "Bash(git status:*)",
      "Bash(git diff:*)",
      "Bash(git log:*)",
      "Bash(git show:*)",
      "Bash(git blame:*)",
      "Bash(git branch)",
      "Bash(git remote -v)",
      "Bash(git stash list)",

      "Bash(ls:*)",
      "Bash(find:*)",
      "Bash(grep:*)",
      "Bash(rg:*)",
      "Bash(wc:*)",
      "Bash(file:*)",

      "Bash(gh pr view:*)",
      "Bash(gh pr list:*)",
      "Bash(gh issue view:*)",
      "Bash(gh issue list:*)",
      "Bash(gh run view:*)",
      "Bash(gh run list:*)"
    ]
  }
}
```

故意**不进**允许列表(每次都会提示)的命令:

| 命令 | 原因 |
|------|------|
| `git push`, `git push --force` | 推到远端,影响他人 |
| `git commit`, `git commit --amend` | 改历史 |
| `make push`, `make pullpush` | 内部包了 git push |
| `make sync`, `make addupstream` | 改 master / 远端配置 |
| `git reset --hard`, `git checkout -- *` | 销毁本地修改 |
| `rm`, `rm -rf` | 删文件 |
| `make docker:*` | 启容器,影响系统状态 |
| `gh pr create`, `gh issue create`, `gh pr merge` | 创建/合并 PR |
| `make:*`(整段通配) | 太宽,会把 push/sync/docker 也覆盖 |

### 4.4 `docs/superpowers/` 骨架

目录结构:

```
docs/superpowers/
├── README.md           工作流说明
├── specs/
│   └── .gitkeep
└── plans/
    └── .gitkeep
```

`docs/superpowers/README.md` 内容:

````markdown
# Chain33 + Superpowers 工作流

本目录存放使用 Claude(superpowers skill 系统)开发 chain33 时产生的设计文档与实施计划。

## 何时启用完整流程

| 场景                                       | 走完整流程 | 直接做 |
|--------------------------------------------|-----------|--------|
| 新模块 / 新功能 / 跨多文件重构              |    ✓     |        |
| 改 protobuf 协议 / 共识规则 / p2p 协议      |    ✓     |        |
| 大于 ~50 行的逻辑改动                       |    ✓     |        |
| 修 typo / 单行 bug / 注释                   |          |   ✓    |
| 跑 lint / 格式化 / 测试                     |          |   ✓    |
| 加测试覆盖率(单个文件、思路明确)            |          |   ✓    |
| 加测试覆盖率(跨模块、需先理清边界)          |    ✓     |        |

## 流程(非平凡改动)

1. **Brainstorm** — 自然语言说出想法,或调 `brainstorming` skill
   - Claude 会反问澄清,提 2-3 方案,与你确认设计
   - 输出:你批准过的设计

2. **Spec** — 设计写入 `specs/YYYY-MM-DD-<topic>-design.md`
   - 命名:日期 + 蛇形主题,如 `2026-05-19-p2p-tss-keygen-design.md`
   - 平均 50-150 行,提交进 git

3. **Plan** — 调 `writing-plans` skill,把 spec 拆成可执行步骤
   - 写入 `plans/YYYY-MM-DD-<topic>-plan.md`
   - 步骤拆到能 TDD 的粒度,提交进 git

4. **Execute** — 调 `executing-plans` 或 `subagent-driven-development`
   - 按 plan 逐步执行,每步 TDD(test-driven-development skill)
   - 每完成一段跑 `make test && make linter`

5. **Verify** — 调 `verification-before-completion`
   - 凭实际命令输出宣布完成,不靠"应该会过"
   - 并发代码必跑 `make race`

## 目录约定

- `specs/` — 设计文档,与 git 历史长期保留(可追溯为什么这么设计)
- `plans/` — 实施计划,与 git 历史长期保留(可追溯做了哪些步骤)
- 完工后不需要删 spec/plan,它们是项目记忆的一部分

## 常用 skill 快查

| skill                              | 何时用                              |
|------------------------------------|------------------------------------|
| `brainstorming`                    | 任何创造性 / 设计性的开始            |
| `writing-plans`                    | spec → 实施计划                     |
| `executing-plans`                  | 隔离 session 跑计划                  |
| `subagent-driven-development`      | 当前 session 用 subagent 跑独立任务  |
| `test-driven-development`          | 写新代码、修 bug 的节奏               |
| `systematic-debugging`             | 遇到 bug、测试失败、行为异常          |
| `verification-before-completion`   | 宣布完成前的硬性核验                  |
| `requesting-code-review`           | 完工前请 reviewer                   |
| `receiving-code-review`            | 处理 review 意见的态度                |
| `using-git-worktrees`              | 需要隔离工作区                        |

完整 skill 列表见 superpowers 插件文档。
````

## 5. 产出文件清单

| 文件 | 类型 | 大小估计 |
|------|------|---------|
| `CLAUDE.md` | 新建 | ~120-150 行 |
| `.claude/settings.json` | 新建 | ~55 行 |
| `.gitignore` | 修改(`.claude*` 一行替换成 `.claude/*` + `!.claude/settings.json` 两行) | — |
| `docs/superpowers/README.md` | 新建 | ~65 行 |
| `docs/superpowers/specs/.gitkeep` | 新建 | 空 |
| `docs/superpowers/plans/.gitkeep` | 新建 | 空 |
| `docs/superpowers/specs/2026-05-19-claude-collaboration-setup-design.md` | 新建(本文件) | — |

合计:**6 个新文件 + 1 个修改**(本设计文档本身计入新文件)

## 6. 验收标准

实施完成后必须满足:

1. **结构**:`tree -L 2 docs/superpowers .claude` 显示上述目录结构,没有多余文件
2. **JSON 合法**:`python3 -m json.tool .claude/settings.json` 无报错
3. **gitignore 生效**:`git check-ignore -v .claude/settings.local.json` 显示被忽略;`git check-ignore -v .claude/settings.json` 显示**未**被忽略
4. **Markdown 合法**:`CLAUDE.md` 和 `docs/superpowers/README.md` 用 markdown lint 工具(或人眼)扫一遍,没有破碎表格、没有未闭合代码块
5. **CLAUDE.md 命令真实**:CLAUDE.md 里提到的每个 `make` 命令都能在 Makefile 里找到(`make test`、`make race`、`make coverage` 等)
6. **不破坏构建**:`make test` 在引入这些文件后依然通过
7. **提交分离**:理想情况下,本工作分 2-3 个 commit 提交(基础结构 / settings / docs)

## 7. 后续工作(不在本次范围)

- 用 1-2 个月跑实际 spec 流程,观察哪些痛点反复出现
- 视真实情况添加:
  - hooks(如 commit 前 lint、保存后 gofmt)
  - 专属 agents(如 `chain33-tester`、`protobuf-expert`)
  - `CONTRIBUTING-WITH-CLAUDE.md`(团队协作完整指南)
- 在每个主要模块下,视需要加 `<module>/CLAUDE.md`(模块级提示)

---

**审批与签字**

- [ ] 设计已 review
- [ ] 同意进入 `writing-plans` 阶段
