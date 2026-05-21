# Bugfix Report: Top-20 Findings #2-#20

## F-MAVL-001: VerifyKVPairProof 对畸形 proof nil 解引用

**Severity:** Critical | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`system/store/mavl/db/tree.go:785-789` 中，`ReadProof()` 返回错误后代码没有 `return`，
继续执行 `proofnode.Verify()` 对 nil 指针调用方法，导致 panic。

任何外部节点发送畸形 proof 数据即可远程崩溃目标节点。

### 触发条件

- 收到无效/损坏的 KVPair proof 字节（网络攻击或数据损坏）

### 修复方法

在 `ReadProof` 返回 error 后立即 `return false`。

```go
// Before (buggy)
proofnode, err := ReadProof(roothash, leafHash, proof)
if err != nil {
    treelog.Info("VerifyKVPairProof ReadProof err！", "err", err)
}
istrue := proofnode.Verify(...)  // nil dereference!

// After (fixed)
proofnode, err := ReadProof(roothash, leafHash, proof)
if err != nil {
    treelog.Info("VerifyKVPairProof ReadProof err！", "err", err)
    return false
}
istrue := proofnode.Verify(...)
```

### 测试文件

`system/store/mavl/db/verify_bug_test.go`

---

## F-DL-002: downloadBlockFromPeerOld 对恶意回复 panic

**Severity:** Critical | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`system/p2p/dht/protocol/download/download.go:150` 中，对 peer 返回的响应直接做
`resp.Message.Items[0].Value.(*types.InvData_Block).Block`，没有任何防御性检查：
- `resp.Message` 可能为 nil
- `Items` 可能为空切片
- 类型断言可能失败（peer 返回非 Block 类型）

单个恶意 peer 即可通过返回空响应永久崩溃同步流程。

### 触发条件

- 恶意 peer 返回空 `InvDatas{Items: nil}`
- 恶意 peer 返回 nil Message
- 恶意 peer 返回错误类型的 InvData

### 修复方法

添加 nil 检查、长度检查和 comma-ok 类型断言。

```go
// Before (buggy)
block := resp.Message.Items[0].Value.(*types.InvData_Block).Block

// After (fixed)
if resp.Message == nil || len(resp.Message.Items) == 0 {
    return nil, fmt.Errorf("empty block response from peer")
}
blockData, ok := resp.Message.Items[0].Value.(*types.InvData_Block)
if !ok || blockData == nil || blockData.Block == nil {
    return nil, fmt.Errorf("invalid block data in response")
}
return blockData.Block, nil
```

### 测试文件

`system/p2p/dht/protocol/download/download_bug_test.go`

---

## F-DL-003: wg.Done() 未 defer，panic/cancel 路径死锁

**Severity:** Critical | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`system/p2p/dht/protocol/download/handler.go:141-169` 中，goroutine 内的
`wg.Done()` 放在函数末尾而非 `defer`。如果 `downloadBlock` panic 或
line 149 的 early return 被触发，`wg.Done()` 被跳过，`wg.Wait()` 永久阻塞。

### 触发条件

- `downloadBlock` 内部 panic（如网络异常导致的 nil 解引用）
- context cancel 后的 early return 路径

### 修复方法

将 `wg.Done()` 改为 goroutine 顶部的 `defer wg.Done()`。

```go
// Before (buggy)
go func(blockheight int64, tasks tasks) {
    err := p.downloadBlock(blockheight, tasks)
    // ... error handling with early returns ...
    wg.Done()  // skipped on panic or early return!
}(height, jobS)

// After (fixed)
go func(blockheight int64, tasks tasks) {
    defer wg.Done()
    err := p.downloadBlock(blockheight, tasks)
    // ... error handling ...
}(height, jobS)
```

### 测试文件

`system/p2p/dht/protocol/download/wgdone_bug_test.go`

---

## F-QUE-001: queue.Close() 并发 panic（send on closed channel）

