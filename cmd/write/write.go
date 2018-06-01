package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/log"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	receiveAddr         = "1MHkgR4uUg1ksssR5NFzU6zkzyCqxqjg2Z"
	rpcAddr             = "http://localhost:8801"
	currentHeight int64 = 0
	currentIndex  int64 = 0
	heightFile          = "height.txt"
)

func main() {
	log.SetLogLevel("error")
	err := ioHeightAndIndex()
	if err != nil {
		fmt.Println("file err")
		return
	}
	fmt.Println("starting scaning.............")
	scanWrite()
}

func ioHeightAndIndex() error {
	if _, err := os.Stat(heightFile); os.IsNotExist(err) {
		f, _ := os.Create(heightFile)
		height := strconv.FormatInt(currentHeight, 10)
		index := strconv.FormatInt(currentIndex, 10)
		f.WriteString(height + " " + index)
		f.Close()
	}
	f, _ := os.OpenFile(heightFile, os.O_RDWR, 0666)
	defer f.Close()
	fileContent, err := ioutil.ReadFile(heightFile)
	if err != nil {
		fmt.Print(err)
		return err
	}
	str := string(fileContent)
	heights := strings.Split(str, " ")
	if len(heights) == 2 {
		currentHeight, _ = strconv.ParseInt(heights[0], 10, 64)
		currentIndex, _ = strconv.ParseInt(heights[1], 10, 64)
	} else {
		height := strconv.FormatInt(currentHeight, 10)
		index := strconv.FormatInt(currentIndex, 10)
		f.WriteString(height + " " + index)
	}
	return nil
}

func scanWrite() {
	for {
		time.Sleep(time.Second * 5)
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
				fmt.Fprintln(os.Stderr, "not a user data tx")
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
			if !strings.HasPrefix(string(noteTx.Execer), "user.") {
				fmt.Fprintln(os.Stderr, "not a user defined executor")
				continue
			}
			userTx := &types.Transaction{
				Execer:  noteTx.Execer,
				Payload: noteTx.Payload,
			}
			userTx.To = account.ExecAddress(string(noteTx.Execer))
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
			f, _ := os.OpenFile(heightFile, os.O_RDWR, 0666)
			height := strconv.FormatInt(currentHeight, 10)
			index := strconv.FormatInt(currentIndex, 10)
			f.WriteString(height + " " + index)
			f.Close()
			fmt.Println(sent)
		}
	}
}
