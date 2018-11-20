// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of policy source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"sync"

	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	wcom "github.com/33cn/chain33/wallet/common"
)

const (
	walletBizPolicyX = "walletBizPolicy"
)

func init() {
	wcom.RegisterPolicy(walletBizPolicyX, newPolicy())
}

func newPolicy() wcom.WalletBizPolicy {
	ret := &walletBizPlicy{
		mtx: &sync.Mutex{},
	}
	return ret
}

// 默认的钱包业务实现
type walletBizPlicy struct {
	mtx           *sync.Mutex
	walletOperate wcom.WalletOperate
}

func (policy *walletBizPlicy) Init(walletBiz wcom.WalletOperate, sub []byte) {
	policy.setWalletOperate(walletBiz)
}

func (policy *walletBizPlicy) setWalletOperate(walletBiz wcom.WalletOperate) {
	policy.mtx.Lock()
	defer policy.mtx.Unlock()
	policy.walletOperate = walletBiz
}

func (policy *walletBizPlicy) getWalletOperate() wcom.WalletOperate {
	policy.mtx.Lock()
	defer policy.mtx.Unlock()
	return policy.walletOperate
}

func (policy *walletBizPlicy) OnAddBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) *types.WalletTxDetail {
	return nil
}

func (policy *walletBizPlicy) OnDeleteBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) *types.WalletTxDetail {
	return nil
}

func (policy *walletBizPlicy) SignTransaction(key crypto.PrivKey, req *types.ReqSignRawTx) (needSysSign bool, signtx string, err error) {
	needSysSign = true
	return
}

func (policy *walletBizPlicy) OnCreateNewAccount(acc *types.Account) {
	wg := policy.getWalletOperate().GetWaitGroup()
	wg.Add(1)
	go policy.rescanReqTxDetailByAddr(acc.Addr, wg)
}

func (policy *walletBizPlicy) OnImportPrivateKey(acc *types.Account) {
	wg := policy.getWalletOperate().GetWaitGroup()
	wg.Add(1)
	go policy.rescanReqTxDetailByAddr(acc.Addr, wg)
}

func (policy *walletBizPlicy) OnWalletLocked() {
}

func (policy *walletBizPlicy) OnWalletUnlocked(WalletUnLock *types.WalletUnLock) {
}

func (policy *walletBizPlicy) OnAddBlockFinish(block *types.BlockDetail) {
}

func (policy *walletBizPlicy) OnDeleteBlockFinish(block *types.BlockDetail) {
}

func (policy *walletBizPlicy) OnClose() {
}

func (policy *walletBizPlicy) OnSetQueueClient() {
}

func (policy *walletBizPlicy) Call(funName string, in types.Message) (ret types.Message, err error) {
	err = types.ErrNotSupport
	return
}

//从blockchain模块同步addr参与的所有交易详细信息
func (policy *walletBizPlicy) rescanReqTxDetailByAddr(addr string, wg *sync.WaitGroup) {
	defer wg.Done()
	policy.reqTxDetailByAddr(addr)
}

//从blockchain模块同步addr参与的所有交易详细信息
func (policy *walletBizPlicy) reqTxDetailByAddr(addr string) {
	if len(addr) == 0 {
		walletlog.Error("reqTxDetailByAddr input addr is nil!")
		return
	}
	var txInfo types.ReplyTxInfo

	i := 0
	operater := policy.getWalletOperate()
	for {
		//首先从blockchain模块获取地址对应的所有交易hashs列表,从最新的交易开始获取
		var ReqAddr types.ReqAddr
		ReqAddr.Addr = addr
		ReqAddr.Flag = 0
		ReqAddr.Direction = 0
		ReqAddr.Count = int32(MaxTxHashsPerTime)
		if i == 0 {
			ReqAddr.Height = -1
			ReqAddr.Index = 0
		} else {
			ReqAddr.Height = txInfo.GetHeight()
			ReqAddr.Index = txInfo.GetIndex()
		}
		i++
		ReplyTxInfos, err := operater.GetAPI().GetTransactionByAddr(&ReqAddr)
		if err != nil {
			walletlog.Error("reqTxDetailByAddr", "GetTransactionByAddr error", err, "addr", addr)
			return
		}
		if ReplyTxInfos == nil {
			walletlog.Info("reqTxDetailByAddr ReplyTxInfos is nil")
			return
		}
		txcount := len(ReplyTxInfos.TxInfos)

		var ReqHashes types.ReqHashes
		ReqHashes.Hashes = make([][]byte, len(ReplyTxInfos.TxInfos))
		for index, ReplyTxInfo := range ReplyTxInfos.TxInfos {
			ReqHashes.Hashes[index] = ReplyTxInfo.GetHash()
			txInfo.Hash = ReplyTxInfo.GetHash()
			txInfo.Height = ReplyTxInfo.GetHeight()
			txInfo.Index = ReplyTxInfo.GetIndex()
		}
		operater.GetTxDetailByHashs(&ReqHashes)
		if txcount < int(MaxTxHashsPerTime) {
			return
		}
	}
}