**Severity:** Critical | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`queue/queue.go:159-184` 中，`isClosed()` 检查在锁外面（仅用 atomic），两个 goroutine
可以同时通过检查。第一个执行 `close(q.done)` 后，第二个再执行 `q.done <- struct{}{}`
或 `close(q.done)` 时 panic（send/close on closed channel）。

关机路径上任何并发 Close 调用都会崩节点。

### 触发条件

- 多个模块同时调用 `queue.Close()`（关机时常见）

### 修复方法

用 `sync.Once` 保证 Close 逻辑只执行一次。

```go
// Before (buggy)
func (q *queue) Close() {
    if q.isClosed() { return }  // race: two goroutines pass this
    q.mu.Lock()
    // ...
    q.mu.Unlock()
    q.done <- struct{}{}
    close(q.done)  // second caller panics here
    atomic.StoreInt32(&q.isClose, 1)
}

// After (fixed)
func (q *queue) Close() {
    q.closeOnce.Do(func() {
        q.mu.Lock()
        // ...
        q.mu.Unlock()
        q.done <- struct{}{}
        close(q.done)
        atomic.StoreInt32(&q.isClose, 1)
    })
}
```

### 测试文件

`queue/queue_close_bug_test.go`（用 -race 运行可复现）

---

## F-P2P-MGR-002: procConnections 定时 panic 于坏 RelayNodeAddr

**Severity:** Critical | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`system/p2p/dht/manage/conns.go:188` 中，当 `RelayNodeAddr` 配置格式错误时，
`genAddrInfo()` 返回 error，代码直接 `panic()`。该函数在定时器中每 2 分钟执行一次，
意味着配置错误会在启动 2 分钟后崩溃节点。

### 触发条件

- `RelayNodeAddr` 配置中有格式错误的 multiaddr 字符串
- `RelayEnable = true`

### 修复方法

将 `panic` 替换为 `log.Error` + `continue`。

```go
// Before (buggy)
if err != nil {
    panic(`invalid relayNodeAddr in config...`)
}

// After (fixed)
if err != nil {
    log.Error("procConnections invalid relayNodeAddr", "error", err, "addr", node)
    continue
}
```

### 测试文件

`system/p2p/dht/manage/conns_bug_test.go`

---

## F-CLI-003: RemoveTxsByHashList 吞服务端错误（逻辑反转）

**Severity:** Critical | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`client/queueprotocol.go:216-221` 中，对 `msg.GetData().(error)` 的类型断言结果处理逻辑反转：

- 当 `ok=true`（数据 IS error）时，`err` 持有实际错误，但代码返回 `nil`
- 当 `ok=false`（数据 NOT error）时，`err` 为 nil，代码返回 `nil`（碰巧正确）

结果：mempool 返回的所有错误都被静默吞掉。已入块的 tx 不会从 mempool 移除。

### 触发条件

- mempool 处理 `EventDelTxList` 返回任何错误时

### 修复方法

将 `if !ok` 改为 `if ok`。

```go
// Before (buggy)
err, ok = msg.GetData().(error)
if !ok {
    return err   // ok=false → err=nil → returns nil (accidentally correct)
}
return nil       // ok=true → err=real_error → returns nil (BUG!)

// After (fixed)
err, ok = msg.GetData().(error)
if ok {
    return err   // ok=true → err=real_error → returns error (correct)
}
return nil       // ok=false → success
```

### 测试文件

`queue/removetxs_bug_test.go`

---

## F-BCSYNC-001: push.postwg.Add(1) 在 goroutine 启动后调用

**Severity:** Critical | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`blockchain/push.go:692-693` 中，`push.postwg.Add(1)` 在 `go func(...)` 之后调用。
如果 goroutine 快速执行并调用 `postwg.Done()`，而 `Add(1)` 尚未执行，
WaitGroup 计数变为负数导致 panic。

