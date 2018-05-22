package main

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/log"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	receiveAddr         = "1KcdP2GJeeob69hjg5kzR2mSwXTkXF5Kgv"
	rpcAddr             = "http://localhost:8801"
	currentHeight int64 = -1
	currentIndex  int64 = 0
)

func main() {
	log.SetLogLevel("error")
	fmt.Println("starting scaning.............")
	scanWrite()
}

func scanWrite() {
	for {
		time.Sleep(time.Second*5)
		rpc, err := jsonrpc.NewJSONClient(rpcAddr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		paramsReqAddr := types.ReqAddr{
			Addr:      receiveAddr,
			Flag:      2,
			Count:     0,
			Direction: 1,
			Height:    currentHeight,
		}
		if currentIndex != 0 {
			paramsReqAddr.Index = currentIndex
		}

		var replyTxInfos jsonrpc.ReplyTxInfos
		err = rpc.Call("Chain33.GetTxByAddr", paramsReqAddr, &replyTxInfos)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		last := &jsonrpc.ReplyTxInfo{
			Height: currentHeight,
			Index:  currentIndex,
		}
		var txHashes []string
		for _, tx := range replyTxInfos.TxInfos {
			txHashes = append(txHashes, tx.Hash)
			last = tx
		}
		currentHeight = last.Height
		currentIndex = last.Index
		for _, hash := range txHashes {
			paramsQuery := jsonrpc.QueryParm{Hash: hash}
			var transactionDetail jsonrpc.TransactionDetail
			err = rpc.Call("Chain33.QueryTransaction", paramsQuery, &transactionDetail)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			pl, ok := transactionDetail.Tx.Payload.(map[string]interface{})["Value"].(map[string]interface{})
			if !ok {
				fmt.Println(transactionDetail.Tx.Payload)
				fmt.Fprintln(os.Stderr, "not a coin action")
				continue
			}
			trans, ok := pl["Transfer"]
			if !ok {
				fmt.Fprintln(os.Stderr, "not a transfer action")
				continue
			}
			note := trans.(map[string]interface{})["note"].(string)
			var noteTx types.Transaction
			txBytes, err := common.FromHex(note)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			err = types.Decode(txBytes, &noteTx)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			amount := transactionDetail.Amount
			if amount < types.Coin {
				fmt.Fprintln(os.Stderr, "not enough fee")
				continue
			}
			userTx := &types.Transaction{
				Execer:  []byte("user.write"),
				Payload: noteTx.Payload,
			}
			userTx.To = account.ExecAddress("user.write").String()
			userTx.Fee, err = userTx.GetRealFee(types.MinFee)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			userTx.Nonce = rand.New(rand.NewSource(time.Now().UnixNano())).Int63()
			txHex := types.Encode(userTx)
			paramsReqSignRawTx := types.ReqSignRawTx{
				Addr:   receiveAddr,
				TxHex:  hex.EncodeToString(txHex),
				Expire: "0",
			}
			var signed string
			err = rpc.Call("Chain33.SignRawTx", paramsReqSignRawTx, &signed)
			paramsRaw := jsonrpc.RawParm{
				Data: signed,
			}
			var sent string
			err = rpc.Call("Chain33.SendTransaction", paramsRaw, &sent)
			fmt.Println(sent)
		}
	}
}
