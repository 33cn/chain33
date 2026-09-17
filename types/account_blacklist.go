// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
)

// ForkAccountBlacklist 账户黑名单分叉名，配置在 [fork.system] 节，默认注册高度为 MaxHeight（永不启用）。
//
// 名单本身配置在 mver 段，按高度分版本演进，与 ForkChainParamV1/V2 的用法完全一致：
//
//	[mver.blacklist]
//	accountBlacklist=[]
//	[mver.blacklist.ForkAccountBlacklist]
//	accountBlacklist=["addr1", "addr2"]
//
// 高度 h 生效的名单是「高度不大于 h 的最大分叉」所对应的那一份，每一份都是全量名单而非增量。
//
// 新增一版名单的流程（三步缺一不可）：
//  1. 代码里加常量并在 RegisterSystemFork 中 SetFork(ForkAccountBlacklistV2, MaxHeight)
//  2. 全网 fork toml 加 ForkAccountBlacklistV2=<未来高度或 -1>
//  3. mver 加 [mver.blacklist.ForkAccountBlacklistV2] 全量名单
//
// 已经被链跨过的分叉，其高度与名单内容一律不可再改，只能追加新分叉。
// 否则历史区块回放会用新名单去判定旧区块，执行结果改变，直接破坏共识一致性。
const (
	ForkAccountBlacklist   = "ForkAccountBlacklist"
	ForkAccountBlacklistV2 = "ForkAccountBlacklistV2"
)

// mver 中黑名单配置的段前缀与键名
const (
	blacklistMverPrefix = "mver.blacklist."
	blacklistMverKey    = "accountBlacklist"
	blacklistBaseKey    = blacklistMverPrefix + blacklistMverKey
)

// evmExecName EVM 执行器真实名，用于从 payload 中解析真实目标地址
const evmExecName = "evm"

// blockedAccounts 硬编码的攻击地址名单，作为 mver 配置之外的兜底。
// 支持 base58 编码的 1x 地址与 0x 开头的十六进制地址混写。
// 注意：同一私钥派生的 base58 地址与 0x 地址 hash160 不同，是两个独立账户，
// 如两种形态都在使用，需分别列入。
//
// 该名单同样受 ForkAccountBlacklist 高度门控，只并入高度不小于该分叉的版本，
// 否则历史区块回放会把名单倒查到创世高度。业务名单请一律配在 [mver.blacklist.*] 段。
var blockedAccounts = []string{}

// blockedAccountSet 黑名单原始 20 字节地址集合。
// 以原始 20 字节为 key，天然兼容 base58 与 0x 两种地址形态。
type blockedAccountSet map[[20]byte]struct{}

func (s blockedAccountSet) hasRaw(raw []byte) bool {
	if len(raw) != 20 || len(s) == 0 {
		return false
	}
	var key [20]byte
	copy(key[:], raw)
	_, ok := s[key]
	return ok
}

func (s blockedAccountSet) merge(other blockedAccountSet) {
	for key := range other {
		s[key] = struct{}{}
	}
}

func (s blockedAccountSet) clone() blockedAccountSet {
	dup := make(blockedAccountSet, len(s))
	dup.merge(s)
	return dup
}

// blacklistVersion 一段高度区间内生效的黑名单快照，构建完成后只读
type blacklistVersion struct {
	height int64  // 该版本生效的起始高度（闭区间左端）
	name   string // 分叉名，base 版本为空串，仅用于日志定位
	set    blockedAccountSet
}

// accountBlacklist 按高度分版本的黑名单集合。
// 随 Chain33Config 初始化时一次性构建，之后只读，查询是纯查表，不加锁也不访问 mver。
type accountBlacklist struct {
	versions []*blacklistVersion // 按 height 升序，versions[0] 为 height 0 的 base 版本
}

// at 返回高度 h 生效的版本：versions 中 height 不大于 h 的最大者。
// 绝不选用 height > h 的未来版本。base 版本高度为 0，区块高度非负，故必有结果。
func (b *accountBlacklist) at(height int64) *blacklistVersion {
	for i := len(b.versions) - 1; i > 0; i-- {
		if height >= b.versions[i].height {
			return b.versions[i]
		}
	}
	return b.versions[0]
}