在快速关机场景下（`closechan` 立即关闭），goroutine 会立即调用 `Done()`。

### 触发条件

- 快速关机：subscribe 的 closechan 在 goroutine 启动后立即关闭
- 高负载下 goroutine 调度延迟导致 Add 和 Done 顺序颠倒

### 修复方法

将 `postwg.Add(1)` 移到 `go func()` 之前。

```go
// Before (buggy)
go func(in *pushNotify) {
    // ... 可能快速执行 Done() ...
    push.postwg.Done()
}(input)
push.postwg.Add(1)  // 可能在 Done() 之后才执行!

// After (fixed)
push.postwg.Add(1)  // 保证在 goroutine 启动前计数
go func(in *pushNotify) {
    // ...
    push.postwg.Done()
}(input)
```

### 测试文件

`queue/push_wg_bug_test.go`（pattern 演示，放在 queue 包避免 Go 1.24 链接问题）

---

# Bugfix Report: Top-20 Findings #9-#20

## F-EXEC-009: execenv 缓存更新条件逻辑错误（&& 应为 ||）

**Severity:** Important | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`executor/execenv.go:360` 中，判断当前交易是否已更新的条件使用了 `&&`：
`if e.currExecTx != tx && e.currTxIdx != index`

注释说"均不相等时"，但 `&&` 要求两者同时不同才更新缓存。如果只有 tx 变了（同 index）
或只有 index 变了（同 tx），缓存不会更新，导致使用过期的 driver 缓存。

### 触发条件

- 同一 index 位置的交易被替换（如 mempool 替换）
- 同一交易出现在不同 index（如交易组重排）

### 修复方法

将 `&&` 改为 `||`，任一变化即更新缓存。

```go
// Before (buggy)
if e.currExecTx != tx && e.currTxIdx != index {

// After (fixed)
if e.currExecTx != tx || e.currTxIdx != index {
```

### 测试文件

`queue/execenv_logic_bug_test.go`

---

## F-RPC-004: CORS 预检请求方法名拼写错误（OPTION → OPTIONS）

**Severity:** Important | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`rpc/ethrpc/rpc.go:218` 中，CORS 预检请求检查为 `r.Method == "OPTION"`（单数），
但 HTTP 规范中 CORS preflight 使用的方法名是 `"OPTIONS"`（复数）。

浏览器发送的 CORS 预检请求永远不会匹配，导致跨域请求失败。

### 触发条件

- 浏览器通过 ethrpc 端口发送跨域 RPC 请求

### 修复方法

将 `"OPTION"` 改为 `"OPTIONS"`。

```go
// Before (buggy)
if r.Method == "OPTION" {

// After (fixed)
if r.Method == "OPTIONS" {
```

### 测试文件

`queue/rpc_option_bug_test.go`

---

## F-RPC-005: ethrpc Start() 监听失败时 panic 崩溃进程

**Severity:** Important | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`rpc/ethrpc/rpc.go:188` 中，`net.Listen("tcp", h.endpoint)` 失败时直接
`panic(err)`。生产环境中端口被占用是常见情况，不应崩溃整个进程。

### 触发条件

- 配置的 ethrpc 端口已被其他进程占用
- 端口权限不足（如非 root 绑定 <1024 端口）

### 修复方法

将 `panic(err)` 改为 `return 0, err`，让调用方处理错误。

```go
// Before (buggy)
l, err := net.Listen("tcp", h.endpoint)
if err != nil {
    panic(err)
}

// After (fixed)
l, err := net.Listen("tcp", h.endpoint)
if err != nil {
    return 0, err
}
```

### 测试文件

`queue/rpc_panic_listen_bug_test.go`

---

## F-RPC-003: ethrpc Close() 不关闭 HTTP server（资源泄漏）

**Severity:** Important | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`rpc/ethrpc/rpc.go:208-214` 中，`Close()` 方法有两个问题：
1. `else if h.wsHander != nil` 与 `if h.wsHander != nil` 条件相同——死代码
2. 从未调用 `h.server.Shutdown()` 或关闭 `h.listener`

