package account

import (
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

func LoadExecAccount(db dbm.KVDB, addr, execaddr string) *types.Account {
	value, err := db.Get(ExecKey(addr, execaddr))
	if err != nil {
		return &types.Account{Addr: addr}
	}
	var acc types.Account
	err = types.Decode(value, &acc)
	if err != nil {
		panic(err) //数据库已经损坏
	}
	return &acc
}

func LoadExecAccountQueue(q *queue.Queue, addr, execaddr string) (*types.Account, error) {
	client := q.NewClient()
	//get current head ->
	msg := client.NewMessage("blockchain", types.EventGetLastHeader, nil)
	client.Send(msg, true)
	msg, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}
	get := types.StoreGet{}
	get.StateHash = msg.GetData().(*types.Header).GetStateHash()
	get.Keys = append(get.Keys, ExecKey(addr, execaddr))
	msg = client.NewMessage("store", types.EventStoreGet, &get)
	client.Send(msg, true)
	msg, err = client.Wait(msg)
	if err != nil {
		return nil, err
	}
	values := msg.GetData().(*types.StoreReplyValue)
	value := values.Values[0]
	if value == nil {
		return &types.Account{Addr: addr}, nil
	} else {
		var acc types.Account
		err := types.Decode(value, &acc)
		if err != nil {
			return nil, err
		}
		return &acc, nil
	}
}

func SaveExecAccount(db dbm.KVDB, execaddr string, acc *types.Account) {
	set := GetExecKVSet(execaddr, acc)
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func GetExecKVSet(execaddr string, acc *types.Account) (kvset []*types.KeyValue) {
	value := types.Encode(acc)
	kvset = append(kvset, &types.KeyValue{ExecKey(acc.Addr, execaddr), value})
	return kvset
}

func ExecKey(address, execaddr string) (key []byte) {
	key = append(key, []byte("mavl-acc-exec-")...)
	key = append(key, []byte(execaddr)...)
	key = append(key, []byte(":")...)
	key = append(key, []byte(address)...)
	return key
}

func TransferToExec(db dbm.KVDB, from, to string, amount int64) (*types.Receipt, error) {
	receipt, err := Transfer(db, from, to, amount)
	if err != nil {
		return nil, err
	}
	receipt2, err := execDeposit(db, from, to, amount)
	if err != nil {
		//存款不应该出任何问题
		panic(err)
	}
	return mergeReceipt(receipt, receipt2), nil
}

func TransferWithdraw(db dbm.KVDB, from, to string, amount int64) (*types.Receipt, error) {
	//先判断可以取款
	if err := CheckTransfer(db, to, from, amount); err != nil {
		return nil, err
	}
	receipt, err := execWithdraw(db, to, from, amount)
	if err != nil {
		return nil, err
	}
	//然后执行transfer
	receipt2, err := Transfer(db, to, from, amount)
	if err != nil {
		panic(err) //在withdraw
	}
	return mergeReceipt(receipt, receipt2), nil
}

//四个操作中 Deposit 自动完成，不需要模块外的函数来调用
func ExecFrozen(db dbm.KVDB, addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc := LoadExecAccount(db, addr, execaddr)
	if acc.Balance-amount < 0 {
		alog.Error("ExecFrozen", "balance", acc.Balance, "amount", amount)
		return nil, types.ErrNoBalance
	}
	copyacc := *acc
	acc.Balance -= amount
	acc.Frozen += amount
	receiptBalance := &types.ReceiptExecAccount{execaddr, &copyacc, acc}
	SaveExecAccount(db, execaddr, acc)
	return execReceipt(acc, receiptBalance), nil
}

func ExecActive(db dbm.KVDB, addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc := LoadExecAccount(db, addr, execaddr)
	if acc.Frozen-amount < 0 {
		return nil, types.ErrNoBalance
	}
	copyacc := *acc
	acc.Balance += amount
	acc.Frozen -= amount
	receiptBalance := &types.ReceiptExecAccount{execaddr, &copyacc, acc}
	SaveExecAccount(db, execaddr, acc)
	return execReceipt(acc, receiptBalance), nil
}

func ExecTransfer(db dbm.KVDB, from, to, execaddr string, amount int64) (*types.Receipt, error) {
	if from == to {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	accFrom := LoadExecAccount(db, from, execaddr)
	accTo := LoadExecAccount(db, to, execaddr)

	b := accFrom.GetBalance() - amount
	if b < 0 {
		return nil, types.ErrNoBalance
	}
	copyaccFrom := *accFrom
	copyaccTo := *accTo

	accFrom.Balance -= amount
	accTo.Balance += amount

	receiptBalanceFrom := &types.ReceiptExecAccount{execaddr, &copyaccFrom, accFrom}
	receiptBalanceTo := &types.ReceiptExecAccount{execaddr, &copyaccTo, accTo}

	SaveExecAccount(db, execaddr, accFrom)
	SaveExecAccount(db, execaddr, accTo)
	return execReceipt2(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo), nil
}

//从自己冻结的钱里面扣除，转移到别人的活动钱包里面去
func ExecTransferFrozen(db dbm.KVDB, from, to, execaddr string, amount int64) (*types.Receipt, error) {
	if from == to {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	accFrom := LoadExecAccount(db, from, execaddr)
	accTo := LoadExecAccount(db, to, execaddr)
	b := accFrom.GetFrozen() - amount
	if b < 0 {
		return nil, types.ErrNoBalance
	}
	copyaccFrom := *accFrom
	copyaccTo := *accTo

	accFrom.Frozen -= amount
	accTo.Balance += amount

	receiptBalanceFrom := &types.ReceiptExecAccount{execaddr, &copyaccFrom, accFrom}
	receiptBalanceTo := &types.ReceiptExecAccount{execaddr, &copyaccTo, accTo}

	SaveExecAccount(db, execaddr, accFrom)
	SaveExecAccount(db, execaddr, accTo)
	return execReceipt2(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo), nil
}

var addrSeed = []byte("address seed bytes for public key")
var bname [200]byte

func ExecAddress(name string) *Address {
	if len(name) > 100 {
		panic("name too long")
	}
	buf := append(bname[:0], addrSeed...)
	buf = append(buf, []byte(name)...)
	hash := common.Sha2Sum(buf)
	return PubKeyToAddress(hash[:])
}

func ExecDepositFrozen(db dbm.KVDB, addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	//这个函数只有挖矿的合约才能调用
	list := types.AllowDepositExec
	allow := false
	for _, exec := range list {
		if ExecAddress(exec).String() == execaddr {
			allow = true
			break
		}
	}
	if !allow {
		return nil, types.ErrNotAllowDeposit
	}
	receipt1, err := depositBalance(db, execaddr, amount)
	if err != nil {
		return nil, err
	}
	receipt2, err := execDepositFrozen(db, addr, execaddr, amount)
	if err != nil {
		return nil, err
	}
	return mergeReceipt(receipt1, receipt2), nil
}

func execDepositFrozen(db dbm.KVDB, addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc := LoadExecAccount(db, addr, execaddr)
	copyacc := *acc
	acc.Frozen += amount
	receiptBalance := &types.ReceiptExecAccount{execaddr, &copyacc, acc}
	//alog.Debug("execDeposit", "addr", addr, "execaddr", execaddr, "account", acc)
	SaveExecAccount(db, execaddr, acc)
	return execReceipt(acc, receiptBalance), nil
}

func execDeposit(db dbm.KVDB, addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc := LoadExecAccount(db, addr, execaddr)
	copyacc := *acc
	acc.Balance += amount
	receiptBalance := &types.ReceiptExecAccount{execaddr, &copyacc, acc}
	//alog.Debug("execDeposit", "addr", addr, "execaddr", execaddr, "account", acc)
	SaveExecAccount(db, execaddr, acc)
	return execReceipt(acc, receiptBalance), nil
}

func execWithdraw(db dbm.KVDB, execaddr, addr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc := LoadExecAccount(db, addr, execaddr)
	if acc.Balance-amount < 0 {
		return nil, types.ErrNoBalance
	}
	copyacc := *acc
	acc.Balance -= amount
	receiptBalance := &types.ReceiptExecAccount{execaddr, &copyacc, acc}
	SaveExecAccount(db, execaddr, acc)
	return execReceipt(acc, receiptBalance), nil
}

func execReceipt(acc *types.Account, r *types.ReceiptExecAccount) *types.Receipt {
	log1 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(r)}
	kv := GetExecKVSet(r.ExecAddr, acc)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log1}}
}