// newAccountBlacklist 由 mver 配置与分叉高度构建黑名单版本快照。
// skipForkCheck 对应 needSetForkZero 的场景（local 单测、未配 fork 的平行链），
// 此时所有分叉高度被强制归零、toml 也未必声明 fork，跳过分叉声明相关的校验。
func newAccountBlacklist(cfg *Config, forks *Forks, mver *mversion, skipForkCheck bool) *accountBlacklist {
	checkLegacyBlacklistSection(cfg)
	forkNames := parseBlacklistMverSections(mver, forks, skipForkCheck)
	if !skipForkCheck {
		checkBlacklistForkDeclared(cfg, forkNames)
	}
	blacklist := &accountBlacklist{versions: buildBlacklistVersions(mver, forks, forkNames)}
	for _, v := range blacklist.versions {
		tlog.Info("accountBlacklist version", "fork", v.name, "height", v.height, "size", len(v.set))
	}
	return blacklist
}

// checkLegacyBlacklistSection 拒绝残留的静态 [blacklist] 段。
// 该段已被 [mver.blacklist.*] 取代，若静默忽略，未同步迁移配置的部署会在毫无提示的情况下丢掉名单。
func checkLegacyBlacklistSection(cfg *Config) {
	if cfg == nil || cfg.Blacklist == nil || len(cfg.Blacklist.AccountBlacklist) == 0 {
		return
	}
	panic(fmt.Sprintf("[blacklist] accountBlacklist is no longer supported, %d addresses found;"+
		" move them into [%s%s] %s", len(cfg.Blacklist.AccountBlacklist),
		blacklistMverPrefix, ForkAccountBlacklist, blacklistMverKey))
}

// parseBlacklistMverSections 扫描 [mver.blacklist] 下的全部配置键，返回配置了名单的分叉名集合。
// 键名拼错、引用未注册的分叉都直接 panic：mver 是 per-key 稀疏覆盖，
// 段存在但键写错时该版本根本不会进入 versionList，运行期表现为静默沿用旧名单。
func parseBlacklistMverSections(mver *mversion, forks *Forks, skipForkCheck bool) map[string]struct{} {
	forkNames := make(map[string]struct{})
	if mver == nil {
		return forkNames
	}
	for key := range mver.data {
		if !strings.HasPrefix(key, blacklistMverPrefix) {
			continue
		}
		rest := strings.TrimPrefix(key, blacklistMverPrefix)
		fork, suffix, nested := strings.Cut(rest, ".")
		if !nested {
			if rest != blacklistMverKey {
				panic(fmt.Sprintf("unknown blacklist config key %s, expect %s", key, blacklistBaseKey))
			}
			continue
		}
		if suffix != blacklistMverKey {
			panic(fmt.Sprintf("unknown blacklist config key %s, expect %s%s.%s",
				key, blacklistMverPrefix, fork, blacklistMverKey))
		}
		if !skipForkCheck && !forks.HasFork(fork) {
			panic(fmt.Sprintf("blacklist config %s refers to unregistered fork %s,"+
				" the fork must be registered in RegisterSystemFork first", key, fork))
		}
		forkNames[fork] = struct{}{}
	}
	return forkNames
}

// checkBlacklistForkDeclared 校验 toml 中已启用的黑名单分叉都配了对应的 mver 名单段。
// 只认 toml 里的原始声明，不能用运行时高度：needSetForkZero 会把所有分叉压成 0，
// 按运行时高度判定会让默认本地配置直接起不来。
func checkBlacklistForkDeclared(cfg *Config, forkNames map[string]struct{}) {
	if cfg == nil || cfg.Fork == nil {
		return
	}
	for name, height := range cfg.Fork.System {
		if !strings.HasPrefix(name, ForkAccountBlacklist) || height == -1 {
			continue
		}
		if _, ok := forkNames[name]; !ok {
			panic(fmt.Sprintf("fork %s is enabled at height %d but section [%s%s] is missing",
				name, height, blacklistMverPrefix, name))
		}
	}
}

