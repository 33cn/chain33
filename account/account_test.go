// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package account

import (
	"fmt"
	"testing"

	"strings"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
				msg.Reply(client.NewMessage("", types.EventStoreGetReply, &types.StoreReplyValue{Values: values}))
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

func TestGetExecBalance(t *testing.T) {
	key := "mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	addr := "1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	//prefix1 := "mavl-coins-bty-exec-"
	//prefix2 := "mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:"

	/*req := &types.StoreList {
		StateHash: []byte("0000000000000000"),
		Start: []byte(prefix1),
		End: []byte(addr),
		Count: 2,
		Mode: 2,
	}*/

	fmt.Println("TestGetExecBalance---------------test case 1_1------------------------")
	in := &types.ReqGetExecBalance{
		Symbol:    "bty",
		StateHash: []byte("0000000000"),
		Addr:      []byte(addr),
		Execer:    "coins",
	}

	in.Count = 2
	reply, err := getExecBalance(storeList1_1, in)
	assert.Nil(t, err)
	assert.Equal(t, int64(4), reply.Amount)
	assert.Equal(t, int64(2), reply.AmountFrozen)
	assert.Equal(t, int64(2), reply.AmountActive)
	assert.Equal(t, 2, len(reply.Items))
	assert.Equal(t, len([]byte(key)), len(reply.NextKey))
	fmt.Println("TestGetExecBalance---------------test case 1_2------------------------")
	reply, err = getExecBalance(storeList1_2, in)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), reply.Amount)
	assert.Equal(t, int64(1), reply.AmountFrozen)
	assert.Equal(t, int64(1), reply.AmountActive)
	assert.Equal(t, 1, len(reply.Items))
	assert.Equal(t, 0, len(reply.NextKey))

	fmt.Println("TestGetExecBalance---------------test case 2------------------------")
	in.Count = 3
	reply, err = getExecBalance(storeList2, in)
	assert.Nil(t, err)
	assert.Equal(t, int64(6), reply.Amount)
	assert.Equal(t, int64(3), reply.AmountFrozen)
	assert.Equal(t, int64(3), reply.AmountActive)
	assert.Equal(t, 3, len(reply.Items))
	assert.Equal(t, 0, len(reply.NextKey))

	fmt.Println("TestGetExecBalance---------------test case 3------------------------")
	in.ExecAddr = []byte("16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp")
	reply, err = getExecBalance(storeList3, in)
	assert.NotNil(t, err)

	fmt.Println("TestGetExecBalance---------------test case 4------------------------")
	in.ExecAddr = []byte("26htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp")
	reply, err = getExecBalance(storeList3, in)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), reply.Amount)
	assert.Equal(t, int64(1), reply.AmountFrozen)
	assert.Equal(t, int64(1), reply.AmountActive)
	assert.Equal(t, 1, len(reply.Items))
}

