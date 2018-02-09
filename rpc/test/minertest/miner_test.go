package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
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

func TestAutoClose(t *testing.T) {
	//取出已经miner的列表
	addr := account.PubKeyToAddress(privMiner.PubKey().Bytes()).String()
	tlist, err := getMineredTicketList(addr)
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

//通过rpc 精选close 操作
func closeTickets(priv crypto.PrivKey, ids []string) error {
	ta := &types.TicketAction{}
	tclose := &types.TicketClose{ids}
	ta.Value = &types.TicketAction_Tclose{}
	ta.Ty = types.TicketActionClose
	return sendTransaction(ta, []byte("ticket"), priv, "")
}

//通rpc 进行query
func getMineredTicketList(addr string) ([]*types.Ticket, error) {
	reqaddr := &types.TicketList{addr, 2}
	var req types.Query
	req.Execer = []byte("ticket")
	req.FuncName = "TicketList"
	req.Payload = types.Encode(reqaddr)
	c := types.NewGrpcserviceClient(conn)
	reply, err := c.Query(context.Background(), &req)
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

func sendtoaddress(priv crypto.PrivKey, to string, amount int64) error {
	//defer conn.Close()
	//fmt.Println("sign key privkey: ", common.ToHex(priv.Bytes()))
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	return sendTransaction(transfer, []byte("coins"), priv, to)
}

func sendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) error {
	if to == "" {
		to = account.ExecAddress(string(execer)).String()
	}
	tx := &types.Transaction{Execer: execer, Payload: types.Encode(payload), Fee: 1e6, To: to}
	tx.Nonce = rand.Int63()
	tx.Sign(types.SECP256K1, priv)
	// Contact the server and print out its response.
	c := types.NewGrpcserviceClient(conn)
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func getlastheader() (*types.Header, error) {
	c := types.NewGrpcserviceClient(conn)
	v := &types.ReqNil{}
	return c.GetLastHeader(context.Background(), v)
}
