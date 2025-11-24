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
	"github.com/stretchr/testify/require"
)

func addMainTx(cfg *types.Chain33Config, priv crypto.PrivKey, api client.QueueProtocolAPI) (string, error) {
	txs := util.GenCoinsTxs(cfg, priv, 1)
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

// 构造单笔para交易
func addSingleParaTx(cfg *types.Chain33Config, priv crypto.PrivKey, api client.QueueProtocolAPI, exec string) (string, error) {
	tx := util.CreateTxWithExecer(cfg, priv, exec)
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

// 构造para交易组
func addGroupParaTx(cfg *types.Chain33Config, priv crypto.PrivKey, api client.QueueProtocolAPI, title string, haveMainTx bool) (string, *types.ReplyStrings, error) {
	var tx0 *types.Transaction
	if haveMainTx {
		tx0 = util.CreateTxWithExecer(cfg, priv, "coins")
	} else {
		tx0 = util.CreateTxWithExecer(cfg, priv, title+"coins")
	}
	tx1 := util.CreateTxWithExecer(cfg, priv, title+"token")
	tx2 := util.CreateTxWithExecer(cfg, priv, title+"trade")
	tx3 := util.CreateTxWithExecer(cfg, priv, title+"evm")
	tx4 := util.CreateTxWithExecer(cfg, priv, title+"none")

	var txs types.Transactions
	txs.Txs = append(txs.Txs, tx0)
	txs.Txs = append(txs.Txs, tx1)
	txs.Txs = append(txs.Txs, tx2)
	txs.Txs = append(txs.Txs, tx3)
	txs.Txs = append(txs.Txs, tx4)
	feeRate := cfg.GetMinTxFeeRate()
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
	chainlog.Debug("TestGetParaTxByTitle begin --------------------")

	//构造十个区块
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10

	_, err := blockchain.GetBlock(curheight)
	if err != nil {
		require.NoError(t, err)
	}
	cfg := mock33.GetClient().GetConfig()
	for {
		_, err = addMainTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI())
		require.NoError(t, err)

		_, err = addSingleParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.hyb.none")
		require.NoError(t, err)

		_, _, err = addGroupParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.hyb.", false)

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
	//通过seq获取para交易
	req.Start = 0
	req.End = curheight
	req.Title = "user.p.hyb."
	req.IsSeq = true
	testgetParaTxByTitle(t, blockchain, &req, 0)

	//通过height获取para交易
	req.IsSeq = false
	testgetParaTxByTitle(t, blockchain, &req, 0)

	//通过height获取para交易,End大于当前最新高度报错
	req.IsSeq = false
	req.End = curheight + 10
	testgetParaTxByTitle(t, blockchain, &req, 1)

	curheight = blockchain.GetBlockHeight()
	var height int64
	for height = 0; height <= curheight; height++ {
		testGetParaTxByHeight(cfg, t, blockchain, height)
	}
	//异常测试
	req.Start = 3
	req.End = 2
	_, err = blockchain.GetParaTxByTitle(&req)
	require.Equal(t, err, types.ErrEndLessThanStartHeight)

	req.Start = 1
	req.End = 1
	req.Title = "user.write"
	_, err = blockchain.GetParaTxByTitle(&req)
	require.Equal(t, err, types.ErrInvalidParam)

	//title 对应的交易不存在
	req.Start = 1
	req.End = 1
	req.Title = "user.p.write"
	paratxs, err := blockchain.GetParaTxByTitle(&req)
	require.NoError(t, err)
	require.NotNil(t, paratxs)
	for _, paratx := range paratxs.Items {
		require.Equal(t, types.AddBlock, paratx.Type)
		require.Nil(t, paratx.TxDetails)
		require.Nil(t, paratx.ChildHash)
		require.Nil(t, paratx.Proofs)
		require.Equal(t, uint32(0), paratx.Index)
	}
	chainlog.Debug("TestGetParaTxByTitle end --------------------")
}
func testgetParaTxByTitle(t *testing.T, blockchain *blockchain.BlockChain, req *types.ReqParaTxByTitle, flag int) {
	count := req.End - req.Start + 1
	ParaTxDetails, err := blockchain.GetParaTxByTitle(req)
	if flag == 0 {
		require.NoError(t, err)
	}
	if flag == 1 {
		require.Equal(t, err, types.ErrInvalidParam)
		return
	}
	itemsLen := len(ParaTxDetails.Items)
	require.Equal(t, count, int64(itemsLen))

	for i, txDetail := range ParaTxDetails.Items {
		if txDetail != nil {
			require.Equal(t, txDetail.Header.Height, req.Start+int64(i))
			//chainlog.Debug("testgetParaTxByTitle:", "Height", txDetail.Header.Height)
			for _, tx := range txDetail.TxDetails {
				if tx != nil {
					execer := string(tx.Tx.Execer)
					if !strings.HasPrefix(execer, "user.p.hyb.") && tx.Tx.GetGroupCount() != 0 {
						//chainlog.Debug("testgetParaTxByTitle:maintxingroup", "tx", tx)
						require.Equal(t, tx.Receipt.Ty, int32(types.ExecOk))
					} else {
						require.Equal(t, tx.Receipt.Ty, int32(types.ExecPack))
					}
					if tx.Proofs != nil {
						roothash := merkle.GetMerkleRootFromBranch(tx.Proofs, tx.Tx.Hash(), tx.Index)
						ok := bytes.Equal(roothash, txDetail.Header.GetHash())
						require.Equal(t, ok, false)
					}
				}
			}
		}
	}
}

// 获取当前高度上的所有平行链title
func testGetParaTxByHeight(cfg *types.Chain33Config, t *testing.T, blockchain *blockchain.BlockChain, height int64) {

	block, err := blockchain.GetBlock(height)
	require.NoError(t, err)

	_, err = blockchain.LoadParaTxByHeight(-1, "", 0, 1)
	require.Equal(t, types.ErrInvalidParam, err)

	_, err = blockchain.LoadParaTxByHeight(height, "user.write", 0, 1)
	require.Equal(t, types.ErrInvalidParam, err)

	replyparaTxs, err := blockchain.LoadParaTxByHeight(height, "", 0, 1)
	if height == 0 {
		return
	}

	require.NoError(t, err)
	var mThreePhashes [][]byte
	for _, paratx := range replyparaTxs.Items {
		require.Equal(t, paratx.Height, height)
		require.Equal(t, paratx.Hash, block.Block.Hash(cfg))
		mThreePhashes = append(mThreePhashes, paratx.ChildHash)
	}
	//子roothash的proof证明
	for _, childchain := range replyparaTxs.Items {
		branch := merkle.GetMerkleBranch(mThreePhashes, childchain.GetChildHashIndex())
		root := merkle.GetMerkleRootFromBranch(branch, childchain.ChildHash, childchain.GetChildHashIndex())
		require.Equal(t, block.Block.TxHash, root)
	}
	rootHash := merkle.GetMerkleRoot(mThreePhashes)
	require.Equal(t, block.Block.TxHash, rootHash)
}
func TestMultiLayerMerkleTree(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	cfg := mock33.GetClient().GetConfig()
	blockchain := mock33.GetBlockChain()
	chainlog.Debug("TestMultiLayerMerkleTree begin --------------------")

	//构造十个区块
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10

	_, err := blockchain.GetBlock(curheight)
	if err != nil {
		require.NoError(t, err)
	}

	for {
		_, err = addMainTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI())
		require.NoError(t, err)

		_, err = addSingleParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.hyb.ticket")
		require.NoError(t, err)
		_, err = addSingleParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.test.coins")
		require.NoError(t, err)
		_, err = addSingleParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.para.game")
		require.NoError(t, err)

		_, _, err = addGroupParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.test.", false)
		require.NoError(t, err)

		_, _, err = addGroupParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.hyb.", false)
		require.NoError(t, err)

		_, _, err = addGroupParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.para.", false)
		require.NoError(t, err)
		curheight = blockchain.GetBlockHeight()
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}

	curheight = blockchain.GetBlockHeight()
	var height int64
	for height = 0; height <= curheight; height++ {
		testParaTxByHeight(cfg, t, blockchain, height)
	}
	chainlog.Debug("TestMultiLayerMerkleTree end --------------------")
}

// 获取当前高度上的所有平行链title
func testParaTxByHeight(cfg *types.Chain33Config, t *testing.T, blockchain *blockchain.BlockChain, height int64) {

	block, err := blockchain.GetBlock(height)
	require.NoError(t, err)
	merkleroothash := block.Block.GetTxHash()

	blockheight := block.Block.GetHeight()
	if cfg.IsPara() {
		blockheight = block.Block.GetMainHeight()
	}

	for txindex, tx := range block.Block.Txs {
		txProof, err := blockchain.ProcQueryTxMsg(tx.Hash())
		require.NoError(t, err)
		txhash := tx.Hash()
		if cfg.IsFork(blockheight, "ForkRootHash") {
			txhash = tx.FullHash()
		}

		//证明txproof的正确性,
		if txProof.GetProofs() != nil { //ForkRootHash 之前的proof证明
			brroothash := merkle.GetMerkleRootFromBranch(txProof.GetProofs(), txhash, uint32(txindex))
			require.Equal(t, merkleroothash, brroothash)
		} else if txProof.GetTxProofs() != nil { //ForkRootHash 之后的proof证明
			var childhash []byte
			for i, txproof := range txProof.GetTxProofs() {
				if i == 0 {
					childhash = merkle.GetMerkleRootFromBranch(txproof.GetProofs(), txhash, txproof.GetIndex())
					if txproof.GetRootHash() != nil {
						require.Equal(t, txproof.GetRootHash(), childhash)
					} else {
						require.Equal(t, txproof.GetIndex(), uint32(txindex))
						require.Equal(t, merkleroothash, childhash)
					}
				} else {
					brroothash := merkle.GetMerkleRootFromBranch(txproof.GetProofs(), childhash, txproof.GetIndex())
					require.Equal(t, merkleroothash, brroothash)
				}
			}
		}
	}
}
