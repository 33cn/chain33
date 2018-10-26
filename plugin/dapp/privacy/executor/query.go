package executor

import (
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (this *privacy) Query_ShowAmountsOfUTXO(param *pty.ReqPrivacyToken) (types.Message, error) {
	return this.ShowAmountsOfUTXO(param)
}

func (this *privacy) Query_ShowUTXOs4SpecifiedAmount(param *pty.ReqPrivacyToken) (types.Message, error) {
	return this.ShowUTXOs4SpecifiedAmount(param)
}

func (this *privacy) Query_GetUTXOGlobalIndex(param *pty.ReqUTXOGlobalIndex) (types.Message, error) {
	return this.getGlobalUtxoIndex(param)
}

func (this *privacy) Query_GetTxsByAddr(param *types.ReqAddr) (types.Message, error) {
	return this.GetTxsByAddr(param)
}
