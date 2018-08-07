package para

import (
	"bytes"
	"context"
	"time"

	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/types/executor/paracross"
	"gitlab.33.cn/chain33/chain33/util"
)

const (
	waitMainBlocks = 2
)

var (
	consensusInterval int = 16
)

type CommitMsg struct {
	mainBlockHash []byte
	mainHeight    int64
	initTxHashs   [][]byte
	blockDetail   *types.BlockDetail
}

type CommitMsgClient struct {
	paraClient         *ParaClient
	commitMsgNotify    chan *CommitMsg
	delMsgNotify       chan *CommitMsg
	mainBlockAdd       chan *types.BlockDetail
	currentTx          *types.Transaction
	waitingTx          bool
	checkTxCommitTimes int
	privateKey         crypto.PrivKey
	quit               chan struct{}
}

type sendMsgRst struct {
	err error
}

func (client *CommitMsgClient) handler() {
	var isSync bool
	var notifications []*CommitMsg
	var sendingMsgs []*CommitMsg
	var finishMsgs []*CommitMsg
	readTick := time.Tick(time.Second)
	consensusTick := time.Tick(time.Second * time.Duration(consensusInterval))

	sendMsgFail := make(chan sendMsgRst, 1)
	consensusRst := make(chan *types.ParacrossStatus, 1)
	priKeyRst := make(chan crypto.PrivKey, 1)
	go client.fetchPrivacyKey(priKeyRst)

	for {
		select {
		case msg, ok := <-client.commitMsgNotify:
			if !ok {
				continue
			}
			notifications = append(notifications, msg)
		case msg := <-client.delMsgNotify:
			var found bool
			for i, node := range notifications {
				if node.blockDetail.Block.Height == msg.blockDetail.Block.Height {
					notifications = append(notifications[:i], notifications[i+1:]...)
					found = true
				}
			}
			//check only notify and sending, not check finished
			if !found {
				for i, node := range sendingMsgs {
					if node.blockDetail.Block.Height == msg.blockDetail.Block.Height {
						sendingMsgs = append(sendingMsgs[:i], sendingMsgs[i+1:]...)
						notifications = append(sendingMsgs[:], notifications...)
						sendingMsgs = nil
						client.waitingTx = false
						client.currentTx = nil
						client.checkTxCommitTimes = 0
					}
				}
			}
		case block := <-client.mainBlockAdd:
			if client.currentTx != nil {
				exist, err := checkTxInMainBlock(client.currentTx, block)
				if err != nil {
					continue
				}
				if exist {
					finishMsgs = append(finishMsgs, sendingMsgs...)
					sendingMsgs = nil
					client.waitingTx = false
					client.currentTx = nil
					client.checkTxCommitTimes = 0
				} else {
					client.checkTxCommitTimes++
					if client.checkTxCommitTimes > waitMainBlocks {
						client.checkTxCommitTimes = 0
						//需要从rawtx构建,nonce需要改，不然会认为重复交易
						signTx, _, err := client.calcCommitMsgTxs(sendingMsgs)
						if err != nil || signTx == nil {
							continue
						}

						client.currentTx = signTx
						go client.sendCommitMsgTx(signTx, sendMsgFail)
					}
				}
			}

		case <-readTick:
			if len(notifications) != 0 && !client.waitingTx && isSync && client.privateKey != nil {
				signTx, count, err := client.calcCommitMsgTxs(notifications)
				if err != nil || signTx == nil {
					continue
				}
				sendingMsgs = notifications[:count]
				notifications = notifications[count:]
				client.currentTx = signTx
				client.waitingTx = true
				client.checkTxCommitTimes = 0
				go client.sendCommitMsgTx(signTx, sendMsgFail)
			}
		case <-consensusTick:
			go client.getConsensusHeight(isSync, consensusRst)

		//获取正在共识的高度，也就是可能还没完成共识
		case rsp := <-consensusRst:
			if !isSync {
				if len(notifications) == 0 && rsp.Height == -1 {
					isSync = true
				}
				//如果是节点重启过，未共识过的小于当前共识高度的区块，可以不参与共识
				//如果是新节点，一直等到同步的区块达到了共识高度，才参与共识
				for i, msg := range notifications {
					if msg.blockDetail.Block.Height >= rsp.Height {
						isSync = true
						notifications = notifications[i:]
						break
					}
				}
			} else {
				for i, msg := range finishMsgs {
					if msg.blockDetail.Block.Height >= rsp.Height {
						finishMsgs = finishMsgs[i:]
						break
					}
				}
			}

		case <-sendMsgFail:
			go client.sendCommitMsgTx(client.currentTx, sendMsgFail)

		case key, ok := <-priKeyRst:
			if !ok {
				priKeyRst = nil
				continue
			}
			client.privateKey = key

		case <-client.quit:
			return
		}
	}
}

