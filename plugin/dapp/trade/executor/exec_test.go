package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	"gitlab.33.cn/chain33/chain33/types"
)

type execEnv struct {
	blockTime   int64 // 1539918074
	blockHeight int64
	index       int
	difficulty  uint64

	txHash string
}

type orderArgs struct {
	amount int64
	min    int64
	price  int64
	total  int64
}

var (
	Symbol         = "TEST"
	AssetExecToken = "token"
	AssetExecPara  = "paracross"

	PrivKeyA = "0x6da92a632ab7deb67d38c0f6560bcfed28167998f6496db64c258d5e8393a81b" // 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4
	PrivKeyB = "0x19c069234f9d3e61135fefbeb7791b149cdf6af536f26bebb310d4cd22c3fee4" // 1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR
	PrivKeyC = "0x7a80a1f75d7360c6123c32a78ecf978c1ac55636f87892df38d8b85a9aeff115" // 1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k
	PrivKeyD = "0xcacb1f5d51700aea07fca2246ab43b0917d70405c65edea9b5063d72eb5c6b71" // 1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs
	Nodes    = [][]byte{
		[]byte("1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"),
		[]byte("1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR"),
		[]byte("1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"),
		[]byte("1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"),
	}
)

func TestTrade_Exec_SellLimit(t *testing.T) {
	sellArgs := &orderArgs{100, 2, 2, 100}
	buyArgs := &orderArgs{total: 5}
	expect := &orderArgs{total: sellArgs.total - buyArgs.total}

	total := int64(100000)
	accountA := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    string(Nodes[0]),
	}
	accountB := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    string(Nodes[1]),
	}

	env := execEnv{
		1539918074,
		types.GetDappFork("trade", "ForkTradeAsset"),
		2,
		1539918074,
		"hash",
	}

	stateDB, _ := dbm.NewGoMemDB("1", "2", 100)
	accB := account.NewCoinsAccount()
	accB.SetDB(stateDB)
	accB.SaveExecAccount(address.ExecAddress("trade"), &accountB)

	accA, _ := account.NewAccountDB(AssetExecToken, Symbol, stateDB)
	accA.SaveExecAccount(address.ExecAddress("trade"), &accountA)

	driver := newTrade()
	driver.SetEnv(env.blockHeight, env.blockTime, env.difficulty)
	driver.SetStateDB(stateDB)

	sell := &pty.TradeSellTx{
		Symbol,
		sellArgs.amount,
		sellArgs.min,
		sellArgs.price,
		sellArgs.total,
		0,
		AssetExecToken,
	}
	tx, _ := pty.CreateRawTradeSellTx(sell)
	tx, _ = signTx(tx, PrivKeyA)

	receipt, err := driver.Exec(tx, env.index)
	if err != nil {
		assert.Nil(t, err, "exec failed")
		return
	}

	var acc types.Account
	err = types.Decode(receipt.KV[0].Value, &acc)
	assert.Nil(t, err, "decode account")
	t.Log(acc)
	assert.Equal(t, total-sellArgs.total*sellArgs.amount, acc.Balance)
	assert.Equal(t, sellArgs.total*sellArgs.amount, acc.Frozen)

	var sellOrder pty.SellOrder
	err = types.Decode(receipt.KV[1].Value, &sellOrder)
	assert.Nil(t, err)
	assert.Equal(t, sellArgs.amount, sellOrder.AmountPerBoardlot)
	assert.Equal(t, sellArgs.total, sellOrder.TotalBoardlot)
	assert.Equal(t, sellArgs.price, sellOrder.PricePerBoardlot)
	assert.Equal(t, sellArgs.min, sellOrder.MinBoardlot)
	assert.Equal(t, AssetExecToken, sellOrder.AssetExec)
	assert.Equal(t, Symbol, sellOrder.TokenSymbol)
	assert.Equal(t, int64(0), sellOrder.SoldBoardlot)
	assert.Equal(t, string(Nodes[0]), sellOrder.Address)

	buy := &pty.TradeBuyTx{
		sellOrder.SellID,
		buyArgs.total,
		0,
	}
	tx, _ = pty.CreateRawTradeBuyTx(buy)
	tx, _ = signTx(tx, PrivKeyB)
	receipt, err = driver.Exec(tx, env.index)
	if err != nil {
		assert.Nil(t, err, "exec failed")
		return
	}
	// but coins -, sell coins +, sell asset -, buy asset +, sell order
	err = types.Decode(receipt.KV[0].Value, &acc)
	assert.Nil(t, err)
	assert.Equal(t, accountB.Balance-buyArgs.total*sellArgs.price, acc.Balance)

	err = types.Decode(receipt.KV[1].Value, &acc)
	assert.Nil(t, err)
	assert.Equal(t, buyArgs.total*sellArgs.price, acc.Balance)

	err = types.Decode(receipt.KV[2].Value, &acc)
	assert.Nil(t, err)
	assert.Equal(t, (sellArgs.total-buyArgs.total)*sellArgs.amount, acc.Frozen)

	err = types.Decode(receipt.KV[3].Value, &acc)
	assert.Nil(t, err)
	assert.Equal(t, buyArgs.total*sellArgs.amount, acc.Balance)

	err = types.Decode(receipt.KV[4].Value, &sellOrder)
	assert.Nil(t, err)
	assert.Equal(t, expect.total, sellOrder.TotalBoardlot-sellOrder.SoldBoardlot)

}

