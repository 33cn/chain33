package executor

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/authority"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	ct "gitlab.33.cn/chain33/chain33/plugin/dapp/cert/types"
	"gitlab.33.cn/chain33/chain33/types"
	"reflect"
)

var clog = log.New("module", "execs.cert")
var driverName = "cert"

//初始化过程比较重量级，有很多reflact, 所以弄成全局的
var executorFunList = make(map[string]reflect.Method)
var executorType = ct.NewType()

func init() {
	actionFunList := executorType.GetFuncMap()
	executorFunList = types.ListMethod(&Cert{})
	for k, v := range actionFunList {
		executorFunList[k] = v
	}
}

func Init(name string) {
	driverName = name
	drivers.Register(driverName, newCert, 0)
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
	c.SetExecutorType(executorType)

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
		return types.ErrInitializeAuthority
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

	// 寻找当前高度使用的证书区间
	var historyData types.HistoryCertStore
	for _, v := range result.Values {
		types.Decode(v, &historyData)
		if historyData.CurHeigth < c.GetHeight() && (historyData.NxtHeight >= c.GetHeight() || historyData.NxtHeight == -1) {
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

func (c *Cert) GetFuncMap() map[string]reflect.Method {
	return executorFunList
}