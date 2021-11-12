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
	types.AssertConfig(c.GetAPI())
	cfg := c.GetAPI().GetConfig()

	if cfg.IsDappFork(c.GetHeight(), mty.ManageX, mty.ForkManageAutonomyApprove) {
		return nil, types.ErrNotAllow
	}

	if cfg.IsDappFork(c.GetHeight(), mty.ManageX, mty.ForkManageExec) {
		if err := c.checkTxToAddress(tx, index); err != nil {
			return nil, err
		}
	}
	action := newAction(c, tx, int32(index))
	if !IsSuperManager(cfg, action.fromaddr) {
		return nil, mty.ErrNoPrivilege
	}
	return action.modifyConfig(manageAction)

}

func (c *Manage) Exec_ApplyConfig(payload *mty.ApplyConfig, tx *types.Transaction, index int) (*types.Receipt, error) {
	cfg := c.GetAPI().GetConfig()
	if !cfg.IsDappFork(c.GetHeight(), mty.ManageX, mty.ForkManageAutonomyApprove) {
		return nil, types.ErrNotAllow
	}

	action := newAction(c, tx, int32(index))
	return action.applyConfig(payload)
}

func (c *Manage) Exec_ApproveConfig(payload *mty.ApproveConfig, tx *types.Transaction, index int) (*types.Receipt, error) {
	cfg := c.GetAPI().GetConfig()
	if !cfg.IsDappFork(c.GetHeight(), mty.ManageX, mty.ForkManageAutonomyApprove) {
		return nil, types.ErrNotAllow
	}

	action := newAction(c, tx, int32(index))
	return action.approveConfig(payload)
}
