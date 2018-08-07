package privacybizpolicy

import (
	"bytes"

	"github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet/bizpolicy"
	"gitlab.33.cn/chain33/chain33/wallet/walletoperate"
)

var (
	bizlog = log15.New("module", "wallet.privacy")

	MaxTxHashsPerTime int64 = 100
	maxTxNumPerBlock  int64 = types.MaxTxsPerBlock
)

func New() bizpolicy.WalletBizPolicy {
	return &walletPrivacyBiz{}
}

type walletPrivacyBiz struct {
	funcmap       queue.FuncMap
	store         *privacyStore
	walletOperate walletoperate.WalletOperate

	rescanUTXOflag int32
}

func (biz *walletPrivacyBiz) Init(walletOperate walletoperate.WalletOperate) {
	biz.walletOperate = walletOperate
	biz.store = &privacyStore{db: walletOperate.GetDBStore()}

	biz.funcmap.Init()

	biz.funcmap.Register(types.EventShowPrivacyPK, biz.onShowPrivacyPK)
	biz.funcmap.Register(types.EventShowPrivacyAccountSpend, biz.onShowPrivacyAccountSpend)
	biz.funcmap.Register(types.EventPublic2privacy, biz.onPublic2Privacy)
	biz.funcmap.Register(types.EventPrivacy2privacy, biz.onPrivacy2Privacy)
	biz.funcmap.Register(types.EventPrivacy2public, biz.onPrivacy2Public)
	biz.funcmap.Register(types.EventCreateUTXOs, biz.onCreateUTXOs)
	biz.funcmap.Register(types.EventCreateTransaction, biz.onCreateTransaction)
	biz.funcmap.Register(types.EventPrivacyAccountInfo, biz.onPrivacyAccountInfo)
	biz.funcmap.Register(types.EventPrivacyTransactionList, biz.onPrivacyTransactionList)
	biz.funcmap.Register(types.EventRescanUtxos, biz.onRescanUtxos)
	biz.funcmap.Register(types.EventEnablePrivacy, biz.onEnablePrivacy)
}
func (biz *walletPrivacyBiz) OnAddBlockFinish() {

}

func (biz *walletPrivacyBiz) OnDeleteBlockFinish() {

}

type buildStoreWalletTxDetailParam struct {
	tokenname    string
	block        *types.BlockDetail
	tx           *types.Transaction
	index        int
	newbatch     db.Batch
	senderRecver string
	isprivacy    bool
	addDelType   int32
	sendRecvFlag int32
	utxos        []*types.UTXO
}

