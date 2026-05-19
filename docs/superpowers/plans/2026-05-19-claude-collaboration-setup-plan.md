# Chain33 Claude 协作基础建设 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 chain33 仓库一次性建立使用 Claude(superpowers skill 系统)的基础设施 — 包括 `CLAUDE.md`、`.claude/settings.json`、`.gitignore` 调整,以及 `docs/superpowers/` 目录骨架。

**Architecture:** 6 个新文件 + 1 个修改文件,分 3 个独立 commit 交付。不改 chain33 业务代码,纯配置与文档。

**Tech Stack:** Markdown、JSON 配置、`.gitignore` 模式、`git`、`make`。

**Reference Spec:** [docs/superpowers/specs/2026-05-19-claude-collaboration-setup-design.md](../specs/2026-05-19-claude-collaboration-setup-design.md)

---

## Pre-flight

- [ ] **Step P1: 确认工作目录与 git 状态**

```bash
pwd
git rev-parse --abbrev-ref HEAD
git status --short
```

Expected:
- `pwd` 显示 `/Users/king/go/src/chain33`
- 当前分支显示出来(目前预期是 `fix/remove-codecov-badge` 或新建分支)
- 已有未追踪文件:`coverage_*.out`、`build/coverage/`、`docs/superpowers/`

- [ ] **Step P2: 确认 spec 文档已存在**

```bash
ls -la docs/superpowers/specs/2026-05-19-claude-collaboration-setup-design.md
```

Expected: 文件存在,大小 > 10KB

- [ ] **Step P3(可选,推荐):创建独立分支**

当前分支 `fix/remove-codecov-badge` 主题是 codecov,与 Claude 协作建设混在一起不合适。推荐:

```bash
git fetch upstream master 2>/dev/null || git fetch origin master
git checkout -b chore/setup-claude-collaboration
```

如果坚持在当前分支提交,跳过本步,但需在 Task 1 提交信息中说明。

---

## Task 1: docs/superpowers/ 骨架与 spec/plan 文档

**Files:**
- Create: `docs/superpowers/README.md`
- Create: `docs/superpowers/specs/.gitkeep`
- Create: `docs/superpowers/plans/.gitkeep`
- Stage(已存在): `docs/superpowers/specs/2026-05-19-claude-collaboration-setup-design.md`
- Stage(已存在,本文件): `docs/superpowers/plans/2026-05-19-claude-collaboration-setup-plan.md`

- [ ] **Step 1.1: 确认目录已存在**

```bash
ls -la docs/superpowers/
```

Expected: 显示 `specs/` 与 `plans/` 两个子目录。若不存在,先建:

```bash
mkdir -p docs/superpowers/specs docs/superpowers/plans
```

- [ ] **Step 1.2: 创建两个 .gitkeep 占位**

```bash
touch docs/superpowers/specs/.gitkeep docs/superpowers/plans/.gitkeep
```

Verify:

```bash
ls -la docs/superpowers/specs/ docs/superpowers/plans/
```

Expected: 各显示一个 `.gitkeep` 文件(+ specs/ 里已有 design.md,plans/ 里已有本 plan.md)

- [ ] **Step 1.3: 写 `docs/superpowers/README.md`**

用 Write 工具创建文件,内容(完整):

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

- [ ] **Step 1.4: 验证 README 内容**

```bash
wc -l docs/superpowers/README.md
head -3 docs/superpowers/README.md
```

Expected:
- 行数 50–80 之间
- 首行 `# Chain33 + Superpowers 工作流`

- [ ] **Step 1.5: 暂存并提交**

```bash
git add docs/superpowers/
git status --short
```

Expected: 列出 5 个新文件:
- `docs/superpowers/README.md`
- `docs/superpowers/plans/.gitkeep`
- `docs/superpowers/plans/2026-05-19-claude-collaboration-setup-plan.md`
- `docs/superpowers/specs/.gitkeep`
- `docs/superpowers/specs/2026-05-19-claude-collaboration-setup-design.md`

提交:

```bash
git commit -m "$(cat <<'EOF'
docs: add superpowers workflow scaffolding with setup spec and plan

Establish docs/superpowers/ as the standard location for design specs
and implementation plans produced via the Claude + superpowers workflow.

- README.md explains the brainstorm → spec → plan → execute → verify flow
- specs/ and plans/ subdirectories with .gitkeep placeholders
- Include the spec and plan for this Claude collaboration setup itself

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
git log -1 --stat
```

