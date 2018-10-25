package executor

import (
	"strconv"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	ticketty "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	tokenty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	"gitlab.33.cn/chain33/chain33/queue"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	addr1 string
)

func initUnitEnv() (queue.Queue, *Executor) {
	var q = queue.New("channel")
	cfg, sub := config.InitCfg("../cmd/chain33/chain33.test.toml")
	exec := New(cfg.Exec, sub.Exec)
	exec.SetQueueClient(q.Client())
	return q, exec
}

func createTxEx(priv crypto.PrivKey, to string, amount int64, ty int32, execer string) *types.Transaction {
	var tx *types.Transaction
	switch execer {
	case "coins":
		v := &cty.CoinsAction_Transfer{&types.AssetsTransfer{Amount: amount}}
		transfer := &cty.CoinsAction{Value: v, Ty: ty}
		tx = &types.Transaction{Execer: []byte(execer), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	case "none":
		v := &cty.CoinsAction_Transfer{&types.AssetsTransfer{Amount: amount}}
		transfer := &cty.CoinsAction{Value: v, Ty: ty}
		tx = &types.Transaction{Execer: []byte(execer), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	case "ticket":
		transfer := &ticketty.TicketAction{}
		if ticketty.TicketActionGenesis == ty {
			v := &ticketty.TicketAction_Genesis{&ticketty.TicketGenesis{}}
			transfer.Value = v
			transfer.Ty = ty
		} else if ticketty.TicketActionOpen == ty {
			v := &ticketty.TicketAction_Topen{&ticketty.TicketOpen{}}
			transfer.Value = v
			transfer.Ty = ty
		} else if ticketty.TicketActionClose == ty {
			v := &ticketty.TicketAction_Tclose{&ticketty.TicketClose{}}
			transfer.Value = v
			transfer.Ty = ty
		} else if ticketty.TicketActionMiner == ty {
			v := &ticketty.TicketAction_Miner{&ticketty.TicketMiner{}}
			transfer.Value = v
			transfer.Ty = ty
		} else if ticketty.TicketActionBind == ty {
			v := &ticketty.TicketAction_Tbind{&ticketty.TicketBind{}}
			transfer.Value = v
			transfer.Ty = ty
		} else {
			return nil
		}
		tx = &types.Transaction{Execer: []byte(execer), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	case "token":
		transfer := &tokenty.TokenAction{}
		if tokenty.ActionTransfer == ty {
			v := &tokenty.TokenAction_Transfer{Transfer: &types.AssetsTransfer{Cointoken: "GOOD", Amount: 1e6}}
			transfer.Value = v
			transfer.Ty = ty
		} else if tokenty.ActionWithdraw == ty {
			v := &tokenty.TokenAction_Withdraw{Withdraw: &types.AssetsWithdraw{Cointoken: "GOOD", Amount: 1e6}}
			transfer.Value = v
			transfer.Ty = ty
		} else if tokenty.TokenActionPreCreate == ty {
			v := &tokenty.TokenAction_TokenPreCreate{&tokenty.TokenPreCreate{
				"Yuan chain coin",
				"GOOD",
				"An Easy Way to Build Blockchain",
				10000 * 10000 * 100,
				1 * 1e8,
				"1Lmmwzw6ywVa3UZpA4tHvCB7gR9ZKRwpom"}} // 该处指针不能为空否则会有崩溃
			transfer.Value = v
			transfer.Ty = ty
		} else if tokenty.TokenActionFinishCreate == ty {
			v := &tokenty.TokenAction_TokenFinishCreate{&tokenty.TokenFinishCreate{}}
			transfer.Value = v
			transfer.Ty = ty
		} else if tokenty.TokenActionRevokeCreate == ty {
			v := &tokenty.TokenAction_TokenRevokeCreate{&tokenty.TokenRevokeCreate{}}
			transfer.Value = v
			transfer.Ty = ty
		} else {
			return nil
		}
		tx = &types.Transaction{Execer: []byte(execer), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	default:
		return nil
	}
	tx.Nonce = random.Int63()
	//tx.To = address.ExecAddress(execer).String()
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func genTxsEx(n int64, ty int32, execer string) (txs []*types.Transaction) {
	_, priv := genaddress()
	to, _ := genaddress()
	for i := 0; i < int(n); i++ {
		txs = append(txs, createTxEx(priv, to, types.Coin*(n+1), ty, execer))
	}
	return txs
}

func createBlockEx(n int64, ty int32, execer string) *types.Block {
	newblock := &types.Block{}
	newblock.Height = -1
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = genTxsEx(n, ty, execer)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

func constructionBlockDetail(block *types.Block, height int64, txcount int) *types.BlockDetail {
	var blockdetail types.BlockDetail

	blockdetail.Receipts = make([]*types.ReceiptData, txcount)
	for j := 0; j < txcount; j++ {
		ReceiptData := types.ReceiptData{Ty: 2}
		blockdetail.Receipts[j] = &ReceiptData
	}
	blockdetail.Block = block
	return &blockdetail
}

func genExecTxListMsg(client queue.Client, block *types.Block) queue.Message {
	list := &types.ExecTxList{zeroHash[:], block.Txs, block.BlockTime, block.Height, 0, false}
	msg := client.NewMessage("execs", types.EventExecTxList, list)
	return msg
}

func genExecCheckTxMsg(client queue.Client, block *types.Block) queue.Message {
	list := &types.ExecTxList{zeroHash[:], block.Txs, block.BlockTime, block.Height, 0, false}
	msg := client.NewMessage("execs", types.EventCheckTx, list)
	return msg
}

func genEventAddBlockMsg(client queue.Client, block *types.Block) queue.Message {
	blockDetail := constructionBlockDetail(block, 1, 1)
	msg := client.NewMessage("execs", types.EventAddBlock, blockDetail)
	return msg
}

func genEventDelBlockMsg(client queue.Client, block *types.Block) queue.Message {
	blockDetail := constructionBlockDetail(block, 1, 1)
	msg := client.NewMessage("execs", types.EventDelBlock, blockDetail)
	return msg
}

//"coins", "GetTxsByAddr",
func genEventBlockChainQueryMsg(client queue.Client, param []byte, strDriver string, strFunName string) queue.Message {
	blockChainQue := &types.ChainExecutor{strDriver, strFunName, zeroHash[:], param, nil}
	msg := client.NewMessage("execs", types.EventBlockChainQuery, blockChainQue)
	return msg
}

func storeProcess(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub("store")
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventStoreGet:
				datas := msg.GetData().(*types.StoreGet)
				//fmt.Println("EventStoreGet Keys[0] = %s", string(datas.Keys[0]))

				var value []byte
				if strings.Contains(string(datas.Keys[0]), "this type ***") { //这种type结构体走该分支
					item := &types.ConfigItem{
						Key:  "111111111111111",
						Addr: "2222222222222",
						Value: &types.ConfigItem_Arr{
							Arr: &types.ArrayConfig{}},
					}
					value = types.Encode(item)
				} else {
					account := &types.Account{
						Balance: 1000 * 1e8,
						Addr:    addr1,
					}
					value = types.Encode(account)
				}
				values := make([][]byte, 2)
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

func blockchainProcess(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub("blockchain")
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetLastHeader:
				header := &types.Header{StateHash: []byte("111111111111111111111"), Height: 1}
				msg.Reply(client.NewMessage("", types.EventHeader, header))
			case types.EventLocalGet:
				//fmt.Println("EventLocalGet rsp")
				msg.Reply(client.NewMessage("", types.EventLocalReplyValue, &types.LocalReplyValue{}))
			case types.EventLocalList:
				var values [][]byte
				msg.Reply(client.NewMessage("", types.EventLocalReplyValue, &types.LocalReplyValue{Values: values}))

			case types.EventLocalPrefixCount:
				msg.Reply(client.NewMessage("", types.EventLocalReplyValue, &types.Int64{Data: 0}))

			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func createBlockChainQueryRq(execer string, funcName string) proto.Message {
	switch execer {
	case "coins":
		{
			if funcName == "GetPrefixCount" || funcName == "GetAddrTxsCount" {
				in := &types.ReqKey{Key: []byte("14KEKbYtKKQm4wMthSK9J4La4nAiidGozt")}
				return in
			} else {
				reqAddr := &types.ReqAddr{}
				reqAddr.Addr, _ = genaddress()
				reqAddr.Flag = 0
				reqAddr.Count = 10
				reqAddr.Direction = 0
				reqAddr.Height = -1
				reqAddr.Index = 0

				return reqAddr
			}
		}
	case "norm":
		{
			in := &types.ReqString{Data: "this type ***"}
			return in
		}
	case "ticket":
		{
			if funcName == "TicketInfos" {
				info := &ticketty.TicketInfos{}
				return info
			} else if funcName == "TicketList" {
				list := &ticketty.TicketList{}
				return list
			} else if funcName == "MinerAddress" {
				reqaddr := &types.ReqString{}
				return reqaddr
			} else if funcName == "MinerSourceList" {
				reqaddr := &types.ReqString{}
				return reqaddr
			}
		}
	case "token":
		{
			switch funcName {
			//GetTokens,支持所有状态下的单个token，多个token，以及所有token的信息的查询
			case "GetTokens":
				reqtokens := &tokenty.ReqTokens{}
				return reqtokens
			case "GetTokenInfo":
				symbol := &types.ReqString{Data: "this type ***"}
				return symbol
			case "GetAddrReceiverforTokens":
				addrTokens := &tokenty.ReqAddrTokens{}
				return addrTokens
			case "GetAccountTokenAssets":
				req := &tokenty.ReqAccountTokenAssets{}
				return req
			}
		}
	default:
		return nil
	}

	return nil
}

func TestQueueClient(t *testing.T) {
	q, _ := initUnitEnv()
	storeProcess(q)
	blockchainProcess(q)

	var txNum int64 = 1
	var msgs []queue.Message
	var msg queue.Message
	var block *types.Block

	execTy := [][]string{{"coins", strconv.Itoa(cty.CoinsActionTransfer)},
		{"coins", strconv.Itoa(cty.CoinsActionWithdraw)},
		{"coins", strconv.Itoa(cty.CoinsActionGenesis)},
		{"none", strconv.Itoa(cty.CoinsActionTransfer)},
		{"ticket", strconv.Itoa(ticketty.TicketActionGenesis)},
		{"ticket", strconv.Itoa(ticketty.TicketActionOpen)},
		{"ticket", strconv.Itoa(ticketty.TicketActionBind)},
		{"ticket", strconv.Itoa(ticketty.TicketActionClose)},
		{"ticket", strconv.Itoa(ticketty.TicketActionMiner)},
		{"token", strconv.Itoa(tokenty.TokenActionPreCreate)},
		{"token", strconv.Itoa(tokenty.TokenActionFinishCreate)},
		{"token", strconv.Itoa(tokenty.TokenActionRevokeCreate)},
		{"token", strconv.Itoa(tokenty.ActionTransfer)},
		{"token", strconv.Itoa(tokenty.ActionWithdraw)},
	}

	for _, str := range execTy {
		ty, _ := strconv.Atoi(str[1])
		block = createBlockEx(txNum, int32(ty), str[0])
		addr1 = block.Txs[0].From() //将获取随机生成交易地址

		// 1、测试 EventExecTxList 消息
		msg = genExecTxListMsg(q.Client(), block)
		msgs = append(msgs, msg)

		//2、测试 EventAddBlock 消息
		msg = genEventAddBlockMsg(q.Client(), block)
		msgs = append(msgs, msg)

		// 3、测试 EventDelBlock 消息
		msg = genEventDelBlockMsg(q.Client(), block)
		msgs = append(msgs, msg)

		// 4、测试 EventCheckTx 消息
		msg = genExecCheckTxMsg(q.Client(), block)
		msgs = append(msgs, msg)
	}

	// 5、测试 EventBlockChainQuery 消息
	var reqAddr types.ReqAddr
	reqAddr.Addr, _ = genaddress()
	reqAddr.Flag = 0
	reqAddr.Count = 10
	reqAddr.Direction = 0
	reqAddr.Height = -1
	reqAddr.Index = 0

	execFunName := [][]string{{"coins", "GetAddrReciver"},
		{"coins", "GetTxsByAddr"},
		{"manage", "GetConfigItem"},
		{"norm", "NormGet"},
		{"ticket", "TicketInfos"},
		{"ticket", "TicketList"},
		{"ticket", "MinerAddress"},
		{"ticket", "MinerSourceList"},
		{"token", "GetTokens"},
		{"token", "GetTokenInfo"},
		{"token", "GetAddrReceiverforTokens"},
		{"token", "GetAccountTokenAssets"},
		{"coins", "GetPrefixCount"},
		{"coins", "GetAddrTxsCount"},
	}

	for _, str := range execFunName {
		t := createBlockChainQueryRq(str[0], str[1])
		if nil == t {
			continue
		}
		msg = genEventBlockChainQueryMsg(q.Client(), types.Encode(t), str[0], str[1]) //生成消息
		msgs = append(msgs, msg)
	}

	go func() {
		for _, msga := range msgs {
			q.Client().Send(msga, true)
			_, err := q.Client().Wait(msga)
			if err == nil || err == types.ErrNotFound || err == types.ErrEmpty {
				t.Logf("%v,%v", msga, err)
			} else {
				t.Error(err)
			}
		}
		q.Close()
	}()
	q.Start()
}
