// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	coinstypes "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
	"github.com/BurntSushi/toml"
)

var (
	configPath    = flag.String("f", "write.toml", "configfile")
	receiveAddr   = "1MHkgR4uUg1ksssR5NFzU6zkzyCqxqjg2Z"
	rpcAddr       = "http://localhost:8801"
	currentHeight int64
	currentIndex  int64
	heightFile    = "height.txt"
)

//Config 配置
type Config struct {
	UserWriteConf *UserWriteConf
}

//UserWriteConf 用户配置
type UserWriteConf struct {
	ReceiveAddr   string
	CurrentHeight int64
	CurrentIndex  int64
	HeightFile    string
}

func initWrite() *Config {
	var cfg Config
	if _, err := toml.DecodeFile(*configPath, &cfg); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	return &cfg
}

func main() {
	cfg := initWrite()
	receiveAddr = cfg.UserWriteConf.ReceiveAddr
	currentHeight = cfg.UserWriteConf.CurrentHeight
	currentIndex = cfg.UserWriteConf.CurrentIndex
	heightFile = cfg.UserWriteConf.HeightFile
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
		f, innerErr := os.Create(heightFile)
		if innerErr != nil {
			return innerErr
		}
		height := strconv.FormatInt(currentHeight, 10)
		index := strconv.FormatInt(currentIndex, 10)
		f.WriteString(height + " " + index)
		f.Close()
	}
	f, err := os.OpenFile(heightFile, os.O_RDWR, 0666)
	if err != nil {
		return err
	}
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
		rpc, err := jsonclient.NewJSONClient(rpcAddr)
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

		var replyTxInfos rpctypes.ReplyTxInfos
		err = rpc.Call("Chain33.GetTxByAddr", paramsReqAddr, &replyTxInfos)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		last := &rpctypes.ReplyTxInfo{
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
			paramsQuery := rpctypes.QueryParm{Hash: hash}
			var transactionDetail rpctypes.TransactionDetail
			err = rpc.Call("Chain33.QueryTransaction", paramsQuery, &transactionDetail)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			if len(transactionDetail.Tx.Payload) == 0 {
				fmt.Fprintln(os.Stderr, "not a coin action")
				continue
			}
			action := &coinstypes.CoinsAction{}
			if err = types.Decode(transactionDetail.Tx.Payload, action); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				continue
			}
			if action.Ty != coinstypes.CoinsActionTransfer {
				fmt.Fprintln(os.Stderr, "not a transfer action")
				continue
			}
			var noteTx types.Transaction
			txBytes, err := common.FromHex(string(action.GetTransfer().Note))
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
			userTx.To = address.ExecAddress(string(noteTx.Execer))
			userTx.Fee, err = userTx.GetRealFee(types.GInt("MinFee"))
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
			rpc.Call("Chain33.SignRawTx", paramsReqSignRawTx, &signed)
			paramsRaw := rpctypes.RawParm{
				Data: signed,
			}
			var sent string
			rpc.Call("Chain33.SendTransaction", paramsRaw, &sent)
			f, err := os.OpenFile(heightFile, os.O_RDWR, 0666)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			height := strconv.FormatInt(currentHeight, 10)
			index := strconv.FormatInt(currentIndex, 10)
			f.WriteString(height + " " + index)
			f.Close()
			fmt.Println(sent)
		}
	}
}
