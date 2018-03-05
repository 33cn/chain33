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
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var alog = log.New("module", "account")

type AccountDB struct {
	db               dbm.KVDB
	accountKeyPerfix []byte
}

func LoadAccount(db dbm.KVDB, addr string) *types.Account {
	value, err := db.Get(AccountKey(addr))
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

func CheckTransfer(db dbm.KVDB, from, to string, amount int64) error {
	if !types.CheckAmount(amount) {
		return types.ErrAmount
	}
	accFrom := LoadAccount(db, from)
	b := accFrom.GetBalance() - amount
	if b < 0 {
		return types.ErrNoBalance
	}
	return nil
}

func Transfer(db dbm.KVDB, from, to string, amount int64) (*types.Receipt, error) {
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	accFrom := LoadAccount(db, from)
	accTo := LoadAccount(db, to)
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

		SaveAccount(db, accFrom)
		SaveAccount(db, accTo)
		return transferReceipt(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo), nil
	} else {
		return nil, types.ErrNoBalance
	}
}

func depositBalance(db dbm.KVDB, execaddr string, amount int64) (*types.Receipt, error) {
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc := LoadAccount(db, execaddr)
	copyacc := *acc
	acc.Balance += amount
	receiptBalance := &types.ReceiptAccountTransfer{&copyacc, acc}
	SaveAccount(db, acc)
	log1 := &types.ReceiptLog{types.TyLogDeposit, types.Encode(receiptBalance)}
	kv := GetKVSet(acc)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log1}}, nil
}

func transferReceipt(accFrom, accTo *types.Account, receiptFrom, receiptTo *types.ReceiptAccountTransfer) *types.Receipt {
	log1 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(receiptFrom)}
	log2 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(receiptTo)}
	kv := GetKVSet(accFrom)
	kv = append(kv, GetKVSet(accTo)...)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log1, log2}}
}

func SaveAccount(db dbm.KVDB, acc *types.Account) {
	set := GetKVSet(acc)
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func GetKVSet(acc *types.Account) (kvset []*types.KeyValue) {
	value := types.Encode(acc)
	kvset = append(kvset, &types.KeyValue{AccountKey(acc.Addr), value})
	return kvset
}

func LoadAccounts(client queue.Client, addrs []string) (accs []*types.Account, err error) {
	msg := client.NewMessage("blockchain", types.EventGetLastHeader, nil)
	client.Send(msg, true)
	msg, err = client.Wait(msg)
	if err != nil {
		return nil, err
	}
	get := types.StoreGet{}
	get.StateHash = msg.GetData().(*types.Header).GetStateHash()
	for i := 0; i < len(addrs); i++ {
		get.Keys = append(get.Keys, AccountKey(addrs[i]))
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

func LoadAccountsDB(db dbm.KVDB, addrs []string) (accs []*types.Account, err error) {
	for i := 0; i < len(addrs); i++ {
		acc := LoadAccount(db, addrs[i])
		accs = append(accs, acc)
	}
	return accs, nil
}

//address to save key
func AccountKey(address string) (key []byte) {
	key = append(key, []byte("mavl-acc-")...)
	key = append(key, []byte(address)...)
	return key
}