func cloneByte(v []byte) []byte {
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func storeList1_1(req *types.StoreList) (reply *types.StoreListReply, err error) {
	reply = &types.StoreListReply{
		Start: req.Start,
		End:   req.End,
		Count: req.Count,
		Mode:  req.Mode,
	}
	var acc = &types.Account{
		Currency: 0,
		Balance:  1,
		Frozen:   1,
		Addr:     string(req.End),
	}
	value := types.Encode(acc)

	key1 := []byte("mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")
	key2 := []byte("mavl-coins-bty-exec-26htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")
	key3 := []byte("mavl-coins-bty-exec-36htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")
	reply.Num = 2
	reply.Keys = append(reply.Keys, key1)
	reply.Keys = append(reply.Keys, key2)
	reply.Values = append(reply.Values, cloneByte(value))
	reply.Values = append(reply.Values, cloneByte(value))
	reply.NextKey = key3
	return reply, nil
}

func storeList1_2(req *types.StoreList) (reply *types.StoreListReply, err error) {
	reply = &types.StoreListReply{
		Start: req.Start,
		End:   req.End,
		Count: req.Count,
		Mode:  req.Mode,
	}
	var acc = &types.Account{
		Currency: 0,
		Balance:  1,
		Frozen:   1,
		Addr:     string(req.End),
	}
	value := types.Encode(acc)
	//key1 := []byte("mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")
	//key2 := []byte("mavl-coins-bty-exec-26htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")
	key3 := []byte("mavl-coins-bty-exec-36htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")
	reply.Num = 1
	reply.Keys = append(reply.Keys, key3)
	reply.Values = append(reply.Values, cloneByte(value))
	return reply, nil
}

func storeList2(req *types.StoreList) (reply *types.StoreListReply, err error) {
	reply = &types.StoreListReply{
		Start: req.Start,
		End:   req.End,
		Count: req.Count,
		Mode:  req.Mode,
	}
	var acc = &types.Account{
		Currency: 0,
		Balance:  1,
		Frozen:   1,
		Addr:     string(req.End),
	}
	value := types.Encode(acc)

	key1 := []byte("mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")
	key2 := []byte("mavl-coins-bty-exec-26htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")
	key3 := []byte("mavl-coins-bty-exec-36htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")

	reply.Num = 3
	reply.Keys = append(reply.Keys, key1)
	reply.Keys = append(reply.Keys, key2)
	reply.Keys = append(reply.Keys, key3)
	reply.Values = append(reply.Values, cloneByte(value))
	reply.Values = append(reply.Values, cloneByte(value))
	reply.Values = append(reply.Values, cloneByte(value))

	return reply, nil
}

func storeList3(req *types.StoreList) (reply *types.StoreListReply, err error) {
	reply = &types.StoreListReply{
		Start: req.Start,
		End:   req.End,
		Count: req.Count,
		Mode:  req.Mode,
	}
	var acc = &types.Account{
		Currency: 0,
		Balance:  1,
		Frozen:   1,
		Addr:     string(req.End),
	}
	value := types.Encode(acc)

	//key1 := []byte("mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")
	key2 := []byte("mavl-coins-bty-exec-26htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")
	//key3 := []byte("mavl-coins-bty-exec-36htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP")

	reply.Num = 1
	//reply.Keys = append(reply.Keys, key1)
	reply.Keys = append(reply.Keys, key2)
	//reply.Keys = append(reply.Keys, key3)
	//reply.Values = append(reply.Values, cloneByte(value))
	reply.Values = append(reply.Values, cloneByte(value))
	//reply.Values = append(reply.Values, cloneByte(value))

	return reply, nil
}

func getExecBalance(callback func(*types.StoreList) (*types.StoreListReply, error), in *types.ReqGetExecBalance) (reply *types.ReplyGetExecBalance, err error) {
	req := &types.StoreList{}
	req.StateHash = in.StateHash

	prefix := symbolExecPrefix(in.Execer, in.Symbol)
	if len(in.ExecAddr) > 0 {
		prefix = prefix + "-" + string(in.ExecAddr) + ":"
	} else {
		prefix = prefix + "-"
	}

	req.Start = []byte(prefix)
	req.End = in.Addr
	req.Mode = 2 //1：为[start,end）模式，按前缀或者范围进行查找。2：为prefix + suffix遍历模式，先按前缀查找，再判断后缀是否满足条件。
	req.Count = in.Count

	if len(in.NextKey) > 0 {
		req.Start = in.NextKey
	}

	reply = &types.ReplyGetExecBalance{}
	fmt.Println("DB.GetExecBalance", "hash", common.ToHex(req.StateHash), "Prefix", string(req.Start), "Addr", string(req.End))

	res, err := callback(req)
	fmt.Println("send one req....")
	if err != nil {
		err = types.ErrTypeAsset
		return nil, err
	}

	for i := 0; i < len(res.Keys); i++ {
		strKey := string(res.Keys[i])
		fmt.Println("DB.GetExecBalance process one record", "key", strKey)
		if !strings.HasPrefix(strKey, prefix) {
			fmt.Println("accountDB.GetExecBalance key does not match prefix", "key:", strKey, " prefix:", prefix)
			return nil, types.ErrTypeAsset
		}
		//如果prefix形如：mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:  ,则是查找addr在一个合约地址上的余额，找到一个值即可结束。
		if strings.HasSuffix(prefix, ":") {
			addr := strKey[len(prefix):]
			execAddr := []byte(prefix[(len(prefix) - len(addr) - 1):(len(prefix) - 1)])
			fmt.Println("DB.GetExecBalance record for specific exec addr.", "execAddr:", string(execAddr), " addr:", string(addr))
			reply.AddItem(execAddr, res.Values[i])
		} else {
			combinAddr := strKey[len(prefix):]
			addrs := strings.Split(combinAddr, ":")
			if 2 != len(addrs) {
				fmt.Println("accountDB.GetExecBalance key does not contain exec-addr & addr", "key", strKey, "combinAddr", combinAddr)
				return nil, types.ErrTypeAsset
			}
			fmt.Println("DB.GetExecBalance", "execAddr", string(addrs[0]), "addr", string(addrs[1]))
			reply.AddItem([]byte(addrs[0]), res.Values[i])
		}
	}

	reply.NextKey = res.NextKey

	return reply, nil
}

func TestGetExecBalance2(t *testing.T) {
	accCoin := NewCoinsAccount()
	key := "mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
	execAddr := "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"
	addr := "1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"

	fmt.Println("-------------TestGetExecBalance2---test case1---")
	api := new(mocks.QueueProtocolAPI)
	in := &types.ReqGetExecBalance{}
	api.On("StoreList", mock.Anything).Return(&types.StoreListReply{}, nil)
	_, err := accCoin.GetExecBalance(api, in)
	assert.Nil(t, err)

	fmt.Println("-------------TestGetExecBalance2---test case2---")
	in.Symbol = "bty"
	in.Execer = "coins"
	in.Addr = []byte(addr)
	in.ExecAddr = []byte(execAddr)

	api.On("StoreList", mock.Anything).Return(&types.StoreListReply{}, nil)
	_, err = accCoin.GetExecBalance(api, in)
	assert.Nil(t, err)

	in.NextKey = []byte(key)
	fmt.Println("-------------TestGetExecBalance2---test case3---")
	api.On("StoreList", mock.Anything).Return(&types.StoreListReply{}, nil)
	_, err = accCoin.GetExecBalance(api, in)
	assert.Nil(t, err)

	fmt.Println("-------------TestGetExecBalance2---test case4---")
	api = new(mocks.QueueProtocolAPI)
	api.On("StoreList", mock.Anything).Return(nil, types.ErrInvalidParam)
	_, err = accCoin.GetExecBalance(api, in)
	assert.NotNil(t, err)

	/*
		fmt.Println("TestGetExecBalance---------------test case 1_1------------------------")
		in := &types.ReqGetExecBalance{
			Symbol:    "bty",
			StateHash: []byte("0000000000"),
			Addr:      []byte(addr),
			Execer:    "coins",
		}

		in.Count = 2
		reply, err := getExecBalance(storeList1_1, in)
		assert.Nil(t, err)
		assert.Equal(t, int64(4), reply.Amount)
		assert.Equal(t, int64(2), reply.AmountFrozen)
		assert.Equal(t, int64(2), reply.AmountActive)
		assert.Equal(t, 2, len(reply.Items))
		assert.Equal(t, len([]byte(key)), len(reply.NextKey))
		fmt.Println("TestGetExecBalance---------------test case 1_2------------------------")
		reply, err = getExecBalance(storeList1_2, in)
		assert.Nil(t, err)
		assert.Equal(t, int64(2), reply.Amount)
		assert.Equal(t, int64(1), reply.AmountFrozen)
		assert.Equal(t, int64(1), reply.AmountActive)
		assert.Equal(t, 1, len(reply.Items))
		assert.Equal(t, 0, len(reply.NextKey))

		fmt.Println("TestGetExecBalance---------------test case 2------------------------")
		in.Count = 3
		reply, err = getExecBalance(storeList2, in)
		assert.Nil(t, err)
		assert.Equal(t, int64(6), reply.Amount)
		assert.Equal(t, int64(3), reply.AmountFrozen)
		assert.Equal(t, int64(3), reply.AmountActive)
		assert.Equal(t, 3, len(reply.Items))
		assert.Equal(t, 0, len(reply.NextKey))

		fmt.Println("TestGetExecBalance---------------test case 3------------------------")
		in.ExecAddr = []byte("16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp")
		reply, err = getExecBalance(storeList3, in)
		assert.NotNil(t, err)

		fmt.Println("TestGetExecBalance---------------test case 4------------------------")
		in.ExecAddr = []byte("26htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp")
		reply, err = getExecBalance(storeList3, in)
		assert.Nil(t, err)
		assert.Equal(t, int64(2), reply.Amount)
		assert.Equal(t, int64(1), reply.AmountFrozen)
		assert.Equal(t, int64(1), reply.AmountActive)
		assert.Equal(t, 1, len(reply.Items))
	*/
}

func TestGetBalance(t *testing.T) {
	accCoin := NewCoinsAccount()
	addr := "1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"

	fmt.Println("-------------TestGetExecBalance2---test case1---")
	api := new(mocks.QueueProtocolAPI)
	in := &types.ReqBalance{}
	in.Addresses = append(in.Addresses, addr)
	api.On("StoreList", mock.Anything).Return(&types.StoreListReply{}, nil)
	api.On("GetLastHeader", mock.Anything).Return(&types.Header{StateHash: []byte("111111111111111111111")}, nil)
	api.On("StoreGet", mock.Anything).Return(&types.StoreReplyValue{Values: make([][]byte, 1)}, nil)
	_, err := accCoin.GetBalance(api, in)
	assert.Nil(t, err)

	fmt.Println("-------------TestGetExecBalance2---test case2---")
	in.StateHash = "111111111111111111111"
	_, err = accCoin.GetBalance(api, in)
	assert.Nil(t, err)

	fmt.Println("-------------TestGetExecBalance2---test case3---")
	in.Execer = "coins"
	//api.On("StoreList", mock.Anything).Return(nil, types.ErrInvalidParam)
	_, err = accCoin.GetBalance(api, in)
	t.Log(err)
	assert.Nil(t, err)
}

func TestDB_Mint(t *testing.T) {
	_, tokenCoin := GenerAccDb()
	tokenCoin.GenerAccData()

	_, err := tokenCoin.Mint(addr1, 10*1e8)
	require.NoError(t, err)
	t.Logf("Token mint addr balance [%d]", tokenCoin.LoadAccount(addr1).Balance)
	require.Equal(t, int64(1000*1e8+10*1e8), tokenCoin.LoadAccount(addr1).Balance)
}

func TestDB_Burn(t *testing.T) {
	_, tokenCoin := GenerAccDb()
	tokenCoin.GenerAccData()

	_, err := tokenCoin.Burn(addr1, 10*1e8)
	require.NoError(t, err)
	t.Logf("Token mint addr balance [%d]", tokenCoin.LoadAccount(addr1).Balance)
	require.Equal(t, int64(1000*1e8-10*1e8), tokenCoin.LoadAccount(addr1).Balance)
}
