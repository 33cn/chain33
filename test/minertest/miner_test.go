package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

var conn *grpc.ClientConn
var random *rand.Rand
var privCold crypto.PrivKey      //冷钱包
var privMiner crypto.PrivKey     //热钱包
var privAutoMiner crypto.PrivKey //自动挖矿钱包

func init() {
	var err error
	conn, err = grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	privCold = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	privMiner = getprivkey("4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01")
	privAutoMiner = getprivkey("3257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01")
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func TestSendToAddress(t *testing.T) {
	returnaddr := account.PubKeyToAddress(privCold.PubKey().Bytes()).String()
	err := sendtoaddressWait(privMiner, returnaddr, 1e9)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestSetMinerStart(t *testing.T) {
	err := setAutoMining(1)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestSetMinerStop(t *testing.T) {
	err := setAutoMining(0)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestMinerBind(t *testing.T) {
	//随机生成一个地址，对privCold 执行 open ticket 操作，操作失败
	addr, priv := genaddress()
	err := sendtoaddressWait(privMiner, addr, 1e8)
	if err != nil {
		t.Error(err)
		return
	}
	returnaddr := account.PubKeyToAddress(privCold.PubKey().Bytes()).String()
	err = openticket(addr, returnaddr, priv)
	if err == nil {
		t.Error("no permit")
		return
	}
	//bind MinerAddress
	err = bindminer(addr, returnaddr, privCold)
	if err != nil {
		t.Error(err)
		return
	}
	printAccount(returnaddr, "ticket")
	//open tick user bindminer
	sourcelist, err := getMinerSourceList(addr)
	if err != nil {
		t.Error(err)
		return
	}
	if len(sourcelist) == 0 || sourcelist[0] != returnaddr {
		t.Error("getMinerSourceList error")
		return
	}
	err = openticket(addr, returnaddr, priv)
	if err != nil {
		t.Error(err)
		return
	}
	printAccount(returnaddr, "ticket")
	tlist, err := getMineredTicketList(addr, 1)
	if err != nil {
		t.Error(err)
		return
	}
	for _, ticket := range tlist {
		t.Log(ticket.TicketId)
	}
}

func TestAutoClose(t *testing.T) {
	//取出已经miner的列表
	addr := account.PubKeyToAddress(privMiner.PubKey().Bytes()).String()
	t.Log(addr)
	tlist, err := getMineredTicketList(addr, 2)
	if err != nil {
		t.Error(err)
		return
	}
	now := time.Now().Unix()
	var ids []string
	for i := 0; i < len(tlist); i++ {
		if now-tlist[i].CreateTime > types.TicketWithdrawTime {
			ids = append(ids, tlist[i].TicketId)
		}
	}
	if len(ids) > 0 {
		for i := 0; i < len(ids); i++ {
			t.Log("close ticket", i, ids[i])
		}
		err := closeTickets(privMiner, ids)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func openticket(mineraddr, returnaddr string, priv crypto.PrivKey) error {
	ta := &types.TicketAction{}
	topen := &types.TicketOpen{MinerAddress: mineraddr, ReturnAddress: returnaddr, Count: 1}
	ta.Value = &types.TicketAction_Topen{topen}
	ta.Ty = types.TicketActionOpen
	err := sendTransactionWait(ta, []byte("ticket"), priv, "")
	if err != nil {
		return err
	}
	return nil
}

func bindminer(mineraddr, returnaddr string, priv crypto.PrivKey) error {
	ta := &types.TicketAction{}
	tbind := &types.TicketBind{MinerAddress: mineraddr, ReturnAddress: returnaddr}
	ta.Value = &types.TicketAction_Tbind{tbind}
	ta.Ty = types.TicketActionBind
	err := sendTransactionWait(ta, []byte("ticket"), priv, "")
	if err != nil {
		return err
	}
	return nil
}

//通过rpc 精选close 操作
func closeTickets(priv crypto.PrivKey, ids []string) error {
	for i := 0; i < len(ids); i += 100 {
		end := i + 100
		if end > len(ids) {
			end = len(ids)
		}
		ta := &types.TicketAction{}
		tclose := &types.TicketClose{ids[i:end]}
		ta.Value = &types.TicketAction_Tclose{tclose}
		ta.Ty = types.TicketActionClose
		_, err := sendTransaction(ta, []byte("ticket"), priv, "")
		if err != nil {
			return err
		}
	}
	return nil
}

//通rpc 进行query
func getMineredTicketList(addr string, status int32) ([]*types.Ticket, error) {
	reqaddr := &types.TicketList{addr, status}
	var req types.Query
	req.Execer = []byte("ticket")
	req.FuncName = "TicketList"
	req.Payload = types.Encode(reqaddr)
	c := types.NewGrpcserviceClient(conn)
	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		return nil, err
	}
	if !reply.IsOk {
		return nil, errors.New(string(reply.GetMsg()))
	}
	//decode
	var list types.ReplyTicketList
	err = types.Decode(reply.GetMsg(), &list)
	if err != nil {
		return nil, err
	}
	return list.Tickets, nil
}

func printAccount(addr string, execer string) {
	account, err := getBalance(addr, execer)
	if err != nil {
		panic(err)
	}
	fmt.Println("addr:", account.Addr, "balance:", account.Balance, "frozen:", account.Frozen)
}

func getBalance(addr string, execer string) (*types.Account, error) {
	reqbalance := &types.ReqBalance{Addresses: []string{addr}, Execer: execer}
	c := types.NewGrpcserviceClient(conn)
	reply, err := c.GetBalance(context.Background(), reqbalance)
	if err != nil {
		return nil, err
	}
	accs := reply.GetAcc()
	return accs[0], nil
}

func genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := account.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}

func sendtoaddressWait(priv crypto.PrivKey, to string, amount int64) error {
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	hash, err := sendTransaction(transfer, []byte("coins"), priv, to)
	if err != nil {
		return err
	}
	txinfo := waitTx(hash)
	if txinfo.Receipt.Ty != types.ExecOk {
		return errors.New("sendtoaddressWait error")
	}
	return nil
}

func sendtoaddress(priv crypto.PrivKey, to string, amount int64) ([]byte, error) {
	//defer conn.Close()
	//fmt.Println("sign key privkey: ", common.ToHex(priv.Bytes()))
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	return sendTransaction(transfer, []byte("coins"), priv, to)
}

func sendTransactionWait(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (err error) {
	hash, err := sendTransaction(payload, execer, priv, to)
	if err != nil {
		return err
	}
	txinfo := waitTx(hash)
	if txinfo.Receipt.Ty != types.ExecOk {
		return errors.New("sendTransactionWait error")
	}
	return nil
}

func getMinerSourceList(addr string) ([]string, error) {
	reqaddr := &types.ReqString{addr}
	var req types.Query
	req.Execer = []byte("ticket")
	req.FuncName = "MinerSourceList"
	req.Payload = types.Encode(reqaddr)

	c := types.NewGrpcserviceClient(conn)
	reply, err := c.QueryChain(context.Background(), &req)
	if err != nil {
		return nil, err
	}
	if !reply.IsOk {
		return nil, errors.New(string(reply.GetMsg()))
	}
	var sourcelist types.ReplyStrings
	err = types.Decode(reply.GetMsg(), &sourcelist)
	if err != nil {
		return nil, err
	}
	return sourcelist.Datas, nil
}

func sendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error) {
	if to == "" {
		to = account.ExecAddress(string(execer)).String()
	}
	tx := &types.Transaction{Execer: execer, Payload: types.Encode(payload), Fee: 1e6, To: to}
	tx.Nonce = random.Int63()
	tx.Fee, err = tx.GetRealFee()
	if err != nil {
		return nil, err
	}
	tx.Fee += types.MinFee
	tx.Sign(types.SECP256K1, priv)

	// Contact the server and print out its response.
	c := types.NewGrpcserviceClient(conn)
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return nil, err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return nil, errors.New(string(reply.GetMsg()))
	}
	return tx.Hash(), nil
}

func setAutoMining(flag int32) (err error) {
	req := &types.MinerFlag{flag}
	c := types.NewGrpcserviceClient(conn)
	reply, err := c.SetAutoMining(context.Background(), req)
	if err != nil {
		return err
	}
	if reply.IsOk {
		return nil
	}
	return errors.New(string(reply.Msg))
}

func waitTx(hash []byte) *types.TransactionDetail {
	c := types.NewGrpcserviceClient(conn)
	reqhash := &types.ReqHash{hash}
	for {
		res, err := c.QueryTransaction(context.Background(), reqhash)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
		}
		if res != nil {
			return res
		}
	}
}

func getlastheader() (*types.Header, error) {
	c := types.NewGrpcserviceClient(conn)
	v := &types.ReqNil{}
	return c.GetLastHeader(context.Background(), v)
}
