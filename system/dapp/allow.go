// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dapp

import (
	"bytes"

	"github.com/33cn/chain33/types"
)

// AllowIsSame allow issame drivername
func (d *DriverBase) AllowIsSame(execer []byte) bool {
	execer = types.GetParaExec(execer)
	return d.child.GetDriverName() == string(execer)
}

// AllowIsUserDot1 user.evm
func (d *DriverBase) AllowIsUserDot1(execer []byte) bool {
	execer = types.GetParaExec(execer)
	if !bytes.HasPrefix(execer, types.UserKey) {
		return false
	}
	return d.AllowIsSame(execer[len(types.UserKey):])
}

// AllowIsUserDot2 user.evm.xxx
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

// Allow default behavior: same name  or  parallel chain
func (d *DriverBase) Allow(tx *types.Transaction, index int) error {
	if d.AllowIsSame(tx.Execer) {
		return nil
	}
	return types.ErrNotAllow
}

// IsFriend defines a isfriend function
func (d *DriverBase) IsFriend(myexec, writekey []byte, othertx *types.Transaction) bool {
	return false
}