//only sync once, as main usually sync, here just need the first sync status after start up
func (client *CommitMsgClient) mainSync() error {
	req := &types.ReqNil{}
	reply, err := client.paraClient.grpcClient.IsSync(context.Background(), req)
	if err != nil {
		plog.Error("Paracross main is syncing", "err", err.Error())
		return err
	}
	if !reply.IsOk {
		plog.Error("Paracross main reply not ok")
		return err
	}

	plog.Info("Paracross main sync succ")
	return nil

}

func (client *CommitMsgClient) getConsensusHeight(isSync bool, consensusRst chan *types.ParacrossStatus) {
	if !isSync {
		err := client.mainSync()
		if err != nil {
			return
		}
	}

	payLoad := types.Encode(&types.ReqStr{
		ReqStr: types.GetTitle(),
	})
	query := types.Query{
		Execer:   types.ExecerPara,
		FuncName: "ParacrossGetTitle",
		Payload:  payLoad,
	}
	ret, err := client.paraClient.grpcClient.QueryChain(context.Background(), &query)
	if err != nil {
		plog.Error("getConsensusHeight ", "err", err.Error())
		return
	}
	if !ret.GetIsOk() {
		plog.Info("getConsensusHeight not OK", "error", ret.GetMsg())
		return
	}

	var result types.ParacrossStatus
	types.Decode(ret.Msg, &result)
	consensusRst <- &result

}

func (client *CommitMsgClient) calcCommitMsgTxs(notifications []*CommitMsg) (*types.Transaction, int, error) {
	txs, count, err := client.batchCalcTxGroup(notifications)
	if err != nil {
		txs, err = client.singleCalcTx((notifications)[0])
		if err != nil {
			plog.Error("single calc tx", "height", notifications[0].blockDetail.Block.Height)

			return nil, 0, err
		}
		return txs, 1, nil
	}
	return txs, count, nil
}

func (client *CommitMsgClient) getTxsGroup(txsArr *types.Transactions) (*types.Transaction, error) {
	if len(txsArr.Txs) < 2 {
		tx := txsArr.Txs[0]
		tx.Sign(types.SECP256K1, client.privateKey)
		return tx, nil
	}

	group, err := types.CreateTxGroup(txsArr.Txs)
	if err != nil {
		plog.Error("para CreateTxGroup", "err", err.Error())
		return nil, err
	}
	err = group.Check(types.MinFee)
	if err != nil {
		plog.Error("para CheckTxGroup", "err", err.Error())
		return nil, err
	}
	//key := client.getPrivacyKey()
	for i := range group.Txs {
		group.SignN(i, int32(types.SECP256K1), client.privateKey)
	}

	newtx := group.Tx()
	return newtx, nil
}

func (client *CommitMsgClient) batchCalcTxGroup(notifications []*CommitMsg) (*types.Transaction, int, error) {
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
			return nil, 0, err
		}
		rawTxs.Txs = append(rawTxs.Txs, tx)

	}

	txs, err := client.getTxsGroup(&rawTxs)
	if err != nil {
		return nil, 0, err
	}
	return txs, len(buff), nil
}

func (client *CommitMsgClient) singleCalcTx(msg *CommitMsg) (*types.Transaction, error) {
	tx, err := getCommitMsgTx(msg)
	if err != nil {
		plog.Error("para get commit tx", "block height", msg.blockDetail.Block.Height)
		return nil, err
	}
	tx.Sign(types.SECP256K1, client.privateKey)
	return tx, nil

}

