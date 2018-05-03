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

	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/client"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var alog = log.New("module", "account")

// DB for account
type DB struct {
	db                   dbm.KV
	accountKeyPerfix     []byte
	execAccountKeyPerfix []byte
}

func NewCoinsAccount() *DB {
	return newAccountDB("mavl-coins-bty-")
}

func NewTokenAccount(symbol string, db dbm.KV) *DB {
	accDB := newAccountDB(fmt.Sprintf("mavl-token-%s-", symbol))
	accDB.SetDB(db)
	return accDB
}

func NewTokenAccountWithoutDB(symbol string) *DB {
	return newAccountDB(fmt.Sprintf("mavl-token-%s-", symbol))
}

func newAccountDB(prefix string) *DB {
	acc := &DB{}
	acc.accountKeyPerfix = []byte(prefix)
	acc.execAccountKeyPerfix = append([]byte(prefix), []byte("exec-")...)
	//alog.Warn("NewAccountDB", "prefix", prefix, "key1", string(acc.accountKeyPerfix), "key2", string(acc.execAccountKeyPerfix))
	return acc
}

func (acc *DB) SetDB(db dbm.KV) *DB {
	acc.db = db
	return acc
}

func (acc *DB) IsTokenAccount() bool {
	return "token" == string(acc.accountKeyPerfix[len("mavl-"):len("mavl-token")])
}

func (acc *DB) LoadAccount(addr string) *types.Account {
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

func (acc *DB) CheckTransfer(from, to string, amount int64) error {
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

func (acc *DB) Transfer(from, to string, amount int64) (*types.Receipt, error) {
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

		receiptBalanceFrom := &types.ReceiptAccountTransfer{
			Prev:    &copyfrom,
			Current: accFrom,
		}
		receiptBalanceTo := &types.ReceiptAccountTransfer{
			Prev:    &copyto,
			Current: accTo,
		}

		acc.SaveAccount(accFrom)
		acc.SaveAccount(accTo)
		return acc.transferReceipt(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo), nil
	}

	return nil, types.ErrNoBalance
}

func (acc *DB) depositBalance(execaddr string, amount int64) (*types.Receipt, error) {
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadAccount(execaddr)
	copyacc := *acc1
	acc1.Balance += amount
	receiptBalance := &types.ReceiptAccountTransfer{
		Prev:    &copyacc,
		Current: acc1,
	}
	acc.SaveAccount(acc1)
	ty := int32(types.TyLogDeposit)
	if acc.IsTokenAccount() {
		ty = types.TyLogTokenDeposit
	}
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

func (acc *DB) transferReceipt(accFrom, accTo *types.Account, receiptFrom, receiptTo proto.Message) *types.Receipt {
	ty := int32(types.TyLogTransfer)
	if acc.IsTokenAccount() {
		ty = types.TyLogTokenTransfer
	}
	log1 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(receiptFrom),
	}
	log2 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(receiptTo),
	}
	kv := acc.GetKVSet(accFrom)
	kv = append(kv, acc.GetKVSet(accTo)...)
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1, log2},
	}
}

func (acc *DB) SaveAccount(acc1 *types.Account) {
	set := acc.GetKVSet(acc1)
	for i := 0; i < len(set); i++ {
		acc.db.Set(set[i].GetKey(), set[i].Value)
	}
}

func (acc *DB) GetKVSet(acc1 *types.Account) (kvset []*types.KeyValue) {
	value := types.Encode(acc1)
	kvset = append(kvset, &types.KeyValue{
		Key:   acc.AccountKey(acc1.Addr),
		Value: value,
	})
	return kvset
}

func (acc *DB) LoadAccounts(client queue.Client, addrs []string) (accs []*types.Account, err error) {
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

// TODO:使用API的方式访问,暂时与LoadAccounts()共存,后续将删除LoadAccounts()
func (acc *DB) LoadAccountsAPI(api client.QueueProtocolAPI, addrs []string) (accs []*types.Account, err error) {
	header, err := api.GetLastHeader()
	if err != nil {
		return nil, err
	}
	get := types.StoreGet{StateHash: header.GetStateHash()}
	for i := 0; i < len(addrs); i++ {
		get.Keys = append(get.Keys, acc.AccountKey(addrs[i]))
	}

	values, err := api.StoreGet(&get)
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

func (acc *DB) LoadAccountsDB(addrs []string) (accs []*types.Account, err error) {
	for i := 0; i < len(addrs); i++ {
		acc1 := acc.LoadAccount(addrs[i])
		accs = append(accs, acc1)
	}
	return accs, nil
}

// AccountKey return the key of address in DB
func (acc *DB) AccountKey(address string) (key []byte) {
	key = append(key, acc.accountKeyPerfix...)
	key = append(key, []byte(address)...)
	return key
}

func (acc *DB) GetTotalCoins(client queue.Client, in *types.ReqGetTotalCoins) (reply *types.ReplyGetTotalCoins, err error) {
	req := types.IterateRangeByStateHash{}
	req.StateHash = in.StateHash
	req.Count = in.Count
	if in.Symbol == "bty" {
		if in.StartKey == nil {
			req.Start = []byte("mavl-coins-bty-")
		} else {
			req.Start = in.StartKey
		}
		req.End = []byte("mavl-coins-bty-exec")
	} else {
		if in.StartKey == nil {
			req.Start = []byte(fmt.Sprintf("mavl-token-%s-", in.Symbol))
		} else {
			req.Start = in.StartKey
		}
		req.End = []byte(fmt.Sprintf("mavl-token-%s-exec", in.Symbol))
	}

	msg := client.NewMessage("store", types.EventStoreGetTotalCoins, &req)
	client.Send(msg, true)
	msg, err = client.Wait(msg)
	if err != nil {
		return nil, err
	}
	reply = msg.Data.(*types.ReplyGetTotalCoins)
	return reply, nil
}

// TODO:暂时保留GetTotalCoins()接口,等到后续实现调整以后删除GetTotalCoins()接口实现
func (acc *DB) GetTotalCoinsAPI(api client.QueueProtocolAPI, in *types.ReqGetTotalCoins) (reply *types.ReplyGetTotalCoins, err error) {
	req := types.IterateRangeByStateHash{}
	req.StateHash = in.StateHash
	req.Count = in.Count
	if in.Symbol == "bty" {
		if in.StartKey == nil {
			req.Start = []byte("mavl-coins-bty-")
		} else {
			req.Start = in.StartKey
		}
		req.End = []byte("mavl-coins-bty-exec")
	} else {
		if in.StartKey == nil {
			req.Start = []byte(fmt.Sprintf("mavl-token-%s-", in.Symbol))
		} else {
			req.Start = in.StartKey
		}
		req.End = []byte(fmt.Sprintf("mavl-token-%s-exec", in.Symbol))
	}
	return api.StoreGetTotalCoins(&req)
}
