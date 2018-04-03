package account

//package for account manger
//1. load from db
//2. save to db
//3. KVSet
//4. Transfer
//5. Add
//6. Sub
//7. Account balance query
//8. gen a private key -> private key to address (bitcoin likes)

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var alog = log.New("module", "account")

type AccountDB struct {
	db                   dbm.KVDB
	accountKeyPerfix     []byte
	execAccountKeyPerfix []byte
}

func NewCoinsAccount() *AccountDB {
	return newAccountDB("mavl-coins-bty-")
}

func NewTokenAccount(symbol string, db dbm.KVDB) *AccountDB {
	accDB := newAccountDB(fmt.Sprintf("mavl-token-%s-", symbol))
	accDB.SetDB(db)
	return accDB
}

func NewTokenAccountWithoutDB(symbol string) *AccountDB {
	return newAccountDB(fmt.Sprintf("mavl-token-%s-", symbol))
}

func newAccountDB(prefix string) *AccountDB {
	acc := &AccountDB{}
	acc.accountKeyPerfix = []byte(prefix)
	acc.execAccountKeyPerfix = append([]byte(prefix), []byte("exec-")...)
	//alog.Warn("NewAccountDB", "prefix", prefix, "key1", string(acc.accountKeyPerfix), "key2", string(acc.execAccountKeyPerfix))
	return acc
}

func (acc *AccountDB) SetDB(db dbm.KVDB) *AccountDB {
	acc.db = db
	return acc
}

func (acc *AccountDB) IsTokenAccount() bool {
	return "token" == string(acc.accountKeyPerfix[len("mavl-"):len("mavl-token")])
}

func (acc *AccountDB) LoadAccount(addr string) *types.Account {
	value, err := acc.db.Get(acc.AccountKey(addr))
	if err != nil {
		return &types.Account{Addr: addr}
	}
	var acc1 types.Account
	err = types.Decode(value, &acc1)
	if err != nil {
		panic(err) //数据库已经损坏
	}
	return &acc1
}

func (acc *AccountDB) CheckTransfer(from, to string, amount int64) error {
	if !types.CheckAmount(amount) {
		return types.ErrAmount
	}
	accFrom := acc.LoadAccount(from)
	b := accFrom.GetBalance() - amount
	if b < 0 {
		return types.ErrNoBalance
	}
	return nil
}

func (acc *AccountDB) Transfer(from, to string, amount int64) (*types.Receipt, error) {
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	accFrom := acc.LoadAccount(from)
	accTo := acc.LoadAccount(to)
	if accFrom.Addr == accTo.Addr {
		return nil, types.ErrSendSameToRecv
	}
	if accFrom.GetBalance()-amount >= 0 {
		copyfrom := *accFrom
		copyto := *accTo

		accFrom.Balance = accFrom.GetBalance() - amount
		accTo.Balance = accTo.GetBalance() + amount

		receiptBalanceFrom := &types.ReceiptAccountTransfer{&copyfrom, accFrom}
		receiptBalanceTo := &types.ReceiptAccountTransfer{&copyto, accTo}

		acc.SaveAccount(accFrom)
		acc.SaveAccount(accTo)
		return acc.transferReceipt(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo), nil
	} else {
		return nil, types.ErrNoBalance
	}
}

func (acc *AccountDB) depositBalance(execaddr string, amount int64) (*types.Receipt, error) {
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadAccount(execaddr)
	copyacc := *acc1
	acc1.Balance += amount
	receiptBalance := &types.ReceiptAccountTransfer{&copyacc, acc1}
	acc.SaveAccount(acc1)
	ty := int32(types.TyLogDeposit)
	if acc.IsTokenAccount() {
		ty = types.TyLogTokenDeposit
	}
	log1 := &types.ReceiptLog{ty, types.Encode(receiptBalance)}
	kv := acc.GetKVSet(acc1)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log1}}, nil
}

func (acc *AccountDB) transferReceipt(accFrom, accTo *types.Account, receiptFrom, receiptTo *types.ReceiptAccountTransfer) *types.Receipt {
	ty := int32(types.TyLogTransfer)
	if acc.IsTokenAccount() {
		ty = types.TyLogTokenTransfer
	}
	log1 := &types.ReceiptLog{ty, types.Encode(receiptFrom)}
	log2 := &types.ReceiptLog{ty, types.Encode(receiptTo)}
	kv := acc.GetKVSet(accFrom)
	kv = append(kv, acc.GetKVSet(accTo)...)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log1, log2}}
}

func (acc *AccountDB) SaveAccount(acc1 *types.Account) {
	set := acc.GetKVSet(acc1)
	for i := 0; i < len(set); i++ {
		acc.db.Set(set[i].GetKey(), set[i].Value)
	}
}

func (acc *AccountDB) GetKVSet(acc1 *types.Account) (kvset []*types.KeyValue) {
	value := types.Encode(acc1)
	kvset = append(kvset, &types.KeyValue{acc.AccountKey(acc1.Addr), value})
	return kvset
}

func (acc *AccountDB) LoadAccounts(client queue.Client, addrs []string) (accs []*types.Account, err error) {
	msg := client.NewMessage("blockchain", types.EventGetLastHeader, nil)
	client.Send(msg, true)
	msg, err = client.Wait(msg)
	if err != nil {
		return nil, err
	}
	get := types.StoreGet{}
	get.StateHash = msg.GetData().(*types.Header).GetStateHash()
	for i := 0; i < len(addrs); i++ {
		get.Keys = append(get.Keys, acc.AccountKey(addrs[i]))
	}
	msg = client.NewMessage("store", types.EventStoreGet, &get)
	client.Send(msg, true)
	msg, err = client.Wait(msg)
	if err != nil {
		return nil, err
	}
	values := msg.GetData().(*types.StoreReplyValue)
	for i := 0; i < len(values.Values); i++ {
		value := values.Values[i]
		if value == nil {
			accs = append(accs, &types.Account{Addr: addrs[i]})
		} else {
			var acc types.Account
			err := types.Decode(value, &acc)
			if err != nil {
				return nil, err
			}
			accs = append(accs, &acc)
		}
	}
	return accs, nil
}

func (acc *AccountDB) LoadAccountsDB(addrs []string) (accs []*types.Account, err error) {
	for i := 0; i < len(addrs); i++ {
		acc1 := acc.LoadAccount(addrs[i])
		accs = append(accs, acc1)
	}
	return accs, nil
}

//address to save key
func (acc *AccountDB) AccountKey(address string) (key []byte) {
	key = append(key, acc.accountKeyPerfix...)
	key = append(key, []byte(address)...)
	return key
}