func (biz *walletPrivacyBiz) OnAddBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) {
	txhash := tx.Hash()
	txhashstr := common.Bytes2Hex(txhash)
	_, err := tx.Amount()
	if err != nil {
		bizlog.Error("OnAddBlockTx", "get transaction amount error", err, "txhash ", txhashstr)
		return
	}
	var privateAction types.PrivacyAction
	if err := types.Decode(tx.GetPayload(), &privateAction); err != nil {
		bizlog.Error("OnAddBlockTx", "txhash", txhashstr, "Decode tx.GetPayload() error", err)
		return
	}

	txExecRes := block.Receipts[index].Ty
	privacyOutput := privateAction.GetOutput()
	tokenname := privateAction.GetTokenName()
	RpubKey := privacyOutput.GetRpubKeytx()
	totalUtxosLeft := len(privacyOutput.Keyoutput)

	// 处理交易中的输入信息
	if privacyInfo, err := biz.getPrivacyKeyPairs(); err == nil {
		matchedCount := 0
		utxoProcessed := make([]bool, len(privacyOutput.Keyoutput))
		for _, info := range privacyInfo {
			privacykeyParirs := info.PrivacyKeyPair
			matched4addr := false
			var utxos []*types.UTXO
			for indexoutput, output := range privacyOutput.Keyoutput {
				if utxoProcessed[indexoutput] {
					continue
				}
				if types.ExecOk != txExecRes {
					//对于执行失败的交易，只需要将该交易记录在钱包就行
					bizlog.Error("OnAddBlockTx", "txhash", txhashstr, "txExecRes", txExecRes)
					break
				}
				priv, err := privacy.RecoverOnetimePriKey(RpubKey, privacykeyParirs.ViewPrivKey, privacykeyParirs.SpendPrivKey, int64(indexoutput))
				if err != nil {
					bizlog.Error("OnAddBlockTx", "txhash", txhashstr, "RecoverOnetimePriKey error", err)
					continue
				}
				if !bytes.Equal(priv.PubKey().Bytes()[:], output.Onetimepubkey) {
					continue
				}
				//为了避免匹配成功之后不必要的验证计算，需要统计匹配次数
				//因为目前只会往一个隐私账户转账，
				//1.一般情况下，只会匹配一次，如果是往其他钱包账户转账，
				//2.但是如果是往本钱包的其他地址转账，因为可能存在的change，会匹配2次
				matched4addr = true
				totalUtxosLeft--
				utxoProcessed[indexoutput] = true
				info2store := &types.PrivacyDBStore{
					Txhash:           txhash,
					Tokenname:        tokenname,
					Amount:           output.Amount,
					OutIndex:         int32(indexoutput),
					TxPublicKeyR:     RpubKey,
					OnetimePublicKey: output.Onetimepubkey,
					Owner:            *info.Addr,
					Height:           block.Block.Height,
					Txindex:          index,
					Blockhash:        block.Block.Hash(),
				}

				utxoGlobalIndex := &types.UTXOGlobalIndex{
					Outindex: int32(indexoutput),
					Txhash:   txhash,
				}

				utxoCreated := &types.UTXO{
					Amount: output.Amount,
					UtxoBasic: &types.UTXOBasic{
						UtxoGlobalIndex: utxoGlobalIndex,
						OnetimePubkey:   output.Onetimepubkey,
					},
				}

				utxos = append(utxos, utxoCreated)
				biz.store.setUTXO(info.Addr, &txhashstr, indexoutput, info2store, dbbatch)
				bizlog.Info("OnAddBlockTx", "add tx txhash", txhashstr, "setUTXO addr ", *info.Addr, "indexoutput", indexoutput)

			}
			if matched4addr {
				matchedCount++
				//匹配次数达到2次，不再对本钱包中的其他地址进行匹配尝试
				param := &buildStoreWalletTxDetailParam{
					tokenname:    tokenname,
					block:        block,
					tx:           tx,
					index:        int(index),
					newbatch:     dbbatch,
					senderRecver: *info.Addr,
					isprivacy:    true,
					addDelType:   AddTx,
					sendRecvFlag: recvTx,
					utxos:        utxos,
				}
				biz.buildAndStoreWalletTxDetail(param)
				if 2 == matchedCount || 0 == totalUtxosLeft || types.ExecOk != txExecRes {
					bizlog.Info("OnAddBlockTx", "txhash", txhashstr, "Get matched privacy transfer for address address", *info.Addr, "totalUtxosLeft", totalUtxosLeft, "matchedCount", matchedCount)
					break
				}
			}
		}
	}

	// 处理Output
	ftxos, keys := biz.store.getFTXOlist()
	for i, ftxo := range ftxos {
		//查询确认该交易是否为记录的支付交易
		if ftxo.Txhash != txhashstr {
			continue
		}
		if types.ExecOk == txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
			bizlog.Info("OnAddBlockTx", "txhash", txhashstr, "moveFTXO2STXO, key", string(keys[i]), "txExecRes", txExecRes)
			biz.store.moveFTXO2STXO(keys[i], txhashstr, dbbatch)
		} else if types.ExecOk != txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
			//如果执行失败
			bizlog.Info("OnAddBlockTx", "txhash", txhashstr, "moveFTXO2UTXO, key", string(keys[i]), "txExecRes", txExecRes)
			biz.store.moveFTXO2UTXO(keys[i], dbbatch)
		}
		//该交易正常执行完毕，删除对其的关注
		param := &buildStoreWalletTxDetailParam{
			tokenname:    tokenname,
			block:        block,
			tx:           tx,
			index:        int(index),
			newbatch:     dbbatch,
			senderRecver: ftxo.Sender,
			isprivacy:    true,
			addDelType:   AddTx,
			sendRecvFlag: sendTx,
			utxos:        nil,
		}
		biz.buildAndStoreWalletTxDetail(param)
	}
}

