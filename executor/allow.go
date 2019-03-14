// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"bytes"

	"github.com/pkg/errors"

	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
)

func isAllowKeyWrite(e *executor, key, realExecer []byte, tx *types.Transaction, index int) bool {
	keyExecer, err := types.FindExecer(key)
	if err != nil {
		elog.Error("find execer ", "err", err, "key", string(key), "keyexecer", string(keyExecer))
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
	if !types.IsFork(e.height, "ForkExecKey") {
		if bytes.Equal(exec, []byte("manage")) && bytes.Equal(keyExecer, []byte("config")) {
			return true
		}
		if bytes.Equal(exec, []byte("token")) {
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
	c := e.loadDriver(&types.Transaction{Execer: execdriver}, index)
	//交给 -> friend 来判定
	return c.IsFriend(execdriver, key, tx)
}

func isAllowLocalKey(execer []byte, key []byte) error {
	err := isAllowLocalKey2(execer, key)
	if err != nil {
		realexec := types.GetRealExecName(execer)
		if !bytes.Equal(realexec, execer) {
			err2 := isAllowLocalKey2(realexec, key)
			err = errors.Wrapf(err2, "1st check err: %s. 2nd check err", err.Error())
		}
		if err != nil {
			elog.Error("isAllowLocalKey failed", "err", err.Error())
			return errors.Cause(err)
		}

	}
	return nil
}

func isAllowLocalKey2(execer []byte, key []byte) error {
	if len(execer) < 1 {
		return errors.Wrap(types.ErrLocalPrefix, "execer empty")
	}
	minkeylen := len(types.LocalPrefix) + len(execer) + 2
	if len(key) <= minkeylen {
		err := errors.Wrapf(types.ErrLocalKeyLen, "isAllowLocalKey too short. key=%s exec=%s", string(key), string(execer))
		return err
	}
	if key[minkeylen-1] != '-' || key[len(types.LocalPrefix)] != '-' {
		err := errors.Wrapf(types.ErrLocalPrefix,
			"isAllowLocalKey prefix last char or separator is not '-'. key=%s exec=%s minkeylen=%d title=%s",
			string(key), string(execer), minkeylen, types.GetTitle())
		return err
	}
	if !bytes.HasPrefix(key, types.LocalPrefix) {
		err := errors.Wrapf(types.ErrLocalPrefix, "isAllowLocalKey common prefix not match. key=%s exec=%s",
			string(key), string(execer))
		return err
	}
	if !bytes.HasPrefix(key[len(types.LocalPrefix)+1:], execer) {
		err := errors.Wrapf(types.ErrLocalPrefix, "isAllowLocalKey key prefix not match. key=%s exec=%s",
			string(key), string(execer))
		return err
	}
	return nil
}