// buildBlacklistVersions 生成按高度升序的版本序列，首个元素恒为高度 0 的 base 版本。
// 各版本的名单直接向 mver 按高度取值，取舍规则与运行期 MG 完全一致（含同高度多分叉的取舍）。
func buildBlacklistVersions(mver *mversion, forks *Forks, forkNames map[string]struct{}) []*blacklistVersion {
	base := &blacklistVersion{height: 0, set: blockedAccountSet{}}
	if mver != nil {
		base.set = parseBlockedAccounts(blacklistBaseKey, parseStrList(mver.data[blacklistBaseKey]))
	}
	if len(base.set) > 0 {
		// base 段不受任何分叉门控，自创世高度即生效。已上线的链这样配会改变历史区块的执行结果，
		// 迁移存量名单必须落到 [mver.blacklist.<分叉名>]，只有全新链才适合直接写 base 段
		tlog.Warn("accountBlacklist base section takes effect from genesis, it is NOT fork gated",
			"key", blacklistBaseKey, "size", len(base.set))
	}
	byHeight := map[int64]*blacklistVersion{0: base}
	for fork := range forkNames {
		height := forks.GetFork(fork)
		// 分叉高度为负等价于创世即生效，归一到 0 与 mver 的 height >= forkHeight 判定保持一致
		if height < 0 {
			height = 0
		}
		old, exist := byHeight[height]
		// 同高度并列时保留字母序更大的分叉名，与 versionList.addItem 的取舍一致；
		// base 版本名为空串，任何分叉都排在它之后，故分叉版本胜出
		if exist && strings.Compare(fork, old.name) <= 0 {
			tlog.Warn("accountBlacklist same fork height", "height", height, "keep", old.name, "drop", fork)
			continue
		}
		key := blacklistMverPrefix + fork + "." + blacklistMverKey
		byHeight[height] = &blacklistVersion{
			height: height,
			name:   fork,
			set:    parseBlockedAccounts(key, mverStrList(mver, blacklistBaseKey, height)),
		}
	}

	versions := make([]*blacklistVersion, 0, len(byHeight))
	for _, v := range byHeight {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i].height < versions[j].height })
	return mergeHardcodedBlacklist(versions, forks.GetFork(ForkAccountBlacklist))
}

// mverStrList 按高度取 mver 字符串列表。
// 这里不能走 Chain33Config.MG：构建发生在 chain33CfgInit 内部，其已持有 c.mu。
func mverStrList(mver *mversion, key string, height int64) []string {
	if mver == nil {
		return nil
	}
	data, err := mver.Get(key, height)
	if err != nil {
		return nil
	}
	return parseStrList(data)
}

// mergeHardcodedBlacklist 把硬编码兜底名单并入高度不小于 gate 的所有版本。
// gate 处没有版本边界时补一个，否则兜底名单会在稀疏配置下静默失效。
func mergeHardcodedBlacklist(versions []*blacklistVersion, gate int64) []*blacklistVersion {
	hardcoded := parseBlockedAccounts("blockedAccounts", blockedAccounts)
	if len(hardcoded) == 0 || gate >= MaxHeight {
		return versions
	}
	if gate < 0 {
		gate = 0
	}
	if !hasBlacklistBoundary(versions, gate) {
		src := lastVersionNotAfter(versions, gate)
		versions = append(versions, &blacklistVersion{height: gate, name: src.name, set: src.set.clone()})
		sort.Slice(versions, func(i, j int) bool { return versions[i].height < versions[j].height })
	}
	for _, v := range versions {
		if v.height >= gate {
			v.set.merge(hardcoded)
		}
	}
	return versions
}

func hasBlacklistBoundary(versions []*blacklistVersion, height int64) bool {
	for _, v := range versions {
		if v.height == height {
			return true
		}
	}
	return false
}

// lastVersionNotAfter 取高度不大于 height 的最后一个版本，versions 需按高度升序且首元素高度为 0
func lastVersionNotAfter(versions []*blacklistVersion, height int64) *blacklistVersion {
	for i := len(versions) - 1; i > 0; i-- {
		if height >= versions[i].height {
			return versions[i]
		}
	}
	return versions[0]
}

