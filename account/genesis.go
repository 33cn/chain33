package account

import (
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
)

var genesisKey = []byte("mavl-acc-genesis")

func GenesisInit(db dbm.KVDB, addr string, amount int64) (*types.Receipt, error) {
	g := &types.Genesis{}
	g.Isrun = true
	SaveGenesis(db, g)
	accTo := LoadAccount(db, addr)
	copyto := *accTo
	accTo.Balance = accTo.GetBalance() + amount
	receiptBalanceTo := &types.ReceiptAccountTransfer{&copyto, accTo}
	SaveAccount(db, accTo)
	receipt := genesisReceipt(accTo, receiptBalanceTo, g)
	return receipt, nil
}

func GetGenesis(db dbm.KVDB) *types.Genesis {
	value, err := db.Get(genesisKey)
	if err != nil {
		return &types.Genesis{}
	}
	var g types.Genesis
	err = types.Decode(value, &g)
	if err != nil {
		panic(err) //数据库已经损坏
	}
	return &g
}

func SaveGenesis(db dbm.KVDB, g *types.Genesis) {
	set := GetGenesisKVSet(g)
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func GetGenesisKVSet(g *types.Genesis) (kvset []*types.KeyValue) {
	value := types.Encode(g)
	kvset = append(kvset, &types.KeyValue{genesisKey, value})
	return kvset
}

func GenesisInitExec(db dbm.KVDB, addr string, amount int64, execaddr string) (*types.Receipt, error) {
	g := &types.Genesis{}
	g.Isrun = true
	SaveGenesis(db, g)
	accTo := LoadAccount(db, execaddr)
	copyto := *accTo
	accTo.Balance = accTo.GetBalance() + amount
	receiptBalanceTo := &types.ReceiptAccountTransfer{&copyto, accTo}
	SaveAccount(db, accTo)
	receipt := genesisReceipt(accTo, receiptBalanceTo, g)
	receipt2, err := execDeposit(db, addr, execaddr, amount)
	if err != nil {
		panic(err)
	}
	receipt = mergeReceipt(receipt, receipt2)
	return receipt, nil
}

func genesisReceipt(accTo *types.Account, receiptTo *types.ReceiptAccountTransfer, g *types.Genesis) *types.Receipt {
	log2 := &types.ReceiptLog{types.TyLogGenesisTransfer, types.Encode(receiptTo)}
	kv := GetGenesisKVSet(g)
	kv = append(kv, GetKVSet(accTo)...)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log2}}
}
