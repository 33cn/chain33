// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package account

import (
	"testing"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

var (
	addr1 = "14ZTV2wHG3uPHnA5cBJmNxAxxvbzS7Z5mE"
	addr2 = "24ZTV2wHG3uPHnA5cBJmNxAxxvbzS7Z5mE"
	addr3 = "34ZTV2wHG3uPHnA5cBJmNxAxxvbzS7Z5mE"
	addr4 = "44ZTV2wHG3uPHnA5cBJmNxAxxvbzS7Z5mE"
)

func GenerAccDb() (*DB, *DB) {
	//构造账户数据库
	accCoin := NewCoinsAccount()
	stroedb, _ := db.NewGoMemDB("gomemdb", "test", 128)
	accCoin.SetDB(stroedb)

	accToken, _ := NewAccountDB("token", "test", nil)
	stroedb2, _ := db.NewGoMemDB("gomemdb", "test", 128)
	accToken.SetDB(stroedb2)

	return accCoin, accToken
}

func (acc *DB) GenerAccData() {
	// 加入账户
	account := &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addr1,
	}
	acc.SaveAccount(account)

	account.Balance = 900 * 1e8
	account.Addr = addr2
	acc.SaveAccount(account)

	account.Balance = 800 * 1e8
	account.Addr = addr3
	acc.SaveAccount(account)

	account.Balance = 700 * 1e8
	account.Addr = addr4
	acc.SaveAccount(account)
}

func TestCheckTransfer(t *testing.T) {
	accCoin, tokenCoin := GenerAccDb()
	accCoin.GenerAccData()
	tokenCoin.GenerAccData()

	err := accCoin.CheckTransfer(addr1, addr2, 10*1e8)
	require.NoError(t, err)

	err = tokenCoin.CheckTransfer(addr3, addr4, 10*1e8)
	require.NoError(t, err)
}

func TestTransfer(t *testing.T) {
	accCoin, tokenCoin := GenerAccDb()
	accCoin.GenerAccData()
	tokenCoin.GenerAccData()

	_, err := accCoin.Transfer(addr1, addr2, 10*1e8)
	require.NoError(t, err)
	t.Logf("Coin from addr balance [%d] to addr balance [%d]",
		accCoin.LoadAccount(addr1).Balance,
		accCoin.LoadAccount(addr2).Balance)
	require.Equal(t, int64(1000*1e8-10*1e8), accCoin.LoadAccount(addr1).Balance)
	require.Equal(t, int64(900*1e8+10*1e8), accCoin.LoadAccount(addr2).Balance)

	_, err = tokenCoin.Transfer(addr3, addr4, 10*1e8)
	require.NoError(t, err)

	t.Logf("token from addr balance [%d] to addr balance [%d]",
		tokenCoin.LoadAccount(addr3).Balance,
		tokenCoin.LoadAccount(addr4).Balance)
	require.Equal(t, int64(800*1e8-10*1e8), tokenCoin.LoadAccount(addr3).Balance)
	require.Equal(t, int64(700*1e8+10*1e8), tokenCoin.LoadAccount(addr4).Balance)
}

func TestDepositBalance(t *testing.T) {
	accCoin, tokenCoin := GenerAccDb()
	accCoin.GenerAccData()
	tokenCoin.GenerAccData()

	_, err := accCoin.depositBalance(addr1, 20*1e8)
	require.NoError(t, err)
	t.Logf("Coin deposit balance [%d]", accCoin.LoadAccount(addr1).Balance)
	require.Equal(t, int64(1000*1e8+20*1e8), accCoin.LoadAccount(addr1).Balance)

	_, err = tokenCoin.depositBalance(addr1, 30*1e8)
	require.NoError(t, err)
	t.Logf("token deposit balance [%d]", tokenCoin.LoadAccount(addr1).Balance)
	require.Equal(t, int64(1000*1e8+30*1e8), tokenCoin.LoadAccount(addr1).Balance)
}

func initEnv() queue.Queue {
	var q = queue.New("channel")
	return q
}

func initQueAPI(q queue.Queue) (client.QueueProtocolAPI, error) {
	return client.New(q.Client(), nil)
}

func blockchainProcess(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub("blockchain")
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetLastHeader:
				header := &types.Header{StateHash: []byte("111111111111111111111")}
				msg.Reply(client.NewMessage("account", types.EventHeader, header))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func storeProcess(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub("store")
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventStoreGet:
				//datas := msg.GetData().(*types.StoreGet)
				//fmt.Println("EventStoreGet data = %v", datas)

				values := make([][]byte, 2)
				account := &types.Account{
					Balance: 1000 * 1e8,
					Addr:    addr1,
				}
				value := types.Encode(account)
				values = append(values[:0], value)
				msg.Reply(client.NewMessage("", types.EventStoreGetReply, &types.StoreReplyValue{values}))
			case types.EventStoreGetTotalCoins:
				req := msg.GetData().(*types.IterateRangeByStateHash)
				resp := &types.ReplyGetTotalCoins{}
				resp.Count = req.Count
				msg.Reply(client.NewMessage("", types.EventGetTotalCoinsReply, resp))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func TestLoadAccounts(t *testing.T) {
	q := initEnv()
	qAPI, _ := initQueAPI(q)
	blockchainProcess(q)
	storeProcess(q)

	accCoin, _ := GenerAccDb()
	addrs := make([]string, 1)
	addrs[0] = addr1
	accs, err := accCoin.LoadAccounts(qAPI, addrs)
	require.NoError(t, err)
	t.Logf("LoadAccounts is %v", accs)
}

func TestGetTotalCoins(t *testing.T) {
	q := initEnv()
	qAPI, _ := initQueAPI(q)
	storeProcess(q)

	accCoin, _ := GenerAccDb()
	symbols := [2]string{"bty", ""}
	for _, symbol := range symbols {
		reqTotalCoin := &types.ReqGetTotalCoins{
			Symbol: symbol,
			Count:  100000,
		}
		rsp, err := accCoin.GetTotalCoins(qAPI, reqTotalCoin)
		require.NoError(t, err)
		t.Logf("GetTotalCoins is %v", rsp)
		require.Equal(t, int64(100000), rsp.Count)
	}
}

func TestAccountName(t *testing.T) {
	stroedb, _ := db.NewGoMemDB("gomemdb", "test", 128)

	accCoin := NewCoinsAccount()
	accCoin.SetDB(stroedb)
	coinsAddr := address.ExecAddress("coins")
	t.Log("coinsAddr:", coinsAddr)

	accToken, _ := NewAccountDB("token", "test", nil)
	accToken.SetDB(stroedb)
	tokenAddr := address.ExecAddress("token")
	t.Log("tokenAddr:", tokenAddr)

	tradeAddr := address.ExecAddress("trade")
	t.Log("tradeAddr:", tradeAddr)

	paraAddr := address.ExecAddress("paracross")
	t.Log("paraAddr:", paraAddr)

	myAddr := "13DP8mVru5Rtu6CrjXQMvLsjvve3epRR1i"
	t.Log("paraAddr:", accToken.ExecAddress("paracross"))
	t.Log("user.p.guodun.paraAddr:", accToken.ExecAddress("user.p.guodun.paracross"))
	fromAcc := accToken.LoadExecAccount(myAddr, paraAddr)
	t.Log("myAddr of paracorss", fromAcc)
}