// parseBlockedAccounts 将名单统一解析为 20 字节原始地址集合。
// 任一地址解析失败直接 panic，防止错别字地址在上线时静默漏放。
func parseBlockedAccounts(source string, addrs []string) blockedAccountSet {
	set := make(blockedAccountSet, len(addrs))
	for _, addr := range addrs {
		raw, err := parseBlockedAccount(addr)
		if err != nil {
			panic(fmt.Sprintf("invalid blocked account address %s in %s: %v", addr, source, err))
		}
		if len(raw) != 20 {
			panic(fmt.Sprintf("invalid blocked account address %s in %s: raw length %d != 20", addr, source, len(raw)))
		}
		var key [20]byte
		copy(key[:], raw)
		set[key] = struct{}{}
	}
	return set
}

// parseBlockedAccount 解析单个地址：0x 十六进制地址按 hex 解码取 20 字节，
// 其余按 base58 比特币格式地址解析取 hash160。
func parseBlockedAccount(addr string) ([]byte, error) {
	if address.IsEthAddress(addr) {
		return common.FromHex(addr)
	}
	btcAddr, err := address.NewBtcAddress(addr)
	if err != nil {
		return nil, err
	}
	return btcAddr.Hash160[:], nil
}

// initAccountBlacklist 构建黑名单快照，需在分叉高度与 mver 全部就绪后调用。
// 调用方 chain33CfgInit 已持有 c.mu，这里不再加锁。
func (c *Chain33Config) initAccountBlacklist(cfg *Config) {
	c.blacklist.Store(newAccountBlacklist(cfg, c.forks, c.mver, c.needSetForkZero()))
}

// blacklistAt 返回高度 h 生效的黑名单版本，配置为空或快照尚未构建时返回 nil
func (c *Chain33Config) blacklistAt(height int64) *blacklistVersion {
	if c == nil {
		return nil
	}
	blacklist := c.blacklist.Load()
	if blacklist == nil {
		return nil
	}
	return blacklist.at(height)
}

// SetBlockedAccountsForTest 仅供测试注入黑名单：addrs 自 fromHeight 起生效，返回恢复原快照的函数。
// 生产代码请勿调用。
func (c *Chain33Config) SetBlockedAccountsForTest(fromHeight int64, addrs []string) func() {
	old := c.blacklist.Load()
	version := &blacklistVersion{height: fromHeight, name: "test", set: parseBlockedAccounts("test", addrs)}
	versions := []*blacklistVersion{version}
	if fromHeight > 0 {
		// 补一个空的 base 版本，fromHeight 之前必须放行
		versions = []*blacklistVersion{{height: 0, set: blockedAccountSet{}}, version}
	} else {
		version.height = 0
	}
	c.blacklist.Store(&accountBlacklist{versions: versions})
	return func() {
		c.blacklist.Store(old)
	}
}

// IsBlockedAccount 判断地址（base58 或 0x 形态）在高度 h 是否命中黑名单。
// 地址无法解析时视为未命中，返回 false。
func (c *Chain33Config) IsBlockedAccount(addr string, height int64) bool {
	return isBlockedAccount(c.blacklistAt(height), addr)
}

// IsBlockedAccountRaw 判断 20 字节原始地址在高度 h 是否命中黑名单
func (c *Chain33Config) IsBlockedAccountRaw(raw []byte, height int64) bool {
	version := c.blacklistAt(height)
	return version != nil && version.set.hasRaw(raw)
}

func isBlockedAccount(version *blacklistVersion, addr string) bool {
	if version == nil || len(version.set) == 0 {
		return false
	}
	raw, err := parseBlockedAccount(addr)
	if err != nil {
		return false
	}
	return version.set.hasRaw(raw)
}

// CheckTxBlockedAccount 共识层拦截，供 executor.checkTx / checkTxGroup、BaseClient.AddTxsToBlock 调用。
// 取高度 h 生效的名单做深度判定，覆盖四个维度：发送方（tx.From）、接收方（tx.To / GetRealToAddr）、
// EVM 合约目标地址（ContractAddr）、EVM 纯转账原始地址（20 字节 Para）。
// 命中返回包装后的 ErrBlockedAccount，调用方可用 errors.Is 判定。
// 这里没有 fork 门控分支：门控已在构建快照时固化为版本高度。
func CheckTxBlockedAccount(cfg *Chain33Config, height int64, tx *Transaction) error {
	return checkTxBlockedAccount(cfg.blacklistAt(height), height, tx)
}

