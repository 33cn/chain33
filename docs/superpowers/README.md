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
