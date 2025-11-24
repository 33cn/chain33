// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/system/dapp"
	mty "github.com/33cn/chain33/system/dapp/manage/types"
	"github.com/33cn/chain33/types"
)

// ExecLocal_Apply local apply
func (c *Manage) ExecLocal_Apply(payload *mty.ApplyConfig, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return c.execAutoLocalItem(tx, receiptData)
}

// ExecLocal_Approve local approve
func (c *Manage) ExecLocal_Approve(payload *mty.ApproveConfig, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return c.execAutoLocalItem(tx, receiptData)
}

func (c *Manage) execAutoLocalItem(tx *types.Transaction, receiptData *types.ReceiptData) (*types.LocalDBSet, error) {
	set, err := c.execLocalItem(receiptData)
	if err != nil {
		return set, err
	}
	dbSet := &types.LocalDBSet{}
	dbSet.KV = c.AddRollbackKV(tx, tx.Execer, set.KV)
	return dbSet, nil
}

func (c *Manage) execLocalItem(receiptData *types.ReceiptData) (*types.LocalDBSet, error) {
	table := NewConfigTable(c.GetLocalDB())
	for _, log := range receiptData.Logs {
		switch log.Ty {
		case mty.TyLogApplyConfig:
			{
				var receipt mty.ReceiptApplyConfig
				err := types.Decode(log.Log, &receipt)
				if err != nil {
					return nil, err
				}
				err = table.Replace(receipt.Status)
				if err != nil {
					return nil, err
				}
			}
		case mty.TyLogApproveConfig:
			{
				var receipt mty.ReceiptApproveConfig
				err := types.Decode(log.Log, &receipt)
				if err != nil {
					return nil, err
				}
				err = table.Replace(receipt.GetCur())
				if err != nil {
					return nil, err
				}
			}
		default:
			break
		}
	}
	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	dbSet := &types.LocalDBSet{}
	dbSet.KV = append(dbSet.KV, kvs...)
	return dbSet, nil
}

func (c *Manage) listProposalItem(req *mty.ReqQueryConfigList) (types.Message, error) {
	if req == nil {
		return nil, types.ErrInvalidParam
	}
	localDb := c.GetLocalDB()
	query := NewConfigTable(localDb).GetQuery(localDb)
	var primary []byte
	if req.Height > 0 {
		primary = []byte(dapp.HeightIndexStr(req.Height, int64(req.Index)))
	}
	indexName := ""
	if req.Status > 0 && len(req.Proposer) > 0 {
		indexName = "addr_status"
	} else if req.Status > 0 {
		indexName = "status"
	} else if len(req.Proposer) > 0 {
		indexName = "addr"
	}

	cur := &ConfigRow{
		ConfigStatus: &mty.ConfigStatus{},
	}
	cur.Proposer = req.Proposer
	cur.Status = req.Status
	cur.Height = req.Height
	cur.Index = req.Index
	prefix, err := cur.Get(indexName)
	if err != nil {
		clog.Error("Get", "indexName", indexName, "err", err)
		return nil, err
	}

	rows, err := query.ListIndex(indexName, prefix, primary, req.Count, req.Direction)
	if err != nil {
		clog.Error("query List failed", "indexName", indexName, "prefix", "prefix", "key", string(primary), "err", err)
		return nil, err
	}
	if len(rows) == 0 {
		return nil, types.ErrNotFound
	}

	var rep mty.ReplyQueryConfigList
	for _, row := range rows {
		r, ok := row.Data.(*mty.ConfigStatus)
		if !ok {
			clog.Error("listConfigStatus", "err", "bad row type")
			return nil, types.ErrDecode
		}
		rep.Lists = append(rep.Lists, r)
	}
	return &rep, nil
}
