// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/33cn/chain33/blockchain"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/merkle"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	types.Init("local", nil)
}

func addMainTx(priv crypto.PrivKey, api client.QueueProtocolAPI) (string, error) {
	txs := util.GenCoinsTxs(priv, 1)
	hash := common.ToHex(txs[0].Hash())
	reply, err := api.SendTx(txs[0])
	if err != nil {
		return hash, err
	}
	if !reply.GetIsOk() {
		return hash, errors.New("sendtx unknow error")
	}
	return hash, nil
}

//构造单笔para交易
func addSingleParaTx(priv crypto.PrivKey, api client.QueueProtocolAPI) (string, error) {
	tx := util.CreateTxWithExecer(priv, "user.p.hyb.none")

	hash := common.ToHex(tx.Hash())
	reply, err := api.SendTx(tx)
	if err != nil {
		return hash, err
	}
	if !reply.GetIsOk() {
		return hash, errors.New("sendtx unknow error")
	}
	return hash, nil
}

//构造para交易组
func addGroupParaTx(priv crypto.PrivKey, api client.QueueProtocolAPI, haveMainTx bool) (string, *types.ReplyStrings, error) {
	var tx0 *types.Transaction
	if haveMainTx {
		tx0 = util.CreateTxWithExecer(priv, "coins")
	} else {
		tx0 = util.CreateTxWithExecer(priv, "user.p.hyb.coins")
	}
	tx1 := util.CreateTxWithExecer(priv, "user.p.hyb.token")
	tx2 := util.CreateTxWithExecer(priv, "user.p.hyb.trade")
	tx3 := util.CreateTxWithExecer(priv, "user.p.hyb.evm")
	tx4 := util.CreateTxWithExecer(priv, "user.p.hyb.none")

	var txs types.Transactions
	txs.Txs = append(txs.Txs, tx0)
	txs.Txs = append(txs.Txs, tx1)
	txs.Txs = append(txs.Txs, tx2)
	txs.Txs = append(txs.Txs, tx3)
	txs.Txs = append(txs.Txs, tx4)
	feeRate := types.GInt("MinFee")
	group, err := types.CreateTxGroup(txs.Txs, feeRate)
	if err != nil {
		chainlog.Error("addGroupParaTx", "err", err.Error())
		return "", nil, err
	}

	var txHashs types.ReplyStrings
	for i, tx := range group.Txs {
		group.SignN(i, int32(types.SECP256K1), priv)
		txhash := common.ToHex(tx.Hash())
		txHashs.Datas = append(txHashs.Datas, txhash)

	}

	newtx := group.Tx()

	hash := common.ToHex(newtx.Hash())
	reply, err := api.SendTx(newtx)
	if err != nil {
		return "", nil, err
	}
	if !reply.GetIsOk() {
		return "", nil, errors.New("sendtx unknow error")
	}

	return hash, &txHashs, nil
}

func TestGetParaTxByTitle(t *testing.T) {
	//log.SetLogLevel("crit")
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	blockchain := mock33.GetBlockChain()
	chainlog.Info("TestGetParaTxByTitle begin --------------------")

	//构造十个区块
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10

	_, err := blockchain.GetBlock(curheight)
	if err != nil {
		require.NoError(t, err)
	}

	for {
		_, err = addMainTx(mock33.GetGenesisKey(), mock33.GetAPI())
		require.NoError(t, err)

		_, err = addSingleParaTx(mock33.GetGenesisKey(), mock33.GetAPI())
		require.NoError(t, err)

		//_, _, err = addGroupParaTx(mock33.GetGenesisKey(), mock33.GetAPI(), true)
		//require.NoError(t, err)

		_, _, err = addGroupParaTx(mock33.GetGenesisKey(), mock33.GetAPI(), false)
		require.NoError(t, err)

		curheight = blockchain.GetBlockHeight()
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	var req types.ReqParaTxByTitle
	req.Start = 0
	req.End = curheight
	req.Title = "user.p.hyb."
	testgetParaTxByTitle(t, blockchain, &req, false, false, nil)

	chainlog.Info("TestGetParaTxByTitle end --------------------")

}
func testgetParaTxByTitle(t *testing.T, blockchain *blockchain.BlockChain, req *types.ReqParaTxByTitle, isGroup bool, haveMainTx bool, hashs []string) {
	ParaTxDetails, err := blockchain.GetParaTxByTitle(req)
	require.NoError(t, err)

	for i, txDetail := range ParaTxDetails.Items {
		if txDetail != nil {
			assert.Equal(t, txDetail.Header.Height, req.Start+int64(i))
			//chainlog.Info("testgetParaTxByTitle:", "Height", txDetail.Header.Height)
			for _, tx := range txDetail.TxDetails {
				if tx != nil {
					execer := string(tx.Tx.Execer)
					if !strings.HasPrefix(execer, "user.p.hyb.") && tx.Tx.GetGroupCount() != 0 {
						//chainlog.Info("testgetParaTxByTitle:maintxingroup", "tx", tx)
						assert.Equal(t, tx.Receipt.Ty, int32(types.ExecOk))
					} else {
						assert.Equal(t, tx.Receipt.Ty, int32(types.ExecPack))
					}
					if tx.Proofs != nil {
						roothash := merkle.GetMerkleRootFromBranch(tx.Proofs, tx.Tx.Hash(), tx.Index)
						ok := bytes.Equal(roothash, txDetail.Header.GetHash())
						assert.Equal(t, ok, false)
					}
				}
			}
		}
	}
}
