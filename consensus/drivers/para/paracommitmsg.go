package para

import (
	"bytes"
	"context"
	"encoding/hex"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/types/executor/paracross"
)

const (
	waitMainBlocks = 2
)

type ParaCommitMsg struct {
	height             int64
	oriParaTxHashs     [][]byte
	mainBlockStateHash []byte
	block              *types.BlockDetail
}

type ParaCommitClient struct {
	paraClient         *ParaClient
	commitMsgNofity    chan *ParaCommitMsg
	mainBlockNotify    chan *types.Block
	currentTx          string
	waitingTx          bool
	checkTxCommitTimes int
}

func (client *ParaCommitClient) handler() {
	var notifications []*ParaCommitMsg

	for {
		select {
		case msg, ok := <-client.commitMsgNofity:
			if !ok {
				continue
			}

			notifications = append(notifications, msg)

			if !client.waitingTx {
				client.sendCommitMsg(&notifications)

			}

		case block := <-client.mainBlockNotify:
			if client.currentTx == "" {
				continue
			}
			exist, err := checkTxInMainBlock(client.currentTx, block)
			if err != nil {
				continue
			}
			if exist {
				if len(notifications) != 0 {
					client.sendCommitMsg(&notifications)
				} else {
					client.waitingTx = false
					client.currentTx = ""
				}

			} else {
				client.checkTxCommitTimes++
				if client.checkTxCommitTimes > waitMainBlocks {
					client.sendCommitMsgTx(client.currentTx)
					client.checkTxCommitTimes = 0
				}
			}

		}
	}
}

func (client *ParaCommitClient) sendCommitMsg(notifications *[]*ParaCommitMsg) error {
	var buff []*ParaCommitMsg
	if len(*notifications) > types.TxGroupMaxCount {
		buff = (*notifications)[:types.TxGroupMaxCount]
		*notifications = (*notifications)[types.TxGroupMaxCount:]
	} else {
		buff = (*notifications)[:]
		*notifications = (*notifications)[len(*notifications):]
	}
	var rawTxs types.Transactions
	for _, msg := range buff {
		tx, err := getCommitMsgTx(msg)
		if err != nil {
			plog.Error("para get commit tx", "block height", msg.block.Block.Height)
			continue
		}
		rawTxs.Txs = append(rawTxs.Txs, tx)

	}
	txs, err := getTxsGroup(&rawTxs)
	if err != nil {
		return err
	}
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

func checkTxInMainBlock(targetTx string, block *types.Block) (bool, error) {
	data, err := common.FromHex(targetTx)
	if err != nil {
		plog.Error("checkTxInMainBlock targetTx", "tx", targetTx, "err", err.Error())
		return false, err
	}
	var decodeTx types.Transaction
	types.Decode(data, &decodeTx)
	targetHash := decodeTx.Hash()

	for _, tx := range block.Txs {
		if bytes.Equal(targetHash, tx.Hash()) {
			return true, nil
		}
	}
	return false, nil

}

func getCommitMsgTx(msg *ParaCommitMsg) (*types.Transaction, error) {
	status := &types.ParacrossNodeStatus{
		Title:         types.GetTitle(),
		Height:        msg.block.Block.Height,
		StateHash:     msg.block.Block.StateHash,
		MainStateHash: msg.mainBlockStateHash,
	}

	var curTxsHash [][]byte
	for _, tx := range msg.block.Block.Txs {
		curTxsHash = append(curTxsHash, tx.Hash())
	}

	var bitRst byte
	for index := 0; index < len(msg.oriParaTxHashs); index++ {
		if index > 0 && index%8 == 0 {
			status.TxResult = append(status.TxResult, bitRst)
			bitRst = 0
		}
		for i, curHash := range curTxsHash {
			if bytes.Equal(msg.oriParaTxHashs[index], curHash) {
				if msg.block.Receipts[i].Ty == types.ExecOk {
					bitRst |= 1 << (7 - (uint32(index) % 8))
				}
			}
		}
	}

	if len(msg.oriParaTxHashs)%8 > 0 {
		status.TxResult = append(status.TxResult, bitRst)
	}
	status.TxCounts = uint32(len(msg.oriParaTxHashs))

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

func (client *ParaCommitClient) signCommitMsgTx(rawTx string) (string, error) {

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

func (client *ParaCommitClient) sendCommitMsgTx(txHex string) error {
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

func (client *ParaCommitClient) onBlockAdded(msg *ParaCommitMsg) {
	checkTicker := time.NewTicker(time.Second * 1)
	select {
	case client.commitMsgNofity <- msg:
	case <-checkTicker.C:
		//case <-b.quit:
	}
}

func (client *ParaCommitClient) onMainBlockAdded(block *types.Block) {
	checkTicker := time.NewTicker(time.Second * 1)
	select {
	case client.mainBlockNotify <- block:
	case <-checkTicker.C:
		//case <-b.quit:
	}
}
