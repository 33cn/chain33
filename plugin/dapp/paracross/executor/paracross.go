package executor

import (
	"bytes"

	log "github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/common"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
)

var (
	clog                    = log.New("module", "execs.paracross")
	enableParacrossTransfer = true
	driverName              = pt.ParaX
)

type Paracross struct {
	drivers.DriverBase
}

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Paracross{}))
}

func Init(name string, sub []byte) {
	drivers.Register(GetName(), newParacross, types.GetDappFork(driverName, "Enable"))
	setPrefix()
}

func GetName() string {
	return newParacross().GetName()
}

func newParacross() drivers.Driver {
	c := &Paracross{}
	c.SetChild(c)
	c.SetExecutorType(types.LoadExecutorType(driverName))
	return c
}

func (c *Paracross) GetDriverName() string {
	return pt.ParaX
}

func (c *Paracross) checkTxGroup(tx *types.Transaction, index int) ([]*types.Transaction, error) {
	if tx.GroupCount >= 2 {
		txs, err := c.GetTxGroup(index)
		if err != nil {
			clog.Error("ParacrossActionAssetTransfer", "get tx group failed", err, "hash", common.Bytes2Hex(tx.Hash()))
			return nil, err
		}
		return txs, nil
	}
	return nil, nil
}

//经para filter之后，交易组里面只会存在主链平行链跨链交易或全部平行链交易，全部平行链交易group里面有可能有资产转移交易
func crossTxGroupProc(txs []*types.Transaction, index int) ([]*types.Transaction, int32) {
	var headIdx, endIdx int32

	for i := index; i >= 0; i-- {
		if bytes.Equal(txs[index].Header, txs[i].Hash()) {
			headIdx = int32(i)
			break
		}
	}
	//cross mix tx, contain main and para tx, main prefix with pt.ParaX
	endIdx = headIdx + txs[index].GroupCount
	for i := headIdx; i < endIdx; i++ {
		if bytes.HasPrefix(txs[i].Execer, []byte(pt.ParaX)) {
			return txs[headIdx:endIdx], endIdx
		}
	}
	//cross asset transfer in tx group
	var transfers []*types.Transaction
	for i := headIdx; i < endIdx; i++ {
		if types.IsParaExecName(string(txs[i].Execer)) &&
			bytes.HasSuffix(txs[i].Execer, []byte(pt.ParaX)) {
			transfers = append(transfers, txs[i])

		}
	}
	return transfers, endIdx

}

func (c *Paracross) saveLocalParaTxs(tx *types.Transaction, isDel bool) (*types.LocalDBSet, error) {
	var set types.LocalDBSet

	var payload pt.ParacrossAction
	err := types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}
	if payload.Ty != pt.ParacrossActionCommit || payload.GetCommit() == nil {
		return nil, nil
	}

	commit := payload.GetCommit()
	for i := 0; i < len(commit.Status.CrossTxHashs); i++ {
		success := util.BitMapBit(commit.Status.CrossTxResult, uint32(i))

		paraTx, err := GetTx(c.GetApi(), commit.Status.CrossTxHashs[i])
		if err != nil {
			clog.Crit("paracross.Commit Load Tx failed", "para title", commit.Status.Title,
				"para height", commit.Status.Height, "para tx index", i, "error", err, "txHash",
				common.Bytes2Hex(commit.Status.CrossTxHashs[i]))
			return nil, err
		}

		var payload pt.ParacrossAction
		err = types.Decode(paraTx.Tx.Payload, &payload)
		if err != nil {
			clog.Crit("paracross.Commit Decode Tx failed", "para title", commit.Status.Title,
				"para height", commit.Status.Height, "para tx index", i, "error", err, "txHash",
				common.Bytes2Hex(commit.Status.CrossTxHashs[i]))
			return nil, err
		}
		if payload.Ty == pt.ParacrossActionAssetTransfer {
			kv, err := c.updateLocalAssetTransfer(tx, paraTx.Tx, success, isDel)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, kv)
		} else if payload.Ty == pt.ParacrossActionAssetWithdraw {
			kv, err := c.initLocalAssetWithdraw(tx, paraTx.Tx, true, success, isDel)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, kv)
		}
	}

	return &set, nil
}

func getCommitHeight(payload []byte) (int64, error) {
	var a pt.ParacrossAction
	err := types.Decode(payload, &a)
	if err != nil {
		return 0, err
	}
	if a.GetCommit() == nil {
		return 0, types.ErrInvalidParam
	}

	return a.GetCommit().Status.Height, nil
}

func (c *Paracross) initLocalAssetTransfer(tx *types.Transaction, success, isDel bool) (*types.KeyValue, error) {
	clog.Debug("para execLocal", "tx hash", common.Bytes2Hex(tx.Hash()), "action name", log.Lazy{tx.ActionName})
	key := calcLocalAssetKey(tx.Hash())
	if isDel {
		c.GetLocalDB().Set(key, nil)
		return &types.KeyValue{key, nil}, nil
	}

	var payload pt.ParacrossAction
	err := types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}
	if payload.GetAssetTransfer() == nil {
		return nil, errors.New("GetAssetTransfer is nil")
	}
	exec := "coins"
	symbol := types.BTY
	if payload.GetAssetTransfer().Cointoken != "" {
		exec = "token"
		symbol = payload.GetAssetTransfer().Cointoken
	}

	var asset pt.ParacrossAsset
	amount, err := tx.Amount()
	if err != nil {
		return nil, err
	}

	asset = pt.ParacrossAsset{
		From:       tx.From(),
		To:         tx.To,
		Amount:     amount,
		IsWithdraw: false,
		TxHash:     tx.Hash(),
		Height:     c.GetHeight(),
		Exec:       exec,
		Symbol:     symbol,
	}

	err = c.GetLocalDB().Set(key, types.Encode(&asset))
	if err != nil {
		clog.Error("para execLocal", "set", common.Bytes2Hex(tx.Hash()), "failed", err)
	}
	return &types.KeyValue{key, types.Encode(&asset)}, nil
}