func (biz *walletPrivacyBiz) OnDeleteBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) {
	txhash := tx.Hash()
	txhashstr := common.Bytes2Hex(txhash)
	_, err := tx.Amount()
	if err != nil {
		bizlog.Error("OnDeleteBlockTx", "get transaction amount error", err, "txhash ", txhashstr)
		return
	}
	var privateAction types.PrivacyAction
	if err := types.Decode(tx.GetPayload(), &privateAction); err != nil {
		bizlog.Error("OnDeleteBlockTx", "txhash", txhashstr, "Decode tx.GetPayload() error", err)
		return
	}

	txExecRes := block.Receipts[index].Ty
	privacyOutput := privateAction.GetOutput()
	tokenname := privateAction.GetTokenName()
	RpubKey := privacyOutput.GetRpubKeytx()
	totalUtxosLeft := len(privacyOutput.Keyoutput)

	// 处理交易中的输入信息
	if privacyInfo, err := biz.getPrivacyKeyPairs(); err == nil {
		matchedCount := 0
		utxoProcessed := make([]bool, len(privacyOutput.Keyoutput))
		for _, info := range privacyInfo {
			privacykeyParirs := info.PrivacyKeyPair
			matched4addr := false
			var utxos []*types.UTXO
			for indexoutput, output := range privacyOutput.Keyoutput {
				if utxoProcessed[indexoutput] {
					continue
				}
				if types.ExecOk != txExecRes {
					//对于执行失败的交易，只需要将该交易记录在钱包就行
					bizlog.Error("OnDeleteBlockTx", "txhash", txhashstr, "txExecRes", txExecRes)
					break
				}
				priv, err := privacy.RecoverOnetimePriKey(RpubKey, privacykeyParirs.ViewPrivKey, privacykeyParirs.SpendPrivKey, int64(indexoutput))
				if err != nil {
					bizlog.Error("OnDeleteBlockTx", "txhash", txhashstr, "RecoverOnetimePriKey error", err)
					continue
				}
				if !bytes.Equal(priv.PubKey().Bytes()[:], output.Onetimepubkey) {
					continue
				}
				//为了避免匹配成功之后不必要的验证计算，需要统计匹配次数
				//因为目前只会往一个隐私账户转账，
				//1.一般情况下，只会匹配一次，如果是往其他钱包账户转账，
				//2.但是如果是往本钱包的其他地址转账，因为可能存在的change，会匹配2次
				matched4addr = true
				totalUtxosLeft--
				utxoProcessed[indexoutput] = true
				bizlog.Info("OnDeleteBlockTx", "delete tx txhash", txhashstr, "unsetUTXO addr ", *info.Addr, "indexoutput", indexoutput)
				biz.store.unsetUTXO(info.Addr, &txhashstr, indexoutput, tokenname, dbbatch)

			}
			if matched4addr {
				matchedCount++
				//匹配次数达到2次，不再对本钱包中的其他地址进行匹配尝试
				param := &buildStoreWalletTxDetailParam{
					tokenname:    tokenname,
					block:        block,
					tx:           tx,
					index:        int(index),
					newbatch:     dbbatch,
					senderRecver: *info.Addr,
					isprivacy:    true,
					addDelType:   DelTx,
					sendRecvFlag: recvTx,
					utxos:        utxos,
				}
				biz.buildAndStoreWalletTxDetail(param)
				if 2 == matchedCount || 0 == totalUtxosLeft || types.ExecOk != txExecRes {
					bizlog.Info("OnDeleteBlockTx", "txhash", txhashstr, "Get matched privacy transfer for address address", *info.Addr, "totalUtxosLeft", totalUtxosLeft, "matchedCount", matchedCount)
					break
				}
			}
		}
	}

	// 处理Output
	//当发生交易回撤时，从记录的STXO中查找相关的交易，并将其重置为FTXO，因为该交易大概率会在其他区块中再次执行
	stxosInOneTx, _, _ := biz.store.getWalletFtxoStxo(STXOs4Tx)
	for _, ftxo := range stxosInOneTx {
		if ftxo.Txhash == txhashstr {
			param := &buildStoreWalletTxDetailParam{
				tokenname:    tokenname,
				block:        block,
				tx:           tx,
				index:        int(index),
				newbatch:     dbbatch,
				senderRecver: "",
				isprivacy:    true,
				addDelType:   DelTx,
				sendRecvFlag: sendTx,
				utxos:        nil,
			}

			if types.ExecOk == txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
				bizlog.Info("OnDeleteBlockTx", "txhash", txhashstr, "moveSTXO2FTXO txExecRes", txExecRes)
				biz.store.moveSTXO2FTXO(tx, txhashstr, dbbatch)
				biz.buildAndStoreWalletTxDetail(param)
			} else if types.ExecOk != txExecRes && types.ActionPublic2Privacy != privateAction.Ty {
				bizlog.Info("OnDeleteBlockTx", "txhash", txhashstr)
				biz.buildAndStoreWalletTxDetail(param)
			}
		}
	}
}

func (biz *walletPrivacyBiz) OnRecvQueueMsg(msg *queue.Message) (bool, error) {
	if msg == nil {
		return false, types.ErrInvalidParams
	}
	existed, topic, rettype, retdata, err := biz.funcmap.Process(msg)
	if !existed {
		return false, nil
	}
	if err == nil {
		msg.Reply(biz.walletOperate.GetAPI().NewMessage(topic, rettype, retdata))
	} else {
		msg.Reply(biz.walletOperate.GetAPI().NewMessage(topic, rettype, err))
	}
	return true, err
}