结果：调用 Close() 后 HTTP server 继续运行，端口不释放。

### 触发条件

- 正常关机流程调用 Close()

### 修复方法

删除死代码分支，添加 `h.server.Shutdown(context.Background())`。

```go
// Before (buggy)
func (h *httpServer) Close() {
    if h.wsHander != nil {
        h.wsHander.server.Stop()
    } else if h.wsHander != nil {  // dead code!
        h.wsHander.server.Stop()
    }
    // HTTP server never stopped!
}

// After (fixed)
func (h *httpServer) Close() {
    if h.wsHander != nil {
        h.wsHander.server.Stop()
    }
    if h.server != nil {
        h.server.Shutdown(context.Background())
    }
}
```

### 测试文件

`queue/rpc_close_leak_bug_test.go`

---

## F-EXEC-001: globalPlugins map 迭代非确定性（共识分叉风险）

**Severity:** Critical | **Effort:** Small | **Confidence:** High

### Bug 描述

`executor/executor.go:416,516` 中，`for name, plugin := range globalPlugins`
直接迭代 map。Go map 迭代顺序是非确定性的，导致 plugin 产生的 KV pairs
在不同节点上以不同顺序追加到 `kvset.KV`。

在区块链中，所有节点必须产生完全相同的状态，KV 顺序不同 = 状态哈希不同 = 共识分叉。

### 触发条件

- 任何区块执行（每个区块都会触发 plugin 迭代）
- 多个 plugin 同时启用时（当前有 6 个 plugin）

### 修复方法

添加 `sortedPluginNames()` 辅助函数，排序后按固定顺序迭代。

```go
// Before (buggy)
for name, plugin := range globalPlugins {

// After (fixed)
for _, name := range sortedPluginNames() {
    plugin := globalPlugins[name]
```

### 测试文件

`queue/exec_map_order_bug_test.go`

---

## F-RPC-002: paraseDERCode 对畸形 DER 签名 panic

**Severity:** Critical | **Effort:** Trivial | **Confidence:** High

### Bug 描述

`rpc/ethrpc/types/tx.go:127-147` 中，`paraseDERCode` 函数存在多个 panic 向量：
1. 无长度检查直接访问 `sig[0]`、`sig[2]`、`sig[3]`
2. `sig[3]` 作为长度使用但无边界检查（如 0xFF 会越界）
3. 初始守卫用 `&&` 而非 `||`，大部分畸形输入通过守卫

任何外部 Ethereum RPC 请求携带畸形 DER 签名即可崩溃节点。

### 触发条件

- 外部发送畸形/截断的 DER 编码签名
- 签名长度 < 4 字节

### 修复方法

添加完整的边界检查，每步索引前验证长度。

```go
// Before (buggy)
if sig[0] != 0x30 && sig[2] != 0x02 {  // wrong: && lets most through
    return nil, nil, fmt.Errorf("no der code")
}
r = sig[4 : int(sig[3])+4]  // panic if sig[3] > len(sig)

// After (fixed)
if len(sig) < 4 {
    return nil, nil, fmt.Errorf("DER signature too short")
}
if sig[0] != 0x30 || sig[2] != 0x02 {
    return nil, nil, fmt.Errorf("no der code")
}
rLen := int(sig[3])
if len(sig) < 4+rLen {
    return nil, nil, fmt.Errorf("DER R length exceeds signature")
}
// ... similar bounds checks for S component
```

### 测试文件

`queue/rpc_dercode_panic_bug_test.go`

---

## F-ACCT-001: Transfer/depositBalance 余额加法溢出

**Severity:** Critical | **Effort:** Small | **Confidence:** High

### Bug 描述

`account/account.go:135` 中 `accTo.Balance = accTo.GetBalance() + amount`
和 `account/account.go:161` 中 `acc1.Balance += amount` 都没有溢出检查。