func execReceipt2(acc1, acc2 *types.Account, r1, r2 *types.ReceiptExecAccount) *types.Receipt {
	log1 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(r1)}
	log2 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(r2)}
	kv := GetExecKVSet(r1.ExecAddr, acc1)
	kv = append(kv, GetExecKVSet(r2.ExecAddr, acc2)...)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log1, log2}}
}

func mergeReceipt(receipt, receipt2 *types.Receipt) *types.Receipt {
	receipt.Logs = append(receipt.Logs, receipt2.Logs...)
	receipt.KV = append(receipt.KV, receipt2.KV...)
	return receipt
}

func GenesisInitExec(db dbm.KVDB, addr string, amount int64, execaddr string) (*types.Receipt, error) {
	g := &types.Genesis{}
	g.Isrun = true
	SaveGenesis(db, g)
	accTo := LoadAccount(db, execaddr)
	tob := accTo.GetBalance() + amount
	receiptBalanceTo := &types.ReceiptBalance{accTo.GetBalance(), tob, amount}
	accTo.Balance = tob
	SaveAccount(db, accTo)
	receipt := genesisReceipt(accTo, receiptBalanceTo, g)
	if execaddr != "" {
		receipt2, err := execDeposit(db, addr, execaddr, amount)
		if err != nil {
			panic(err)
		}
		receipt = mergeReceipt(receipt, receipt2)
	}
	return receipt, nil
}

func genesisReceipt(accTo *types.Account, receiptBalanceTo *types.ReceiptBalance, g *types.Genesis) *types.Receipt {
	log1 := &types.ReceiptLog{types.TyLogGenesis, nil}
	log2 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(receiptBalanceTo)}
	kv := GetGenesisKVSet(g)
	kv = append(kv, GetKVSet(accTo)...)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log1, log2}}
}