func getCommitMsgTx(msg *CommitMsg) (*types.Transaction, error) {
	status := &types.ParacrossNodeStatus{
		MainBlockHash:   msg.mainBlockHash,
		MainBlockHeight: msg.mainHeight,
		Title:           types.GetTitle(),
		Height:          msg.blockDetail.Block.Height,
		PreBlockHash:    msg.blockDetail.Block.ParentHash,
		BlockHash:       msg.blockDetail.Block.Hash(),
		PreStateHash:    msg.blockDetail.PrevStatusHash,
		StateHash:       msg.blockDetail.Block.StateHash,
	}

	var curTxsHash [][]byte
	for _, tx := range msg.blockDetail.Block.Txs {
		curTxsHash = append(curTxsHash, tx.Hash())
	}

	status.TxResult = util.CalcByteBitMap(msg.initTxHashs, curTxsHash, msg.blockDetail.Receipts)
	status.TxCounts = uint32(len(msg.initTxHashs))

	tx, err := paracross.CreateRawCommitTx4MainChain(status, types.ParaX, 0)
	if err != nil {
		plog.Error("getCommitMsgTx fee", "err", err.Error())
		return nil, err
	}

	return tx, nil
}

func (client *CommitMsgClient) sendCommitMsgTx(tx *types.Transaction, ch chan sendMsgRst) {
	err := client.sendCommitMsgTxEx(tx)
	if err != nil {
		rst := sendMsgRst{err: err}
		//wait 1s re-send
		time.Sleep(time.Second * 1)
		ch <- rst
	}
}

func (client *CommitMsgClient) sendCommitMsgTxEx(tx *types.Transaction) error {
	resp, err := client.paraClient.grpcClient.SendTransaction(context.Background(), tx)
	if err != nil {
		plog.Error("sendCommitMsgTx send tx", "tx", tx, "err", err.Error())
		return err
	}

	if !resp.GetIsOk() {
		plog.Error("sendCommitMsgTx send tx Nok", "tx", tx, "err", string(resp.GetMsg()))
		return errors.New(string(resp.GetMsg()))
	}

	return nil

}

func checkTxInMainBlock(targetTx *types.Transaction, detail *types.BlockDetail) (bool, error) {
	targetHash := targetTx.Hash()

	for i, tx := range detail.Block.Txs {
		if bytes.Equal(targetHash, tx.Hash()) && detail.Receipts[i].Ty == types.ExecOk {
			return true, nil
		}
	}
	return false, nil

}

func (client *CommitMsgClient) onBlockAdded(msg *CommitMsg) {
	checkTicker := time.NewTicker(time.Second * 1)
	select {
	case client.commitMsgNotify <- msg:
	case <-checkTicker.C:
	case <-client.quit:
	}
}

func (client *CommitMsgClient) onBlockDeleted(msg *CommitMsg) {
	checkTicker := time.NewTicker(time.Second * 1)
	select {
	case client.delMsgNotify <- msg:
	case <-checkTicker.C:
	case <-client.quit:
	}
}

func (client *CommitMsgClient) onMainBlockAdded(block *types.BlockDetail) {
	checkTicker := time.NewTicker(time.Second * 1)
	select {
	case client.mainBlockAdd <- block:
	case <-checkTicker.C:
	case <-client.quit:
	}
}

func (client *CommitMsgClient) fetchPrivacyKey(priKeyRst chan crypto.PrivKey) {
	if client.paraClient.authAccount == "" {
		close(priKeyRst)
		return
	}

	req := &types.ReqStr{ReqStr: client.paraClient.authAccount}
	for {
		msg := client.paraClient.GetQueueClient().NewMessage("wallet", types.EventDumpPrivkey, req)
		client.paraClient.GetQueueClient().Send(msg, true)
		resp, err := client.paraClient.GetQueueClient().Wait(msg)
		if err != nil {
			plog.Error("sendCommitMsgTx wallet", "err", err.Error())
			time.Sleep(time.Second * 1)
			continue
		}
		str := resp.GetData().(*types.ReplyStr).Replystr
		pk, err := common.FromHex(str)
		if err != nil && pk == nil {
			panic(err)
		}

		secp, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
		if err != nil {
			panic(err)
		}

		priKey, err := secp.PrivKeyFromBytes(pk)
		if err != nil {
			panic(err)
		}

		priKeyRst <- priKey
		break
	}
	close(priKeyRst)

}