同文件的 `GenesisInit`（genesis.go:22）正确使用了 `safeAdd()`，但 `Transfer`
和 `depositBalance` 遗漏了。int64 溢出会导致余额变为负数。

### 触发条件

- 向已有大余额的账户转入大额（使 balance+amount > MaxInt64）
- 向已有大余额的执行器地址 deposit

### 修复方法

在 `Transfer` 和 `depositBalance` 中使用 `safeAdd()`。

```go
// Before (buggy)
accTo.Balance = accTo.GetBalance() + amount

// After (fixed)
newBalance, err := safeAdd(accTo.GetBalance(), amount)
if err != nil {
    return nil, err
}
accTo.Balance = newBalance
```

### 测试文件

`queue/account_overflow_bug_test.go`

---

## F-WAL-001: AES-GCM nonce 重用（密钥派生 nonce）

**Severity:** Critical | **Effort:** Small | **Confidence:** High

### Bug 描述

`wallet/seed.go:210` 中，AES-GCM 的 nonce 使用 `key[:12]`（密钥前 12 字节）。
这意味着同一密码加密任何内容时 nonce 完全相同。AES-GCM 在 nonce 重用时安全性
彻底崩溃：攻击者可恢复明文并伪造认证标签。

### 触发条件

- 同一密码加密 seed 两次（如备份恢复流程）
- 理论上只需一次加密即存在风险（nonce 可预测）

### 修复方法

使用 `crypto/rand` 生成随机 12 字节 nonce，prepend 到密文前。
解密时先尝试新格式（前 12 字节为 nonce），失败则回退到旧格式（兼容已有数据）。

```go
// Before (buggy)
Encrypted := aesgcm.Seal(nil, key[:12], seed, nil)

// After (fixed)
nonce := make([]byte, 12)
io.ReadFull(rand.Reader, nonce)
ciphertext := aesgcm.Seal(nil, nonce, seed, nil)
return append(nonce, ciphertext...), nil
```

### 测试文件

`queue/wallet_crypto_bug_test.go`

---

## F-WAL-002: AES-CBC IV 重用（密钥派生 IV）

**Severity:** Critical | **Effort:** Small | **Confidence:** High

### Bug 描述

`wallet/common/crypto.go:25` 中，AES-CBC 的 IV 使用 `key[:block.BlockSize()]`
（密钥前 16 字节）。同一密码加密不同私钥时 IV 相同，相同明文产生相同密文，
泄漏明文信息。

### 触发条件

- 同一钱包密码加密多个私钥

### 修复方法

使用 `crypto/rand` 生成随机 16 字节 IV，prepend 到密文前。
解密时检测新格式（长度 = BlockSize + 原始密文长度），回退旧格式兼容已有数据。

```go
// Before (buggy)
iv := key[:block.BlockSize()]
encrypter := cipher.NewCBCEncrypter(block, iv)

// After (fixed)
iv := make([]byte, block.BlockSize())
io.ReadFull(rand.Reader, iv)
encrypter := cipher.NewCBCEncrypter(block, iv)
return append(iv, Encrypted...)
```

### 测试文件

`queue/wallet_crypto_bug_test.go`

---

## F-CMD-001: Webhook RCE（空 secret + 未验证输入 + 执行攻击者代码）

**Severity:** Critical | **Effort:** Small | **Confidence:** High

### Bug 描述

`cmd/webhook/main.go` 存在多个严重安全漏洞：
1. **空 webhook secret**（line 22）：`github.New(github.Options.Secret(""))`，
   任何人可伪造 webhook payload
2. **未验证用户输入**（line 69-73）：`user` 和 `branch` 直接来自 payload，
   用于构造文件路径和 git URL，可路径穿越或命令注入