Expected: 1 个 commit 创建,5 个文件被加入

---

## Task 2: 项目根目录 CLAUDE.md

**Files:**
- Create: `CLAUDE.md`(项目根)

- [ ] **Step 2.1: 确认 CLAUDE.md 不存在**

```bash
test -f CLAUDE.md && echo "ALREADY EXISTS — STOP and review" || echo "OK to create"
```

Expected: `OK to create`

- [ ] **Step 2.2: 写 CLAUDE.md**

用 Write 工具创建文件,内容(完整):

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

- [ ] **Step 2.3: 验证文件存在且内容合理**

```bash
wc -l CLAUDE.md
head -3 CLAUDE.md
```

Expected: 行数 70–100;首行 `# Chain33 — Working with Claude`

- [ ] **Step 2.4: 验证 CLAUDE.md 提到的每个 make 命令都在 Makefile 中存在**

```bash
for cmd in test testq race coverage linter fmt build_ci depends branch push sync pull; do
  grep -qE "^${cmd}[:[:space:]]" Makefile && echo "✓ make $cmd" || echo "✗ make $cmd  MISSING"
done
```

Expected: 全部 `✓`。若有 `✗`,停下来,根据 Makefile 实际情况修正 CLAUDE.md。

- [ ] **Step 2.5: 暂存并提交**

```bash
git add CLAUDE.md
git status --short
```

Expected: 只显示 `A  CLAUDE.md` 一行新增

