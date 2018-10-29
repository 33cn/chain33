package executor

import (
	"bytes"

	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

func isAllowKeyWrite(key, realExecer []byte, tx *types.Transaction, height int64) bool {
	keyExecer, err := types.FindExecer(key)
	if err != nil {
		elog.Error("find execer ", "err", err)
		return false
	}
	//平行链中 user.p.guodun.xxxx -> 实际上是 xxxx
	//注意: user.p.guodun.user.evm.hash -> user.evm.hash 而不是 evm
	exec := types.GetParaExec(tx.Execer)
	//默认规则1: (执行器只能修改执行器自己内部的数据)
	if bytes.Equal(keyExecer, exec) {
		return true
	}
	// 历史原因做只针对对bityuan的fork特殊化处理一下
	// manage 的key 是 config
	// token 的部分key 是 mavl-create-token-
	if !types.IsMatchFork(height, types.ForkV13ExecKey) {
		if bytes.Equal(exec, types.ExecerManage) && bytes.Equal(keyExecer, types.ExecerConfig) {
			return true
		}
		if bytes.Equal(exec, types.ExecerToken) {
			if bytes.HasPrefix(key, []byte("mavl-create-token-")) {
				return true
			}
		}
	}
	//每个合约中，都会开辟一个区域，这个区域是另外一个合约可以修改的区域
	//我们把数据限制在这个位置，防止合约的其他位置被另外一个合约修改
	//  execaddr 是加了前缀生成的地址， 而参数 realExecer 是没有前缀的执行器名字
	keyExecAddr, ok := types.GetExecKey(key)
	if ok && keyExecAddr == drivers.ExecAddress(string(tx.Execer)) {
		return true
	}
	//对应上面两种写权限，调用真实的合约，进行判断:
	//执行器会判断一个合约是否可以 被另一个合约写入
	execdriver := keyExecer
	if ok && keyExecAddr == drivers.ExecAddress(string(realExecer)) {
		//判断user.p.xxx.token 是否可以写 token 合约的内容之类的
		execdriver = realExecer
	}
	d, err := drivers.LoadDriver(string(execdriver), height)
	if err != nil {
		elog.Error("load drivers error", "err", err)
		return false
	}
	//交给 -> friend 来判定
	return d.IsFriend(execdriver, key, tx)
}

func isAllowLocalKey(execer []byte, key []byte) error {
	execer = types.GetRealExecName(execer)
	//println(string(execer), string(key))
	minkeylen := len(types.LocalPrefix) + len(execer) + 2
	if len(key) <= minkeylen {
		elog.Error("isAllowLocalKey too short", "key", string(key), "exec", string(execer))
		return types.ErrLocalKeyLen
	}
	if key[minkeylen-1] != '-' {
		elog.Error("isAllowLocalKey prefix last char is not '-'", "key", string(key), "exec", string(execer),
			"minkeylen", minkeylen, )
		return types.ErrLocalPrefix
	}
	if !bytes.HasPrefix(key, types.LocalPrefix) {
		elog.Error("isAllowLocalKey common prefix not match", "key", string(key), "exec", string(execer))
		return types.ErrLocalPrefix
	}
	if !bytes.HasPrefix(key[len(types.LocalPrefix)+1:], execer) {
		elog.Error("isAllowLocalKey key prefix not match", "key", string(key), "exec", string(execer))
		return types.ErrLocalPrefix
	}
	return nil
}
