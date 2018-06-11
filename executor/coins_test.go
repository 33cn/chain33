package executor

import (
	"testing"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
)

//from->to, amount
func createTransfer(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	tx.Nonce = random.Int63()
	tx.To = to
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func createTransferToExec(priv crypto.PrivKey, execname string, amount int64) *types.Transaction {
	v := &types.CoinsAction_TransferToExec{&types.CoinsTransferToExec{ExecName: execname, Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransferToExec}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: account.ExecAddress(execname)}
	tx.Nonce = random.Int63()
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func createWithdrawFromExec(priv crypto.PrivKey, execname string, amount int64) *types.Transaction {
	v := &types.CoinsAction_Withdraw{&types.CoinsWithdraw{ExecName: execname, Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionWithdraw}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: account.ExecAddress(execname)}
	tx.Nonce = random.Int63()
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func TestCoins(t *testing.T) {
	q, chain, s := initEnv()
	prev := types.MinFee
	types.SetMinFee(100000)
	defer types.SetMinFee(prev)
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	block := createGenesisBlock()
	_, _, err := ExecBlock(q.Client(), zeroHash[:], block, false, true)
	if err != nil {
		t.Error(err)
		return
	}
	printAccount(t, q.Client(), block.StateHash, cfg.Consensus.Genesis)
	var txs []*types.Transaction
	addr2, priv2 := genaddress()
	txs = append(txs, createTransfer(genkey, addr2, 2*types.Coin))
	txs = append(txs, createTransferToExec(priv2, "user.name", types.Coin))
	txs = append(txs, createWithdrawFromExec(priv2, "user.name", types.Coin))
	block = execAndCheckBlock(t, q.Client(), block, txs, types.ExecOk)
	printAccount(t, q.Client(), block.StateHash, cfg.Consensus.Genesis)
	printAccount(t, q.Client(), block.StateHash, addr2)
}