```bash
git commit -m "$(cat <<'EOF'
docs: add CLAUDE.md with project conventions for AI-assisted development

Document chain33-specific conventions Claude (and humans new to the
codebase) need to know: Makefile-wrapped git workflow, GOPATH path
constraint, GOPROXY for CN users, layered module map, testing
discipline, and pitfall list.

The conventions cover the chain33 specifics that aren't visible from
code alone — for instance, `make branch` / `make push` wrap git,
`types/*.pb.go` shouldn't be hand-edited, and `.travis.yml` /
`.gitlab-ci.yml` should not be touched without confirmation.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
git log -1 --stat
```

Expected: 1 个 commit 创建,`CLAUDE.md` 被加入

---

## Task 3: .gitignore 调整 + .claude/settings.json

**Files:**
- Modify: `.gitignore`(把 `.claude*` 一行替换为两行)
- Create: `.claude/settings.json`

**重要顺序**:必须先改 `.gitignore`,后建 `.claude/settings.json`。否则 `git add .claude/settings.json` 会被现有的 `.claude*` 规则拒绝。

- [ ] **Step 3.1: 确认 .gitignore 当前状态**

```bash
grep -n "claude" .gitignore
```

Expected: 一行显示 `<line_number>:.claude*`(预期约第 31 行)。记住这个行号。

- [ ] **Step 3.2: 改 .gitignore — 把 `.claude*` 替换为两行**

用 Edit 工具:
- `old_string`:`.claude*`
- `new_string`:`.claude/*` + 换行 + `!.claude/settings.json`

完成后验证:

```bash
grep -n "claude" .gitignore
```

Expected: 两行,内容为:
```
N:.claude/*
N+1:!.claude/settings.json
```

- [ ] **Step 3.3: 确认 .claude/ 目录存在**

```bash
mkdir -p .claude
ls -la .claude/
```

Expected: 空目录(`.` `..` 之外没有其他文件)

- [ ] **Step 3.4: 写 `.claude/settings.json`**

用 Write 工具创建文件,内容(完整):

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

- [ ] **Step 3.5: 验证 JSON 合法**

```bash
python3 -m json.tool .claude/settings.json > /dev/null && echo "✓ JSON valid"
```

Expected: `✓ JSON valid`。若报错,检查逗号、引号、花括号。

- [ ] **Step 3.6: 验证 gitignore 行为**

```bash
git check-ignore -v .claude/settings.json && echo "✗ BAD: settings.json IS ignored" || echo "✓ settings.json NOT ignored"
git check-ignore -v .claude/settings.local.json && echo "✓ settings.local.json IS ignored" || echo "✗ BAD: local should be ignored"
```

Expected: 两个 `✓`

- [ ] **Step 3.7: 暂存并提交**

```bash
git add .gitignore .claude/settings.json
git status --short
```

Expected:
```
M  .gitignore
A  .claude/settings.json
```

提交:

```bash
git commit -m "$(cat <<'EOF'
chore: add .claude/settings.json with safe-command allowlist

Add team-shared Claude Code permissions for chain33. Allowlist covers:
- make targets that are read-only or safely scoped (test, race, lint,
  coverage, fmt, depends, build, cli) — excludes push/sync/docker
- go subcommands that don't mutate remote state
- git read-only commands (status, diff, log, show, blame)
- search/navigation utilities (ls, find, grep, rg, wc, file)
- gh read commands (pr/issue/run view & list)

Commands that mutate shared state (git push, make push, gh pr create,
git reset --hard) stay outside the allowlist and prompt every time.

Adjust .gitignore so .claude/settings.json (team config) is tracked
while .claude/settings.local.json (personal overrides) stays ignored.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
git log -1 --stat
```

Expected: 1 个 commit 创建,包含 `.gitignore` 修改与 `.claude/settings.json` 新增

---

## Task 4: 最终验收(对应 spec § 6)

- [ ] **Step 4.1: 结构验证**

```bash
find docs/superpowers .claude -type f | sort
```

Expected output(顺序可能因文件系统而略异,但条目应一致):
```
.claude/settings.json
docs/superpowers/README.md
docs/superpowers/plans/.gitkeep
docs/superpowers/plans/2026-05-19-claude-collaboration-setup-plan.md
docs/superpowers/specs/.gitkeep
docs/superpowers/specs/2026-05-19-claude-collaboration-setup-design.md
```

- [ ] **Step 4.2: JSON 合法性**

```bash
python3 -m json.tool .claude/settings.json > /dev/null && echo "✓ settings.json valid"
```

Expected: `✓ settings.json valid`

- [ ] **Step 4.3: .gitignore 行为再确认**

```bash
git check-ignore -v .claude/settings.json && echo "✗ BAD" || echo "✓ tracked"
git check-ignore -v .claude/settings.local.json && echo "✓ ignored" || echo "✗ BAD"
```

Expected: 两个 `✓`

- [ ] **Step 4.4: Markdown 文件最小合法性扫描(无未闭合代码块)**

```bash
for f in CLAUDE.md docs/superpowers/README.md docs/superpowers/specs/2026-05-19-claude-collaboration-setup-design.md docs/superpowers/plans/2026-05-19-claude-collaboration-setup-plan.md; do
  fences=$(grep -c '^```' "$f")
  if [ $((fences % 2)) -eq 0 ]; then
    echo "✓ $f (代码块 fence 数: $fences)"
  else
    echo "✗ $f 有未闭合代码块"
  fi
done
```

Expected: 4 个 `✓`

- [ ] **Step 4.5: CLAUDE.md 引用的 make 命令在 Makefile 中存在**

```bash
for cmd in test testq race coverage linter fmt build_ci depends branch push sync pull; do
  grep -qE "^${cmd}[:[:space:]]" Makefile && echo "✓ make $cmd" || echo "✗ make $cmd MISSING"
done
```

Expected: 12 个 `✓`

- [ ] **Step 4.6: 提交历史核对(3 个 commit)**

```bash
git log --oneline -5
```

Expected: 最近 3 个 commit 分别对应:
1. `chore: add .claude/settings.json with safe-command allowlist`
2. `docs: add CLAUDE.md with project conventions for AI-assisted development`
3. `docs: add superpowers workflow scaffolding with setup spec and plan`

- [ ] **Step 4.7: 不破坏现有构建**

```bash
make test 2>&1 | tail -20
```

Expected: 与本次改动**之前**相同的测试结果(全部通过,或同样的预期失败)。本次改动不动业务代码,**不应**引入新的测试失败。

如果出现新失败,停下来排查 —— 不应该发生。

---

## 完工说明

完成 Task 1-4 后:

- 仓库根新增 `CLAUDE.md`、`.claude/settings.json`、`docs/superpowers/`
- `.gitignore` 调整 1 处
- git history 多出 3 个清晰可回滚的 commit
- 后续开发者可直接走 brainstorm → spec → plan → execute → verify 流程
- 后续 Claude 会话会自动加载 `CLAUDE.md`,无需手动 `Read`

**不在本次范围**(参 spec § 7):hooks、自定义 agents、`CONTRIBUTING-WITH-CLAUDE.md`、模块级 `<module>/CLAUDE.md`。

**推送**:本计划**不包含** `git push`。是否推送、推送到哪个 remote 由你决定。
