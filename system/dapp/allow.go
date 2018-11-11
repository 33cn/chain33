package dapp

import (
	"bytes"

	"github.com/33cn/chain33/types"
)

func (d *DriverBase) AllowIsSame(execer []byte) bool {
	execer = types.GetParaExec(execer)
	return d.child.GetDriverName() == string(execer)
}

//user.evm
func (d *DriverBase) AllowIsUserDot1(execer []byte) bool {
	execer = types.GetParaExec(execer)
	if !bytes.HasPrefix(execer, types.UserKey) {
		return false
	}
	return d.AllowIsSame(execer[len(types.UserKey):])
}

//user.evm.xxx
func (d *DriverBase) AllowIsUserDot2(execer []byte) bool {
	execer = types.GetParaExec(execer)
	if !bytes.HasPrefix(execer, types.UserKey) {
		return false
	}
	count := 0
	index := 0
	s := len(types.UserKey)
	for i := s; i < len(execer); i++ {
		if execer[i] == '.' {
			count++
			index = i
		}
	}
	if count == 1 && d.AllowIsSame(execer[s:index]) {
		return true
	}
	return false
}

//默认行为: 名字相同 或者 是平行链
func (d *DriverBase) Allow(tx *types.Transaction, index int) error {
	if d.AllowIsSame(tx.Execer) {
		return nil
	}
	return types.ErrNotAllow
}

func (d *DriverBase) IsFriend(myexec, writekey []byte, othertx *types.Transaction) bool {
	return false
}
