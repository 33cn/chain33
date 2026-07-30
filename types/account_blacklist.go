// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
)

// ForkAccountBlacklist 账户黑名单分叉名，配置在 [fork.system] 节。
// 默认注册高度为 MaxHeight（永不启用），配置为具体高度 H 后，
// 高度 >= H 的区块中涉及黑名单地址的交易将被共识层判为非法。
const ForkAccountBlacklist = "ForkAccountBlacklist"

// evmExecName EVM 执行器真实名，用于从 payload 中解析真实目标地址
const evmExecName = "evm"

// blockedAccountSet 黑名单原始 20 字节地址集合，由 toml [blacklist] 段的
// accountBlacklist 在 Chain33Config 初始化时解析生成，未配置则为空。
// 以原始 20 字节为 key，天然兼容 base58 与 0x 两种地址形态。
var blockedAccountSet map[[20]byte]struct{}

// SetBlockedAccountsForTest 仅供测试注入黑名单，返回恢复原集合的函数。
// 生产代码请勿调用。
func SetBlockedAccountsForTest(addrs []string) func() {
	old := blockedAccountSet
	blockedAccountSet = parseBlockedAccounts(addrs)
	return func() {
		blockedAccountSet = old
	}
}

// parseBlockedAccounts 将名单统一解析为 20 字节原始地址集合。
// 任一地址解析失败直接 panic，防止错别字地址在上线时静默漏放。
func parseBlockedAccounts(addrs []string) map[[20]byte]struct{} {
	set := make(map[[20]byte]struct{}, len(addrs))
	for _, addr := range addrs {
		raw, err := parseBlockedAccount(addr)
		if err != nil {
			panic(fmt.Sprintf("invalid blocked account address %s: %v", addr, err))
		}
		if len(raw) != 20 {
			panic(fmt.Sprintf("invalid blocked account address %s: raw length %d != 20", addr, len(raw)))
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

// IsBlockedAccountRaw 判断 20 字节原始地址是否命中黑名单
func IsBlockedAccountRaw(raw []byte) bool {
	if len(raw) != 20 || len(blockedAccountSet) == 0 {
		return false
	}
	var key [20]byte
	copy(key[:], raw)
	_, ok := blockedAccountSet[key]
	return ok
}

// IsBlockedAccount 判断地址（base58 或 0x 形态）是否命中黑名单。
// 地址无法解析时视为未命中，返回 false。
func IsBlockedAccount(addr string) bool {
	if len(blockedAccountSet) == 0 {
		return false
	}
	raw, err := parseBlockedAccount(addr)
	if err != nil {
		return false
	}
	return IsBlockedAccountRaw(raw)
}

// CheckTxBlockedAccount 共识层拦截（fork 门控 + 深度判定），
// 供 executor.checkTx / checkTxGroup、BaseClient.AddTxsToBlock 调用。
// ForkAccountBlacklist 高度后检查交易是否涉及黑名单地址，覆盖四个维度：
// 发送方（tx.From）、接收方（tx.To / GetRealToAddr）、
// EVM 合约目标地址（ContractAddr）、EVM 纯转账原始地址（20 字节 Para）。
// 命中返回包装后的 ErrBlockedAccount，调用方可用 errors.Is 判定。
func CheckTxBlockedAccount(cfg *Chain33Config, height int64, tx *Transaction) error {
	if cfg == nil || !cfg.IsFork(height, ForkAccountBlacklist) {
		return nil
	}
	return checkTxBlockedAccountCore(tx, height)
}

// CheckTxBlockedAccountImmediate 入口层拦截（不带 fork 门控 + 深度判定），
// 供 mempool.checkTx / checkTxs 与延时交易 eventAddDelayTx / addDelayTx 调用。
// mempool 是节点本地行为、不进状态计算、无共识分叉风险，随二进制升级立即生效，提前止血。
func CheckTxBlockedAccountImmediate(tx *Transaction) error {
	return checkTxBlockedAccountCore(tx, 0)
}

// CheckTxsBlockedAccount 交易组共识层拦截：任一笔命中返回 error，全部通过返回 nil。
// 语义与 procExecTxList 的错误处理一致（组内任一笔 err 则整组每笔都 ExecErr）。
func CheckTxsBlockedAccount(cfg *Chain33Config, height int64, txs []*Transaction) error {
	for _, tx := range txs {
		if err := CheckTxBlockedAccount(cfg, height, tx); err != nil {
			return err
		}
	}
	return nil
}

// CheckTxsBlockedAccountImmediate 交易组入口层拦截（不带 fork 门控）
func CheckTxsBlockedAccountImmediate(txs []*Transaction) error {
	for _, tx := range txs {
		if err := CheckTxBlockedAccountImmediate(tx); err != nil {
			return err
		}
	}
	return nil
}

// checkTxBlockedAccountCore 黑名单深度判定核心逻辑，唯一实现完整四维判定。
// logHeight 仅用于日志，不做 fork 判断；Immediate 入口传 0。
func checkTxBlockedAccountCore(tx *Transaction, logHeight int64) error {
	if len(blockedAccountSet) == 0 || tx == nil {
		return nil
	}
	txhash := common.ToHex(tx.Hash())
	if from := tx.From(); IsBlockedAccount(from) {
		tlog.Error("CheckTxBlockedAccount hit", "txhash", txhash, "height", logHeight, "pos", "from", "addr", from)
		return fmt.Errorf("%w: from %s", ErrBlockedAccount, from)
	}
	if to := tx.GetTo(); IsBlockedAccount(to) {
		tlog.Error("CheckTxBlockedAccount hit", "txhash", txhash, "height", logHeight, "pos", "to", "addr", to)
		return fmt.Errorf("%w: to %s", ErrBlockedAccount, to)
	}
	if realTo := tx.GetRealToAddr(); realTo != tx.GetTo() && IsBlockedAccount(realTo) {
		tlog.Error("CheckTxBlockedAccount hit", "txhash", txhash, "height", logHeight, "pos", "realTo", "addr", realTo)
		return fmt.Errorf("%w: real to %s", ErrBlockedAccount, realTo)
	}
	if err := checkEVMTxBlockedTarget(tx, logHeight, txhash); err != nil {
		return err
	}
	return nil
}

// checkEVMTxBlockedTarget 解析 EVM 交易 payload 中的真实目标地址。
// 平行链及 ETH 风格 coins 转账场景下 tx.To 可能被改写为执行器地址，
// 真实接收方只能从 EVMContractAction4Chain33 中取得：
// 合约调用取 ContractAddr，纯转账取 20 字节的 Para 原始地址。
// len(Para) == 20 的判据是概率论证而非绝对安全：ABI calldata 极少恰好 20 字节，
// 即便 isTransferNote 的备注误撞，概率约 N/2^160，可忽略。
func checkEVMTxBlockedTarget(tx *Transaction, logHeight int64, txhash string) error {
	if string(GetRealExecName(tx.GetExecer())) != evmExecName {
		return nil
	}
	action := new(EVMContractAction4Chain33)
	if err := Decode(tx.GetPayload(), action); err != nil {
		return nil
	}
	if contractAddr := action.GetContractAddr(); contractAddr != "" && IsBlockedAccount(contractAddr) {
		tlog.Error("CheckTxBlockedAccount hit", "txhash", txhash, "height", logHeight, "pos", "evmContractAddr", "addr", contractAddr)
		return fmt.Errorf("%w: evm contract addr %s", ErrBlockedAccount, contractAddr)
	}
	if para := action.GetPara(); IsBlockedAccountRaw(para) {
		addr := common.ToHex(para)
		tlog.Error("CheckTxBlockedAccount hit", "txhash", txhash, "height", logHeight, "pos", "evmPara", "addr", addr)
		return fmt.Errorf("%w: evm transfer to %s", ErrBlockedAccount, addr)
	}
	return nil
}
