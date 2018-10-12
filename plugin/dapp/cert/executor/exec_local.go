package executor

import (
	"gitlab.33.cn/chain33/chain33/types"
	ct "gitlab.33.cn/chain33/chain33/plugin/dapp/cert/types"
	"gitlab.33.cn/chain33/chain33/authority"
	"fmt"
)

func (c *Cert) execLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	if !authority.IsAuthEnable {
		clog.Error("Authority is not available. Please check the authority config or authority initialize error logs.")
		return nil, types.ErrInitializeAuthority
	}

	set, err := c.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	return set, nil
}

func (c *Cert) ExecLocal_New(payload *ct.CertNew, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := c.execLocal(tx, receiptData, index)
	if err != nil {
		return nil, err
	}

	historityCertdata := &types.HistoryCertStore{}
	authority.Author.HistoryCertCache.CurHeight = c.GetHeight()
	authority.Author.HistoryCertCache.ToHistoryCertStore(historityCertdata)
	key := fmt.Sprintf("cert_%d", c.GetHeight())
	set.KV = append(set.KV, &types.KeyValue{[]byte(key), types.Encode(historityCertdata)})

	// 构造非证书历史数据
	noneCertdata := &types.HistoryCertStore{}
	noneCertdata.NxtHeight = historityCertdata.CurHeigth
	noneCertdata.CurHeigth = 0
	set.KV = append(set.KV, &types.KeyValue{[]byte("cert_0"), types.Encode(noneCertdata)})

	return set, nil
}

func (c *Cert) ExecLocal_Update(payload *ct.CertUpdate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := c.execLocal(tx, receiptData, index)
	if err != nil {
		return nil, err
	}

	// 写入上一纪录的next-height
	key := []byte(fmt.Sprintf("cert_%d", authority.Author.HistoryCertCache.CurHeight))
	historityCertdata := &types.HistoryCertStore{}
	authority.Author.HistoryCertCache.NxtHeight = c.GetHeight()
	authority.Author.HistoryCertCache.ToHistoryCertStore(historityCertdata)
	set.KV = append(set.KV, &types.KeyValue{key, types.Encode(historityCertdata)})

	// 证书更新
	historityCertdata = &types.HistoryCertStore{}
	authority.Author.ReloadCertByHeght(c.GetHeight())
	authority.Author.HistoryCertCache.ToHistoryCertStore(historityCertdata)
	setKey := fmt.Sprintf("cert_%d", c.GetHeight())
	set.KV = append(set.KV, &types.KeyValue{[]byte(setKey), types.Encode(historityCertdata)})
	return set, nil
}

func (c *Cert) ExecLocal_Normal(payload *ct.CertNormal, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return c.execLocal(tx, receiptData, index)
}
