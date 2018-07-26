package para

import (
	"bytes"
	"context"
	"encoding/hex"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/types/executor/paracross"
	"gitlab.33.cn/chain33/chain33/util"
)

const (
	waitMainBlocks = 2
)

type CommitMsg struct {
	mainBlockHash []byte
	initTxHashs   [][]byte
	blockDetail   *types.BlockDetail
}

type CommitMsgClient struct {
	paraClient         *ParaClient
	commitMsgNofity    chan *CommitMsg
	mainBlockNotify    chan *types.BlockDetail
	currentTx          string
	waitingTx          bool
	checkTxCommitTimes int
	quit               chan struct{}
}

func (client *CommitMsgClient) handler() {
	var notifications []*CommitMsg
	readTick := time.Tick(time.Second)
	consensusTick := time.Tick(time.Minute)
	type sendResult struct {
		err error
	}
	sendResponse := make(chan sendResult, 1)
	for {
		select {
		case msg, ok := <-client.commitMsgNofity:
			if !ok {
				continue
			}
			notifications = append(notifications, msg)

		case block := <-client.mainBlockNotify:
			if client.currentTx != "" {
				exist, err := checkTxInMainBlock(client.currentTx, block)
				if err != nil {
					continue
				}
				if exist {
					client.waitingTx = false
					client.currentTx = ""
					client.checkTxCommitTimes = 0
				} else {
					client.checkTxCommitTimes++
					if client.checkTxCommitTimes > waitMainBlocks {
						client.checkTxCommitTimes = 0
						go func() {
							err = client.sendCommitMsgTx(client.currentTx)
							if err != nil {
								sendResponse <- sendResult{err}
							}
						}()
					}
				}
			}

		case <-readTick:
			if len(notifications) != 0 && !client.waitingTx {
				rawTxs, count, err := calcRawTxs(notifications)
				if err != nil {
					continue
				}
				signTx, err := client.signCommitMsgTx(rawTxs)
				if err != nil || signTx == "" {
					continue
				}
				notifications = notifications[count:]
				client.currentTx = signTx
				client.waitingTx = true
				client.checkTxCommitTimes = 0
				go func() {
					err = client.sendCommitMsgTx(client.currentTx)
					if err != nil {
						sendResponse <- sendResult{err}
					}
				}()
			}
		case <-consensusTick:

		case <-sendResponse:
			go func() {
				err := client.sendCommitMsgTx(client.currentTx)
				if err != nil {
					sendResponse <- sendResult{err}
				}
			}()
		case <-client.quit:
			return
		}
	}
}

func calcRawTxs(notifications []*CommitMsg) (string, int, error) {
	txs, count, err := batchCalcTxGroup(notifications)
	if err != nil {
		txs, err = singleCalcTx((notifications)[0])
		if err != nil {
			plog.Error("single calc tx", "height", notifications[0].blockDetail.Block.Height)

			return "", 0, err
		}
		return txs, 1, nil
	}
	return txs, count, nil
}

func batchCalcTxGroup(notifications []*CommitMsg) (string, int, error) {
	var buff []*CommitMsg
	if len(notifications) > types.TxGroupMaxCount {
		buff = (notifications)[:types.TxGroupMaxCount]
	} else {
		buff = (notifications)[:]
	}
	var rawTxs types.Transactions
	for _, msg := range buff {
		tx, err := getCommitMsgTx(msg)
		if err != nil {
			plog.Error("para get commit tx", "block height", msg.blockDetail.Block.Height)
			return "", 0, err
		}
		rawTxs.Txs = append(rawTxs.Txs, tx)

	}

	txs, err := getTxsGroup(&rawTxs)
	if err != nil {
		return "", 0, err
	}
	return txs, len(buff), nil
}

func singleCalcTx(msg *CommitMsg) (string, error) {
	tx, err := getCommitMsgTx(msg)
	if err != nil {
		plog.Error("para get commit tx", "block height", msg.blockDetail.Block.Height)
		return "", err
	}

	ret := hex.EncodeToString(types.Encode(tx))
	return ret, nil

}

func (client *CommitMsgClient) sendCommitMsg(txs string) error {
	signTx, err := client.signCommitMsgTx(txs)
	if err != nil {
		return err
	}
	err = client.sendCommitMsgTx(signTx)
	if err != nil {
		//if send fail, not return, just send 2 blocks after
		plog.Error("para sendCommitMsgTx", "tx", signTx)
	}

	client.currentTx = signTx
	client.waitingTx = true
	client.checkTxCommitTimes = 0

	return nil
}

