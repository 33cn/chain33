package cert

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/authority"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.cert")

func Init() {
	drivers.Register(newCert().GetName(), newCert, 0)
}

type Cert struct {
	drivers.DriverBase
}

func newCert() drivers.Driver {
	c := &Cert{}
	c.SetChild(c)
	c.SetIsFree(true)

	return c
}

func (c *Cert) GetName() string {
	return "cert"
}

func (c *Cert) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	if !authority.IsAuthEnable {
		clog.Error("Authority is not available. Please check the authority config or authority initialize error logs.")
		return nil, types.ErrInitializeAuthority
	}

	var action types.CertAction
	err := types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
	var set types.LocalDBSet
	if action.Ty == types.CertActionNew {
		//证书启用
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
	} else if action.Ty == types.CertActionUpdate {
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
	} else if action.Ty == types.CertActionNormal {
		return c.DriverBase.ExecLocal(tx, receipt, index)
	} else {
		return nil, types.ErrActionNotSupport
	}

	return &set, nil
}

func (c *Cert) Query(funcname string, params []byte) (types.Message, error) {
	return nil, types.ErrActionNotSupport
}

func (c *Cert) CheckTx(tx *types.Transaction, index int) error {
	// 基类检查
	err := c.DriverBase.CheckTx(tx, index)
	if err != nil {
		return err
	}

	// auth模块关闭则返回
	if !authority.IsAuthEnable {
		clog.Error("Authority is not available. Please check the authority config or authority initialize error logs.")
		return types.ErrInitializeAuthority
	}

	// 重启
	if authority.Author.HistoryCertCache.CurHeight == -1 {
		c.initHistoryByPrefix()
	}

	// 当前区块<上次证书变更区块，cert回滚
	if c.GetHeight() <= authority.Author.HistoryCertCache.CurHeight {
		c.loadHistoryByPrefix()
	}

	// 当前区块>上次变更下一区块，下一区块不为-1，即非最新证书变更记录，用于cert回滚时判断是否到了下一变更记录
	nxtHeight := authority.Author.HistoryCertCache.NxtHeight
	if nxtHeight != -1 && c.GetHeight() > nxtHeight {
		c.loadHistoryByHeight()
	}

	// auth校验
	return authority.Author.Validate(tx.GetSignature())
}

/**
系统重启，初始化auth模块的curHeight
*/
func (c *Cert) initHistoryByPrefix() error {
	parm := &types.LocalDBList{[]byte("cert_"), nil, 0, 0}
	result, err := c.DriverBase.GetApi().LocalList(parm)
	if err != nil {
		return err
	}

	// 数据库没有变更记录，说明创世区块开始cert校验
	if len(result.Values) == 0 {
		authority.Author.HistoryCertCache.CurHeight = 0
		return nil
	}

	// 数据库有变更记录，nxt = -1为最近一次变更
	var historyData types.HistoryCertStore
	for _, v := range result.Values {
		types.Decode(v, &historyData)
		if historyData.NxtHeight == -1 {
			authority.Author.HistoryCertCache.CurHeight = historyData.CurHeigth
			return nil
		}
	}

	return types.ErrGetHistoryCertData
}

/**
根据前缀查找证书变更记录，cert回滚用到
*/
func (c *Cert) loadHistoryByPrefix() error {
	parm := &types.LocalDBList{[]byte("cert_"), nil, 0, 0}
	result, err := c.DriverBase.GetApi().LocalList(parm)
	if err != nil {
		return err
	}

	var historyData types.HistoryCertStore
	for _, v := range result.Values {
		types.Decode(v, &historyData)
		if historyData.CurHeigth < c.GetHeight() && historyData.NxtHeight >= c.GetHeight() {
			return authority.Author.ReloadCert(&historyData)
		}
	}

	return types.ErrGetHistoryCertData
}

/**
根据具体高度查找变更记录，cert回滚用到
*/
func (c *Cert) loadHistoryByHeight() error {
	key := []byte(fmt.Sprintf("cert_%s", c.GetHeight()))
	parm := &types.LocalDBGet{[][]byte{key}}
	result, err := c.DriverBase.GetApi().LocalGet(parm)
	if err != nil {
		return err
	}

	var historyData types.HistoryCertStore
	for _, v := range result.Values {
		types.Decode(v, &historyData)
		if historyData.CurHeigth < c.GetHeight() && historyData.NxtHeight >= c.GetHeight() {
			return authority.Author.ReloadCert(&historyData)
		}
	}

	return types.ErrGetHistoryCertData
}
