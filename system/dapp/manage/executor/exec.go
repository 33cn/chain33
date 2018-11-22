// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/system/dapp"
	mty "github.com/33cn/chain33/system/dapp/manage/types"
	"github.com/33cn/chain33/types"
)

func (c *Manage) checkAddress(addr string) error {
	if dapp.IsDriverAddress(addr, c.GetHeight()) {
		return nil
	}
	return address.CheckAddress(addr)
}

func (c *Manage) checkTxToAddress(tx *types.Transaction, index int) error {
	return c.checkAddress(tx.GetRealToAddr())
}

// Exec_Modify modify exec
func (c *Manage) Exec_Modify(manageAction *types.ModifyConfig, tx *types.Transaction, index int) (*types.Receipt, error) {
	clog.Info("manage.Exec", "start index", index)
	// 兼容在区块上没有To地址检查的交易数据
	if types.IsDappFork(c.GetHeight(), mty.ManageX, "ForkManageExec") {
		if err := c.checkTxToAddress(tx, index); err != nil {
			return nil, err
		}
	}
	action := NewAction(c, tx)
	return action.modifyConfig(manageAction)

}
