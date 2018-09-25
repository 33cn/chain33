package executor

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common/db"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
)

// 注： 在计算帐号地址时， 平行链paracross合约地址需要带上title前缀，才能表现出和主链一致, 但是现在不带，

// 其中带{}, 都表示变量， 用需要用真实的地址， 符号代替
// 构建主链资产在平行链paracross帐号
// execName:  user.p.{guodun}.paracross
// symbol: coins.bty, token.{TEST}
// 完整的帐号地址 mavl-{paracross}-coins.bty-{user-address} 不带title{paracross}
// 对应主链上paracross 子帐号 malv-coins-bty-exec-{Address(paracross)}:{Address(user.p.{guodun}.paracross)}
func NewParaAccount(paraTitle, mainExecName, mainSymbol string, db db.KV) (*account.DB, error) {
	// 按照现在的配置， title 是 带 "." 做结尾
	// paraExec := paraTitle + types.ParaX
	paraExec := types.ParaX // 现在平行链是执行器注册和算地址是不带前缀的，
	// 如果能确保(或规定) tokne 的 symbol  和 coins 中的 symbol 不会混淆，  localExecName 可以不要
	paraSymbol := mainExecName + "." +  mainSymbol
	return account.NewAccountDB(paraExec, paraSymbol, db)
}

// 以后如果支持从平行链资产转移到主链， 构建平行链资产在主链的paracross帐号
// execName: paracross
// symbol: user.p.{guodun}.coins.{guodun}  user.p.{guodun}.token.{TEST}
// 完整的帐号地址 mavl-paracross-user.p.{guodun}.coins.guodun-{user-address}
// 对应平行链上子地址  mavl-coins-{guodun}-exec-{Address(paracross)}:{Address(paracross)}
func NewMainAccount(paraTitle, paraExecName, paraSymbol string, db db.KV) (*account.DB, error) {
	mainSymbol := paraTitle + paraExecName + "." + paraSymbol
	return account.NewAccountDB(types.ParaX, mainSymbol, db)
}

func assetDepositBalance(acc *account.DB, addr string, amount int64) (*types.Receipt, error) {
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadAccount(addr)
	copyacc := *acc1
	acc1.Balance += amount
	receiptBalance := &types.ReceiptAccountTransfer{
		Prev:    &copyacc,
		Current: acc1,
	}
	acc.SaveAccount(acc1)
	ty := int32(pt.TyLogParaAssetDeposit)
	log1 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(receiptBalance),
	}
	kv := acc.GetKVSet(acc1)
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1},
	}, nil
}

func assetWithdrawBalance(acc *account.DB, addr string, amount int64) (*types.Receipt, error) {
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadAccount(addr)
	if acc1.Balance-amount < 0 {
		return nil, types.ErrNoBalance
	}
	copyacc := *acc1
	acc1.Balance -= amount
	receiptBalance := &types.ReceiptAccountTransfer{
		Prev:     &copyacc,
		Current:  acc1,
	}
	acc.SaveAccount(acc1)
	ty := int32(pt.TyLogParaAssetWithdraw)
	log1 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(receiptBalance),
	}
	kv := acc.GetKVSet(acc1)
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1},
	}, nil
}

//                          trade add                                user address
// mavl-token-test-exec-1HPkPopVe3ERfvaAgedDtJQ792taZFEHCe:13DP8mVru5Rtu6CrjXQMvLsjvve3epRR1i
// mavl-conis-bty-exec-{para}1e:13DP8mVru5Rtu6CrjXQMvLsjvve3epRR1i


// 用户
//      mavl- `合约` - `币名称` - 地址

// 用户在执行器里的子帐号
//      mavl- `合约` - `币名称` -  exec - `执行器地址` ： 地址                            10    - 5 |  5
//      mavl- `合约` - `币名称` -  exec - `执行器地址` ： `平行链paracross地址`                      |  5

// 执行器里的帐号
//     mavl- `合约` - `币名称` -`执行器地址`
//
// 带title的hu


// 平行链
//   `合约`    paracross  ` : 主链上的    user.p.guodun.paracross`
//    `币名称`         coins.bty
// mavl- `合约` - `币名称` - 地址                                                              5
//


// transfer / withdraw
//

// mavl -token  - xxx - aaaaa