// CheckTxsBlockedAccount 交易组共识层拦截：任一笔命中返回 error，全部通过返回 nil。
// 语义与 procExecTxList 的错误处理一致（组内任一笔 err 则整组每笔都 ExecErr）。
func CheckTxsBlockedAccount(cfg *Chain33Config, height int64, txs []*Transaction) error {
	version := cfg.blacklistAt(height)
	for _, tx := range txs {
		if err := checkTxBlockedAccount(version, height, tx); err != nil {
			return err
		}
	}
	return nil
}

// CheckTxBlockedAccountImmediate 入口层拦截，供 mempool.checkTx 与延时交易入口调用。
// 与共识层共用同一套按高度选版逻辑，height 传即将打包的下一区块高度，
// 保证 mempool 与共识判定一致；不做「提前按未来版本拦截」的旁路。
// 链是去中心化的，本机 mempool 提前过滤挡不住未升级节点出块，
// 真正的网络级拦截点只能是共识在已到达高度上的判定。
func CheckTxBlockedAccountImmediate(cfg *Chain33Config, height int64, tx *Transaction) error {
	return CheckTxBlockedAccount(cfg, height, tx)
}

// CheckTxsBlockedAccountImmediate 交易组入口层拦截，语义同 CheckTxBlockedAccountImmediate
func CheckTxsBlockedAccountImmediate(cfg *Chain33Config, height int64, txs []*Transaction) error {
	return CheckTxsBlockedAccount(cfg, height, txs)
}

// checkTxBlockedAccount 黑名单深度判定核心逻辑，唯一实现完整四维判定。
// 用哪份名单完全由传入的 version 决定，height 仅用于日志定位。
func checkTxBlockedAccount(version *blacklistVersion, height int64, tx *Transaction) error {
	if version == nil || len(version.set) == 0 || tx == nil {
		return nil
	}
	hit := func(pos, addr string) error {
		tlog.Error("CheckTxBlockedAccount hit", "txhash", common.ToHex(tx.Hash()), "height", height,
			"fork", version.name, "forkHeight", version.height, "pos", pos, "addr", addr)
		return fmt.Errorf("%w: %s %s", ErrBlockedAccount, pos, addr)
	}
	if from := tx.From(); isBlockedAccount(version, from) {
		return hit("from", from)
	}
	if to := tx.GetTo(); isBlockedAccount(version, to) {
		return hit("to", to)
	}
	if realTo := tx.GetRealToAddr(); realTo != tx.GetTo() && isBlockedAccount(version, realTo) {
		return hit("realTo", realTo)
	}
	return checkEVMTxBlockedTarget(version, tx, hit)
}

// checkEVMTxBlockedTarget 解析 EVM 交易 payload 中的真实目标地址。
// 平行链及 ETH 风格 coins 转账场景下 tx.To 可能被改写为执行器地址，
// 真实接收方只能从 EVMContractAction4Chain33 中取得：
// 合约调用取 ContractAddr，纯转账取 20 字节的 Para 原始地址。
// len(Para) == 20 的判据是概率论证而非绝对安全：ABI calldata 极少恰好 20 字节，
// 即便 isTransferNote 的备注误撞，概率约 N/2^160，可忽略。
func checkEVMTxBlockedTarget(version *blacklistVersion, tx *Transaction, hit func(pos, addr string) error) error {
	if string(GetRealExecName(tx.GetExecer())) != evmExecName {
		return nil
	}
	action := new(EVMContractAction4Chain33)
	if err := Decode(tx.GetPayload(), action); err != nil {
		return nil
	}
	if contractAddr := action.GetContractAddr(); contractAddr != "" && isBlockedAccount(version, contractAddr) {
		return hit("evmContractAddr", contractAddr)
	}
	if para := action.GetPara(); version.set.hasRaw(para) {
		return hit("evmPara", common.ToHex(para))
	}
	return nil
}
