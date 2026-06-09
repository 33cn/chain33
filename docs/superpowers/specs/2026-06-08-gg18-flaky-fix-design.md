# gg18 集成测试 flaky 根治设计

- 日期:2026-06-08
- 关联:PR #1361(fix: address 20 bugs from codebase review)CI 中 `Run test cases for gg18` job 偶发失败
- 范围:仅 `system/crypto/tss/gg18/gg18_integration_test.go`

## 1. 背景与现象

PR #1361 的 CI 唯一失败的检查是 `Run test cases for gg18`。决定性证据:**同一个 commit `9eabcd5d7`,在 `push` 事件的 workflow 里 gg18 通过(2m23s),在 `pull_request` 事件里失败(12m8s)** —— 典型 flaky。

CI 命令(`.github/workflows/build.yml`):

```
cd system/crypto/tss/gg18
go test -race -v -run ^TestGG18_4Node$ .
```

失败信息(`gg18_integration_test.go`):

- `:160 test 3 node concurrent sign timeout, role=node3 / node2`
- `:147 Received unexpected error: tss listener context done: context deadline exceeded`,随后大量 sign-session 级联 `context canceled`
- `--- FAIL: TestGG18Node` / `--- FAIL: TestGG18_4Node`,包级 `FAIL ... 119.998s`

## 2. 根因

测试用**墙钟时间预算**判定 PASS/FAIL,而 CI 共享 runner 的 CPU 时间不可控,叠加 `-race`(放大 2–10 倍)与每路签名现生成 2048-bit Paillier 密钥(`paillier.NewPaillier(2048)`,素数搜索,耗时本身随机),必然偶发超时。

1. **核心**:`runNodeFlow` 末尾起 10 路并发 `ProcessSign`,外层用 `select { case <-c: case <-time.After(60s): t.Fatalf }` 作为"预算闸"。60s 甚至比单路内部 1min 超时还短,自相矛盾;**只要 CI 算得慢就误判失败**。历史上 `b6b4a46e4` 仅加大超时,治标未治本。
2. 每路 `ProcessSign` 的 Paillier 2048 生成是 CPU 杀手且耗时方差大,10 路 × 3 节点并发放大长尾。
3. `runNodeFlow` 中还有一连串墙钟闸同样会因"算得慢"误判:`waitPeerIDs(4,120s)`、`waitPeerIDs(3,30s)`、`waitBarrier(60s)`、`waitChildExit(30s)`。

与时间无关的真正正确性判据是 `verifySignatureWithDKG`(密码学验签)。

## 3. 设计原则

把"用墙钟时间判对错"换成"用密码学验签判对错 + 兜底超时只防真死锁":

- **成功判据** = `verifySignatureWithDKG` 验签,与耗时无关。
- **失败判据** = 签名出错 / 验签不过(确定性);真死锁由各操作的兜底超时与 `go test` 包级超时兜底。
- 删除所有"预算型"墙钟闸;保留的超时全部放宽为"远大于正常耗时"的兜底值。

**边界(已与维护者确认)**:仅改 `gg18_integration_test.go`,不碰生产代码(`api.go`)与 CI 配置(`build.yml`);并发路数 10 → 4。

## 4. 改动清单(均在 gg18_integration_test.go)

1. 删除并发段外层 `select{ ... time.After(60s) ... t.Fatalf }`,改为 `wg.Wait()` —— 移除唯一的"预算型"失败分支。
2. 并发路数 `10 → concurrentSigns(4)`。
3. 集中超时常量并放宽兜底:

| 常量 | 新值 | 用途 | 原值 |
|---|---|---|---|
| `peerWait4Timeout` | 150s | 等齐 4 节点(含自身)发现 | 120s |
| `peerWait3Timeout` | 90s | 等齐 3 签名节点发现 | 30s |
| `dkgOpTimeout` | 2min | DKG 兜底 | 2min |
| `signOpTimeout` | 90s | 单签兜底(含并发段每路) | 1min |
| `reshareOpTimeout` | 2min | reshare 兜底 | 2min |
| `barrierTimeout` | 120s | 进程间 barrier 同步 | 60s |
| `childExitTimeout` | 120s | 等子进程退出 | 30s |
| `concurrentSigns` | 4 | 并发签名路数 | 10 |

各阶段"全踩满兜底"的串行总和 < `go test` 默认 10min 包级超时,留充分余量。

## 5. 为什么是"完全无概率"(结构性保证)

改后测试体内**不再存在"因耗时而失败"的代码路径**。失败只来自 `require.NoError`(签名出错)或验签不过,均与"算多慢"无关 —— 签名要么成功要么失败、验签要么过要么不过,是确定性的。兜底超时只在真死锁时触发,数值远大于正常耗时,正常/慢都碰不到。

唯一残留的全局线是 `go test` 默认 10min 包级超时,而修复版 race 仅 ~15s,CI 慢 10 倍也才 ~2.5min,数量级安全。

## 6. 验证结果

本地 `-race`(macOS arm64):

- 连续 10 次:**10/10 PASS,no data race**。
- 详细单跑:**15.43s**,并发段 12 路(4×3)**全部 start+end**,验签失败 **0**。
- `gofmt`、`go vet` 通过。

注:本地芯片比 CI 快 3–5 倍(基线 16.77s,烧满 28 个 CPU 进程也仅 19s),**无法自然复现 CI 的慢**;"慢也不失败"依据第 5 节的结构性保证,而非本地慢场景实测。CI 失败的性质由 CI 日志直接证明(失败是超时闸 + context deadline,无验签失败)。

## 7. 风险与回滚

- 并发 10→4 降低并发覆盖强度,但仍覆盖"多会话并发签名互不串扰";如需更强可调大 `concurrentSigns`,不影响稳定性结论。
- 改动仅限单个测试文件;回滚 = `git checkout -- system/crypto/tss/gg18/gg18_integration_test.go`。

## 8. 可选后续(本次不做,均超出"仅改测试"边界)

- CI 的 `go test` 加 `-timeout 30m`,消除对 10min 默认线的理论依赖。
- 生产侧 `api.go` 复用/预生成 Paillier 密钥,从根上削减最大耗时源(需全量回归 `make test && make race`)。