func TestTrade_Exec_BuyLimit(t *testing.T) {
	buyArgs := &orderArgs{100, 2, 2, 100}
	sellArgs := &orderArgs{total: 5}
	expect := &orderArgs{total: buyArgs.total - sellArgs.total}

	total := int64(100000)
	accountA := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    string(Nodes[0]),
	}
	accountB := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    string(Nodes[1]),
	}

	env := execEnv{
		1539918074,
		types.GetDappFork("trade", "ForkTradeAsset"),
		2,
		1539918074,
		"hash",
	}

	stateDB, _ := dbm.NewGoMemDB("1", "2", 100)
	accB := account.NewCoinsAccount()
	accB.SetDB(stateDB)
	accB.SaveExecAccount(address.ExecAddress("trade"), &accountB)

	accA, _ := account.NewAccountDB(AssetExecPara, Symbol, stateDB)
	accA.SaveExecAccount(address.ExecAddress("trade"), &accountA)

	driver := newTrade()
	driver.SetEnv(env.blockHeight, env.blockTime, env.difficulty)
	driver.SetStateDB(stateDB)

	buy := &pty.TradeBuyLimitTx{
		Symbol,
		buyArgs.amount,
		buyArgs.min,
		buyArgs.price,
		buyArgs.total,
		0,
		AssetExecPara,
	}
	tx, _ := pty.CreateRawTradeBuyLimitTx(buy)
	tx, _ = signTx(tx, PrivKeyB)

	receipt, err := driver.Exec(tx, env.index)
	if err != nil {
		assert.Nil(t, err, "exec failed")
		return
	}

	var acc types.Account
	err = types.Decode(receipt.KV[0].Value, &acc)
	assert.Nil(t, err, "decode account")
	t.Log(acc)
	assert.Equal(t, total-buyArgs.total*buyArgs.price, acc.Balance)
	assert.Equal(t, buyArgs.total*buyArgs.price, acc.Frozen)

	var buyLimitOrder pty.BuyLimitOrder
	err = types.Decode(receipt.KV[1].Value, &buyLimitOrder)
	assert.Nil(t, err)
	assert.Equal(t, buyArgs.amount, buyLimitOrder.AmountPerBoardlot)
	assert.Equal(t, buyArgs.total, buyLimitOrder.TotalBoardlot)
	assert.Equal(t, buyArgs.price, buyLimitOrder.PricePerBoardlot)
	assert.Equal(t, buyArgs.min, buyLimitOrder.MinBoardlot)
	assert.Equal(t, AssetExecPara, buyLimitOrder.AssetExec)
	assert.Equal(t, Symbol, buyLimitOrder.TokenSymbol)
	assert.Equal(t, int64(0), buyLimitOrder.BoughtBoardlot)
	assert.Equal(t, string(Nodes[1]), buyLimitOrder.Address)

	sell := &pty.TradeSellMarketTx{
		buyLimitOrder.BuyID,
		sellArgs.total,
		0,
	}
	tx, _ = pty.CreateRawTradeSellMarketTx(sell)
	tx, _ = signTx(tx, PrivKeyA)
	receipt, err = driver.Exec(tx, env.index)
	if err != nil {
		assert.Nil(t, err, "exec failed")
		return
	}
	// buy coins -, sell coins +, sell asset -, buy asset +, buy order
	err = types.Decode(receipt.KV[0].Value, &acc)
	assert.Nil(t, err)
	assert.Equal(t, (buyArgs.total-sellArgs.total)*buyArgs.price, acc.Frozen)
	assert.Equal(t, total-buyArgs.total*buyArgs.price, acc.Balance)

	err = types.Decode(receipt.KV[1].Value, &acc)
	assert.Nil(t, err)
	assert.Equal(t, buyArgs.price*sellArgs.total, acc.Balance)

	err = types.Decode(receipt.KV[2].Value, &acc)
	assert.Nil(t, err)
	assert.Equal(t, total-buyArgs.amount*sellArgs.total, acc.Balance)

	err = types.Decode(receipt.KV[3].Value, &acc)
	assert.Nil(t, err)
	assert.Equal(t, sellArgs.total*buyArgs.amount, acc.Balance)

	err = types.Decode(receipt.KV[4].Value, &buyLimitOrder)
	assert.Nil(t, err)
	assert.Equal(t, expect.total, buyLimitOrder.TotalBoardlot-buyLimitOrder.BoughtBoardlot)
}

func signTx(tx *types.Transaction, hexPrivKey string) (*types.Transaction, error) {
	signType := types.SECP256K1
	c, err := crypto.New(types.GetSignName("trade", signType))
	if err != nil {
		return tx, err
	}

	bytes, err := common.FromHex(hexPrivKey[:])
	if err != nil {
		return tx, err
	}

	privKey, err := c.PrivKeyFromBytes(bytes)
	if err != nil {
		return tx, err
	}

	tx.Sign(int32(signType), privKey)
	return tx, nil
}
