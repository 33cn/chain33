package relay

/*
relay执行器支持跨链token的创建和交易，

主要提供操作有以下几种：
1）挂BTY单出售购买BTC 或者已有BTC买别人的BTY(未实现)；
2）挂TOKEN单出售购买BTC 或者已有BTC买别人的TOKEN（未实现）；
3）买家购买指定的卖单；
4）买家交易相应数额的BTC给卖家，并且提取交易信息在chain33上通过relay验证，验证通过获取相应BTY
5）撤销卖单；
6）插销买单

*/

import (
	log "github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var relaylog = log.New("module", "execs.relay")

func init() {
	r := newRelay()
	drivers.Register(r.GetName(), r, types.ForkV7_add_relay) //TODO: ForkV2_add_token
}

type relay struct {
	drivers.DriverBase
}

func newRelay() *relay {
	r := &relay{}
	r.SetChild(r)
	return r
}

func (r *relay) GetName() string {
	return "relay"
}

func (r *relay) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.RelayAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	relaylog.Info("exec relay tx", "tx hash", common.Bytes2Hex(tx.Hash()), "Ty", relay.GetTy())

	actiondb := newRelayAction(r, tx)
	switch action.GetTy() {
	case types.RelayActionSell:
		return actiondb.relaySell(action.GetRsell())

	case types.RelayActionRevokeSell:
		return actiondb.relayRevokeSell(action.GetRrevokesell())

	case types.RelayActionBuy:
		return actiondb.relayBuy(action.GetRbuy())

	case types.RelayActionRevokeBuy:
		return actiondb.relayRevokeBuy(action.GetRrevokebuy())

	/* orderid, rawTx, index sibling, blockhash */
	case types.RelayActionVerifyBTCTx:
		return actiondb.relayVerifyBTCTx(action.GetRverifybtc)

	default:
		return nil, types.ErrActionNotSupport
	}
}

func (r *relay) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {

}

func (r *relay) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := r.DriverBase.ExecDelLocal(tx, receipt, index)

}
