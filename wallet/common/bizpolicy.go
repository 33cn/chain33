// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

// WalletBizPolicy 细分钱包业务逻辑的街口
type WalletBizPolicy interface {
	// Init 初始化钱包业务策略，在使用前调用
	Init(walletBiz WalletOperate, sub []byte)
	// OnAddBlockTx 当区块被增加确认时调用本函数
	// block： 被增加的区块详细信息
	// tx: 区块增加的交易信息
	// index: 交易信息在区块上的索引为止，从0开始计数
	// dbbatch: 数据库批量操作接口
	// 返回值为提供给外部快速检索的钱包详细信息，如果内部已经处理，不需要外部处理的情况下，可以返回nil
	OnAddBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) *types.WalletTxDetail
	// OnDeleteBlockTx 当区块被回退删除时调用本函数
	// block： 被回退的区块详细信息
	// tx: 区块回退的交易信息
	// index: 交易信息在区块上的索引为止，从0开始计数
	// dbbatch: 数据库批量操作接口
	// 返回值为提供给外部快速检索的钱包详细信息，如果内部已经处理，不需要外部处理的情况下，可以返回nil
	OnDeleteBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) *types.WalletTxDetail
	// SignTransaction 针对特殊的交易，按照新的签名方式签名
	// key 签名的私钥信息
	// req 需要签名交易流信息
	// needSysSign 表示是否需要继续走系统流程的签名，true表示继续，false表示已经完成签名，不需要系统处理
	// signtx 签名成功后，保存签名成功的交易字符串
	// err 错误信息
	SignTransaction(key crypto.PrivKey, req *types.ReqSignRawTx) (needSysSign bool, signtx string, err error)
	// OnCreateNewAccount 当用户在钱包中新建账户时，将调用本接口函数
	OnCreateNewAccount(acc *types.Account)
	// OnImportPrivateKey 当用户导入私钥时，将调用本函数
	OnImportPrivateKey(acc *types.Account)
	OnWalletLocked()
	OnWalletUnlocked(WalletUnLock *types.WalletUnLock)
	OnAddBlockFinish(block *types.BlockDetail)
	OnDeleteBlockFinish(block *types.BlockDetail)
	OnClose()
	OnSetQueueClient()
	Call(funName string, in types.Message) (ret types.Message, err error)
}