func (biz *walletPrivacyBiz) onShowPrivacyAccountSpend(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyShowPrivacyAccountSpend)

	req, ok := msg.Data.(*types.ReqPrivBal4AddrToken)
	if !ok {
		bizlog.Error("onShowPrivacyAccountSpend", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.showPrivacyAccountsSpend(req)
	if err != nil {
		bizlog.Error("showPrivacyAccountsSpend", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onShowPrivacyPK(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyShowPrivacyPK)

	req, ok := msg.Data.(*types.ReqStr)
	if !ok {
		bizlog.Error("onShowPrivacyPK", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.showPrivacyKeyPair(req)
	if err != nil {
		bizlog.Error("showPrivacyKeyPair", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onPublic2Privacy(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPublic2privacy)

	req, ok := msg.Data.(*types.ReqPub2Pri)
	if !ok {
		bizlog.Error("onPublic2Privacy", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.sendPublic2PrivacyTransaction(req)
	if err != nil {
		bizlog.Error("sendPublic2PrivacyTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onPrivacy2Privacy(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacy2privacy)

	req, ok := msg.Data.(*types.ReqPri2Pri)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.sendPrivacy2PrivacyTransaction(req)
	if err != nil {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onPrivacy2Public(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacy2public)

	req, ok := msg.Data.(*types.ReqPri2Pub)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.sendPrivacy2PublicTransaction(req)
	if err != nil {
		bizlog.Error("sendPrivacy2PublicTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onCreateUTXOs(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyCreateUTXOs)

	req, ok := msg.Data.(*types.ReqCreateUTXOs)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()
	reply, err := biz.createUTXOs(req)
	if err != nil {
		bizlog.Error("createUTXOs", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onCreateTransaction(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyCreateTransaction)
	req, ok := msg.Data.(*types.ReqCreateTransaction)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	ok, err := biz.walletOperate.CheckWalletStatus()
	if !ok {
		bizlog.Error("createTransaction", "CheckWalletStatus cause error.", err)
		return topic, retty, nil, err
	}
	if ok, err := biz.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return topic, retty, nil, err
	}
	if !checkAmountValid(req.Amount) {
		err = types.ErrAmount
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return topic, retty, nil, err
	}

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()

	reply, err := biz.createTransaction(req)
	if err != nil {
		bizlog.Error("createTransaction", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onPrivacyAccountInfo(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacyAccountInfo)
	req, ok := msg.Data.(*types.ReqPPrivacyAccount)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()

	reply, err := biz.getPrivacyAccountInfo(req)
	if err != nil {
		bizlog.Error("getPrivacyAccountInfo", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onPrivacyTransactionList(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyPrivacyTransactionList)
	req, ok := msg.Data.(*types.ReqPrivacyTransactionList)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}

	if req == nil {
		bizlog.Error("onPrivacyTransactionList param is nil")
		return topic, retty, nil, types.ErrInvalidParam
	}
	if req.Direction != 0 && req.Direction != 1 {
		bizlog.Error("getPrivacyTransactionList", "invalid direction ", req.Direction)
		return topic, retty, nil, types.ErrInvalidParam
	}
	// convert to sendTx / recvTx
	sendRecvFlag := req.SendRecvFlag + sendTx
	if sendRecvFlag != sendTx && sendRecvFlag != recvTx {
		bizlog.Error("getPrivacyTransactionList", "invalid sendrecvflag ", req.SendRecvFlag)
		return topic, retty, nil, types.ErrInvalidParam
	}
	req.SendRecvFlag = sendRecvFlag

	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()

	reply, err := biz.store.getWalletPrivacyTxDetails(req)
	if err != nil {
		bizlog.Error("getWalletPrivacyTxDetails", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onRescanUtxos(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyRescanUtxos)
	req, ok := msg.Data.(*types.ReqRescanUtxos)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()

	reply, err := biz.rescanUTXOs(req)
	if err != nil {
		bizlog.Error("rescanUTXOs", "err", err.Error())
	}
	return topic, retty, reply, err
}

func (biz *walletPrivacyBiz) onEnablePrivacy(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyEnablePrivacy)
	req, ok := msg.Data.(*types.ReqEnablePrivacy)
	if !ok {
		bizlog.Error("walletPrivacyBiz", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	biz.walletOperate.GetMutex().Lock()
	defer biz.walletOperate.GetMutex().Unlock()

	reply, err := biz.enablePrivacy(req)
	if err != nil {
		bizlog.Error("enablePrivacy", "err", err.Error())
	}
	return topic, retty, reply, err
}
