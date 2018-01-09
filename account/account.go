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

var genesisKey = []byte("mavl-acc-genesis")

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
	b := accFrom.GetBalance() - amount
	if b >= 0 {
		receiptBalanceFrom := &types.ReceiptBalance{accFrom.GetBalance(), b, -amount}
		accFrom.Balance = b
		tob := accTo.GetBalance() + amount
		receiptBalanceTo := &types.ReceiptBalance{accTo.GetBalance(), tob, amount}
		accTo.Balance = tob
		SaveAccount(db, accFrom)
		SaveAccount(db, accTo)
		return transferReceipt(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo), nil
	} else {
		return nil, types.ErrNoBalance
	}
}

func GenesisInit(db dbm.KVDB, addr string, amount int64) (*types.Receipt, error) {
	g := &types.Genesis{}
	g.Isrun = true
	SaveGenesis(db, g)
	accTo := LoadAccount(db, addr)
	tob := accTo.GetBalance() + amount
	receiptBalanceTo := &types.ReceiptBalance{accTo.GetBalance(), tob, amount}
	accTo.Balance = tob
	SaveAccount(db, accTo)
	receipt := genesisReceipt(accTo, receiptBalanceTo, g)
	return receipt, nil
}

func transferReceipt(accFrom, accTo *types.Account, receiptBalanceFrom, receiptBalanceTo *types.ReceiptBalance) *types.Receipt {
	log1 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(receiptBalanceFrom)}
	log2 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(receiptBalanceTo)}
	kv := GetKVSet(accFrom)
	kv = append(kv, GetKVSet(accTo)...)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log1, log2}}
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

func GetGenesisKVSet(g *types.Genesis) (kvset []*types.KeyValue) {
	value := types.Encode(g)
	kvset = append(kvset, &types.KeyValue{genesisKey, value})
	return kvset
}

func LoadAccounts(q *queue.Queue, addrs []string) (accs []*types.Account, err error) {
	client := q.GetClient()
	//get current head ->
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
			accs = append(accs, &types.Account{})
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

type CacheDB struct {
	stateHash []byte
	q         queue.IClient
	cache     map[string][]byte
}

func NewCacheDB(q *queue.Queue, stateHash []byte) *CacheDB {
	return &CacheDB{stateHash, q.GetClient(), make(map[string][]byte)}
}

func (db *CacheDB) Get(key []byte) (value []byte, err error) {
	if value, ok := db.cache[string(key)]; ok {
		return value, nil
	}
	value, err = db.get(key)
	if err != nil {
		return nil, err
	}
	db.cache[string(key)] = value
	return value, err
}

func (db *CacheDB) Set(key []byte, value []byte) error {
	db.cache[string(key)] = value
	return nil
}

func (db *CacheDB) get(key []byte) (value []byte, err error) {
	query := &types.StoreGet{db.stateHash, [][]byte{key}}
	msg := db.q.NewMessage("store", types.EventStoreGet, query)
	db.q.Send(msg, true)
	resp, err := db.q.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	value = resp.GetData().(*types.StoreReplyValue).Values[0]
	if value == nil {
		return nil, types.ErrNotFound
	}
	return value, nil
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