3. **执行攻击者代码**（line 84）：`make webhook` 执行克隆仓库的 Makefile，
   攻击者完全控制 Makefile 内容 = 任意代码执行

### 触发条件

- 攻击者向 webhook 端点发送伪造的 PullRequest payload
- payload 中 user.login 包含 `../` 或特殊字符

### 修复方法

1. 要求从环境变量读取非空 webhook secret
2. 对 user 和 branch 做正则白名单验证（`^[a-zA-Z0-9._-]+$`）

```go
// Before (buggy)
hook, _ := github.New(github.Options.Secret(""))

// After (fixed)
secret := os.Getenv("WEBHOOK_SECRET")
if secret == "" {
    log.Fatal("WEBHOOK_SECRET required")
}
hook, _ := github.New(github.Options.Secret(secret))
// + safeNamePattern.MatchString(user/branch) 验证
```

### 测试文件

`queue/webhook_rce_bug_test.go`

---

## F-RPC-001: ethrpc 绕过 IP 白名单认证

**Severity:** Critical | **Effort:** Small | **Confidence:** High

### Bug 描述

`rpc/ethrpc/rpc.go:219-239` 中，`ServeHTTP` 方法仅记录远程 IP 但从不检查
IP 白名单。标准 JSON-RPC 处理器（`rpc/http.go:67`）会调用 `checkIPWhitelist()`
和 `checkBasicAuth()` 拒绝未授权请求。

ethrpc 端口对所有来源 IP 完全开放，绕过了整个认证层。

### 触发条件

- 任何外部 IP 通过 ethrpc 端口发送请求

### 修复方法

在 `ServeHTTP` 中添加 IP 白名单检查（使用 RPC 配置中的 Whitelist 字段）。

```go
// Before (buggy)
ip, _, _ := net.SplitHostPort(r.RemoteAddr)
if utils.IsPublicIP(ip) {
    log.Debug(...)  // only logs, never blocks
}
h.httpHandler.ServeHTTP(w, r)  // always forwards

// After (fixed)
ip, _, _ := net.SplitHostPort(r.RemoteAddr)
if !h.checkIPWhitelist(ip) {
    http.Error(w, "IP not authorized", http.StatusForbidden)
    return
}
```

### 测试文件

`queue/ethrpc_auth_bypass_bug_test.go`

---

## F-DL-001: tasks 切片并发访问（数据竞争）

**Severity:** Critical | **Effort:** Medium | **Confidence:** High

### Bug 描述

`system/p2p/dht/protocol/download/handler.go:141-168` 中，多个 goroutine
共享同一个 `jobS`（tasks 切片）。在 `downloadBlock` 内部，`tasks.Sort()` 和
`tasks.Remove()` 直接修改底层数组，没有任何同步保护。

多个 goroutine 同时 Sort/Remove 同一切片 = 数据竞争，可能导致 panic 或静默数据损坏。

### 触发条件

- 同步下载多个区块时（每个高度一个 goroutine，共享 tasks 列表）
- 某个 peer 下载失败触发 Remove

### 修复方法

给 `downloadBlock` 添加可选 `*sync.Mutex` 参数，在 handler.go 的并发调用处
传入共享 mutex，保护 Sort/Remove/availbTask 操作。

```go
// Before (buggy)
func (p *Protocol) downloadBlock(height int64, tasks tasks) error {
    tasks.Sort()  // 无锁，多 goroutine 同时修改
    // ...
    tasks = tasks.Remove(task)  // 无锁

// After (fixed)
func (p *Protocol) downloadBlock(height int64, tasks tasks, tasksMu ...*sync.Mutex) error {
    lockTasks()
    tasks.Sort()
    unlockTasks()
    // ...
    lockTasks()
    tasks = tasks.Remove(task)
    unlockTasks()
```

handler.go 调用处传入 mutex：
```go
err := p.downloadBlock(blockheight, tasks, &mutex)
```

### 测试文件

`queue/download_tasks_race_bug_test.go`
