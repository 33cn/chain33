package executor

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/db"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	"gitlab.33.cn/chain33/chain33/types"
)

/*
   在以前版本中只有token 合约发行的币在trade 里面交易， 订单中 symbol 为 token 的symbol，
   现在 symbol 扩展成 exec.sybol@title, @title 先忽略， (因为不必要, 只支持主链到平行链)。
   在订单中增加 exec， 表示币从那个合约中来的。

   在主链
     原来的订单  exec = "" symbol = "TEST"
     新的订单    exec =  "token"  symbol = "token.TEST"

   在平行链, 主链资产和本链资产的表示区别如下
     exec = "paracross"  symbol = "token.TEST"
     exec = "token"      symbol = "token.TEST"

*/

// return exec, symbol
func GetExecSymbol(order *pt.SellOrder) (string, string) {
	if order.AssetExec == "" {
		return defaultAssetExec, defaultAssetExec + "." + order.TokenSymbol
	}
	return order.AssetExec, order.TokenSymbol
}

func checkAsset(height int64, exec, symbol string) bool {
	if types.IsDappFork(height, pt.TradeX, "ForkTradeAsset") {
		if exec == "" || symbol == "" {
			return false
		}
	} else {
		if exec != "" {
			return false
		}
	}
	return true
}

func createAccountDB(height int64, db db.KV, exec, symbol string) (*account.DB, error) {
	if types.IsDappFork(height, pt.TradeX, "ForkTradeAsset") {
		return account.NewAccountDB(exec, symbol, db)
	} else {
		return account.NewAccountDB(defaultAssetExec, symbol, db)
	}
}
