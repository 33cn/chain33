package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"time"

	"gitlab.33.cn/chain33/chain33/common/log"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

const miner1 string = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
const miner2 string = "1EbDHAXpoiewjPLX9uqoz38HsKqMXayZrF"
const miner3 string = "1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa"

var (
	checkInterval int32 = 1
	minMinedBlock int32 = 1
	currentHeight int64 = 0
	currentIndex  int64 = 0
	rpcAddr             = "http://localhost:8801"
)

func main() {
	log.SetLogLevel("error")
	ioHeightAndIndex("miner1.txt")
	checkMiner(miner1, "miner1.txt")
	//go checkMiner(miner2)
	//go checkMiner(miner3)
}

func checkMiner(miner, fname string) {
	rpc, err := jsonrpc.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	paramsReqAddr := types.ReqAddr{
		Addr:      miner,
		Flag:      0,
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
		return
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
	fmt.Println(currentHeight)
	time.Sleep(time.Second * 5)
	for _, hash := range txHashes {
		paramsQuery := jsonrpc.QueryParm{Hash: hash}
		var transactionDetail jsonrpc.TransactionDetail
		err = rpc.Call("Chain33.QueryTransaction", paramsQuery, &transactionDetail)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if transactionDetail.Receipt.Ty != types.ExecOk {
			continue
		}
		pl, ok := transactionDetail.Tx.Payload.(map[string]interface{})["Value"].(map[string]interface{})
		if !ok {
			continue
		}
		miner, ok := pl["Miner"]
		if !ok {
			continue
		}
		fmt.Println(miner.(map[string]interface{})["ticketId"])
		f, _ := os.OpenFile(fname, os.O_WRONLY|os.O_APPEND, 0666)
		height := strconv.FormatInt(currentHeight, 10)
		index := strconv.FormatInt(currentIndex, 10)
		f.WriteString(height + " " + index + "\n")
		f.Close()
	}
}

func ioHeightAndIndex(fname string) error {
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		f, _ := os.Create(fname)
		height := strconv.FormatInt(currentHeight, 10)
		index := strconv.FormatInt(currentIndex, 10)
		f.WriteString(height + " " + index + "\n")
		f.Close()
	}
	f, err := os.OpenFile(fname, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	defer f.Close()
	rd := bufio.NewReader(f)
	fileContent := "0 0"
	for {
		line, _, err := rd.ReadLine()
		if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		if err == io.EOF {
			break
		}
		fileContent = string(line)
	}

	heights := strings.Split(fileContent, " ")
	if len(heights) == 2 {
		currentHeight, _ = strconv.ParseInt(heights[0], 10, 64)
		currentIndex, _ = strconv.ParseInt(heights[1], 10, 64)
	} else {
		height := strconv.FormatInt(currentHeight, 10)
		index := strconv.FormatInt(currentIndex, 10)
		f.WriteString(height + " " + index + "\n")
	}
	return nil
}