func (c *Paracross) initLocalAssetWithdraw(txCommit, tx *types.Transaction, isWithdraw, success, isDel bool) (*types.KeyValue, error) {
	key := calcLocalAssetKey(tx.Hash())
	if isDel {
		c.GetLocalDB().Set(key, nil)
		return &types.KeyValue{key, nil}, nil
	}

	var asset pt.ParacrossAsset

	amount, err := tx.Amount()
	if err != nil {
		return nil, err
	}
	asset.ParaHeight, err = getCommitHeight(txCommit.Payload)
	if err != nil {
		return nil, err
	}

	var payload pt.ParacrossAction
	err = types.Decode(tx.Payload, &payload)
	if err != nil {
		return nil, err
	}
	if payload.GetAssetWithdraw() == nil {
		return nil, errors.New("GetAssetWithdraw is nil")
	}
	exec := "coins"
	symbol := types.BTY
	if payload.GetAssetWithdraw().Cointoken != "" {
		exec = "token"
		symbol = payload.GetAssetWithdraw().Cointoken
	}

	asset = pt.ParacrossAsset{
		From:       tx.From(),
		To:         tx.To,
		Amount:     amount,
		IsWithdraw: isWithdraw,
		TxHash:     tx.Hash(),
		Height:     c.GetHeight(),
		Exec:       exec,
		Symbol:     symbol,
	}

	asset.CommitDoneHeight = c.GetHeight()
	asset.Success = success

	err = c.GetLocalDB().Set(key, types.Encode(&asset))
	if err != nil {
		clog.Error("para execLocal", "set", "", "failed", err)
	}
	return &types.KeyValue{key, types.Encode(&asset)}, nil
}

func (c *Paracross) updateLocalAssetTransfer(txCommit, tx *types.Transaction, success, isDel bool) (*types.KeyValue, error) {
	clog.Debug("para execLocal", "tx hash", common.Bytes2Hex(tx.Hash()))
	key := calcLocalAssetKey(tx.Hash())

	var asset pt.ParacrossAsset
	v, err := c.GetLocalDB().Get(key)
	if err != nil {
		return nil, err
	}
	err = types.Decode(v, &asset)
	if err != nil {
		panic(err)
	}
	if !isDel {
		asset.ParaHeight, err = getCommitHeight(txCommit.Payload)
		if err != nil {
			return nil, err
		}
		asset.CommitDoneHeight = c.GetHeight()
		asset.Success = success
	} else {
		asset.ParaHeight = 0
		asset.CommitDoneHeight = 0
		asset.Success = false
	}
	c.GetLocalDB().Set(key, types.Encode(&asset))
	return &types.KeyValue{key, types.Encode(&asset)}, nil
}

func (c *Paracross) IsFriend(myexec, writekey []byte, tx *types.Transaction) bool {
	//不允许平行链
	if types.IsPara() {
		return false
	}
	//friend 调用必须是自己在调用
	if string(myexec) != c.GetDriverName() {
		return false
	}
	//只允许同系列的执行器（tx 也必须是 paracross）
	if string(types.GetRealExecName(tx.Execer)) != c.GetDriverName() {
		return false
	}
	//只允许跨链交易
	return c.allow(tx, 0) == nil
}

func (c *Paracross) allow(tx *types.Transaction, index int) error {
	// 增加新的规则: 在主链执行器带着title的 asset-transfer/asset-withdraw 交易允许执行
	// 1. user.p.${tilte}.${paraX}
	// 1. payload 的 actionType = t/w
	if !types.IsPara() && c.allowIsParaAssetTx(tx.Execer) {
		var payload pt.ParacrossAction
		err := types.Decode(tx.Payload, &payload)
		if err != nil {
			return err
		}
		if payload.Ty == pt.ParacrossActionAssetTransfer || payload.Ty == pt.ParacrossActionAssetWithdraw {
			return nil
		}
	}
	return types.ErrNotAllow
}

func (c *Paracross) Allow(tx *types.Transaction, index int) error {
	//默认规则
	err := c.DriverBase.Allow(tx, index)
	if err == nil {
		return nil
	}
	//paracross 添加的规则
	return c.allow(tx, index)
}

func (c *Paracross) allowIsParaAssetTx(execer []byte) bool {
	if !bytes.HasPrefix(execer, types.ParaKey) {
		return false
	}
	count := 0
	index := 0
	s := len(types.ParaKey)
	for i := s; i < len(execer); i++ {
		if execer[i] == '.' {
			count++
			index = i
		}
	}
	if count == 1 && c.AllowIsSame(execer[index+1:]) {
		return true
	}
	return false
}
