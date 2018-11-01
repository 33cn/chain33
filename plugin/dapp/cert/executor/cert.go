package executor

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/cert/authority"
	ct "gitlab.33.cn/chain33/chain33/plugin/dapp/cert/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.cert")
var driverName = ct.CertX

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Cert{}))
}

func Init(name string, sub []byte) {
	driverName = name
	var cfg ct.Authority
	if sub != nil {
		types.MustDecode(sub, &cfg)
	}
	authority.Author.Init(&cfg)
	drivers.Register(driverName, newCert, types.GetDappFork(driverName, "Enable"))
}

func GetName() string {
	return newCert().GetName()
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

func (c *Cert) GetDriverName() string {
	return driverName
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
		return ct.ErrInitializeAuthority
	}

	// 重启
	if authority.Author.HistoryCertCache.CurHeight == -1 {
		c.loadHistoryByPrefix()
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
根据前缀查找证书变更记录，cert回滚、重启、同步用到
*/
func (c *Cert) loadHistoryByPrefix() error {
	parm := &types.LocalDBList{[]byte("LODB-cert-"), nil, 0, 0}
	result, err := c.DriverBase.GetApi().LocalList(parm)
	if err != nil {
		return err
	}

	// 数据库没有变更记录，说明创世区块开始cert校验
	if len(result.Values) == 0 {
		authority.Author.HistoryCertCache.CurHeight = 0
		return nil
	}

	// 寻找当前高度使用的证书区间
	var historyData types.HistoryCertStore
	for _, v := range result.Values {
		types.Decode(v, &historyData)
		if historyData.CurHeigth < c.GetHeight() && (historyData.NxtHeight >= c.GetHeight() || historyData.NxtHeight == -1) {
			return authority.Author.ReloadCert(&historyData)
		}
	}

	return ct.ErrGetHistoryCertData
}

/**
根据具体高度查找变更记录，cert回滚用到
*/
func (c *Cert) loadHistoryByHeight() error {
	key := calcCertHeightKey(c.GetHeight())
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
	return ct.ErrGetHistoryCertData
}
