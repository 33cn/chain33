package dapp

import (
	"bytes"
	"strings"

	"gitlab.33.cn/chain33/chain33/types"
)

func (d *DriverBase) AllowIsSame(execer []byte) bool {
	return d.child.GetDriverName() == string(execer)
}

func (d *DriverBase) GetPara(execer []byte) ([]byte, bool) {
	//必须是平行链
	if !types.IsPara() {
		return nil, false
	}
	//必须是相同的平行链
	if !strings.HasPrefix(string(execer), types.GetTitle()) {
		return nil, false
	}
	return execer[len(types.GetTitle()):], true
}

func (d *DriverBase) AllowIsSamePara(execer []byte) bool {
	exec, ok := d.GetPara(execer)
	if !ok {
		return false
	}
	return d.AllowIsSame(exec)
}

func (d *DriverBase) AllowIsUserDot1Para(execer []byte) bool {
	exec, ok := d.GetPara(execer)
	if !ok {
		return false
	}
	return d.AllowIsUserDot1(exec)
}

func (d *DriverBase) AllowIsUserDot2Para(execer []byte) bool {
	exec, ok := d.GetPara(execer)
	if !ok {
		return false
	}
	return d.AllowIsUserDot2(exec)
}

//user.evm
func (d *DriverBase) AllowIsUserDot1(execer []byte) bool {
	if !bytes.HasPrefix(execer, types.UserKey) {
		return false
	}
	return d.AllowIsSame(execer[len(types.UserKey):])
}

//user.evm.xxx
func (d *DriverBase) AllowIsUserDot2(execer []byte) bool {
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
	//主链: 名字相同
	if !types.IsPara() && d.AllowIsSame(tx.Execer) {
		return nil
	}
	//平行链: 除掉title, 名字相同
	if types.IsPara() && d.AllowIsSamePara(tx.Execer) {
		return nil
	}
	return types.ErrNotAllow
}

func (d *DriverBase) IsFriend(myexec, writekey []byte, othertx *types.Transaction) bool {
	return false
}