func checkTxInMainBlock(targetTx string, detail *types.BlockDetail) (bool, error) {
	data, err := common.FromHex(targetTx)
	if err != nil {
		plog.Error("checkTxInMainBlock targetTx", "tx", targetTx, "err", err.Error())
		return false, err
	}
	var decodeTx types.Transaction
	types.Decode(data, &decodeTx)
	targetHash := decodeTx.Hash()

	for i, tx := range detail.Block.Txs {
		if bytes.Equal(targetHash, tx.Hash()) && detail.Receipts[i].Ty == types.ExecOk {
			return true, nil
		}
	}
	return false, nil

}

func getCommitMsgTx(msg *CommitMsg) (*types.Transaction, error) {
	status := &types.ParacrossNodeStatus{
		Title:         types.GetTitle(),
		Height:        msg.blockDetail.Block.Height,
		PreBlockHash:  msg.blockDetail.Block.ParentHash,
		BlockHash:     msg.blockDetail.Block.Hash(),
		MainBlockHash: msg.mainBlockHash,
		PreStateHash:  msg.blockDetail.PrevStatusHash,
		StateHash:     msg.blockDetail.Block.StateHash,
	}

	var curTxsHash [][]byte
	for _, tx := range msg.blockDetail.Block.Txs {
		curTxsHash = append(curTxsHash, tx.Hash())
	}

	status.TxResult = util.CalcByteBitMap(msg.initTxHashs, curTxsHash, msg.blockDetail.Receipts)
	status.TxCounts = uint32(len(msg.initTxHashs))

	tx := paracross.CreateRawCommitTx(status)
	fee, err := tx.GetRealFee(types.MinFee)
	if err != nil {
		plog.Error("getCommitMsgTx fee", "err", err.Error())
		return nil, err
	}
	tx.Fee = fee

	return tx, nil
}

func getTxsGroup(txsArr *types.Transactions) (string, error) {
	if len(txsArr.Txs) < 2 {
		tx := hex.EncodeToString(types.Encode(txsArr.Txs[0]))
		return tx, nil
	}

	group, err := types.CreateTxGroup(txsArr.Txs)
	if err != nil {
		plog.Error("para CreateTxGroup", "err", err.Error())
		return "", err
	}
	err = group.Check(types.MinFee)
	if err != nil {
		plog.Error("para CheckTxGroup", "err", err.Error())
		return "", err
	}
	newtx := group.Tx()
	grouptx := hex.EncodeToString(types.Encode(newtx))
	return grouptx, nil
}

func (client *CommitMsgClient) signCommitMsgTx(rawTx string) (string, error) {

	unsignedTx := &types.ReqSignRawTx{
		Addr:   paraAccount,
		TxHex:  rawTx,
		Expire: (time.Second * 120).String(),
	}

	msg := client.paraClient.GetQueueClient().NewMessage("wallet", types.EventSignRawTx, unsignedTx)
	client.paraClient.GetQueueClient().Send(msg, true)
	resp, err := client.paraClient.GetQueueClient().Wait(msg)
	if err != nil {
		plog.Error("sendCommitMsgTx wallet", "err", err.Error())
		return "", err
	}

	return resp.GetData().(*types.ReplySignRawTx).TxHex, nil
}

func (client *CommitMsgClient) sendCommitMsgTx(txHex string) error {
	var parm types.Transaction
	data, err := common.FromHex(txHex)
	if err != nil {
		plog.Error("sendCommitMsgTx hex", "tx", txHex, "err", err.Error())
		return err
	}
	types.Decode(data, &parm)
	ret, err := client.paraClient.grpcClient.SendTransaction(context.Background(), &parm)
	if err != nil {
		plog.Error("sendCommitMsgTx send tx", "tx", txHex, "err", err.Error())
		return err
	}
	plog.Info("procParaChain SendTransaction", "IsOk: ", ret.GetIsOk(), "msg: ", string(ret.GetMsg()))

	return nil

}

func (client *CommitMsgClient) onBlockAdded(msg *CommitMsg) {
	checkTicker := time.NewTicker(time.Second * 1)
	select {
	case client.commitMsgNofity <- msg:
	case <-checkTicker.C:
	case <-client.quit:
	}
}

func (client *CommitMsgClient) onMainBlockAdded(block *types.BlockDetail) {
	checkTicker := time.NewTicker(time.Second * 1)
	select {
	case client.mainBlockNotify <- block:
	case <-checkTicker.C:
	case <-client.quit:
	}
}
