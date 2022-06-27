// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"fmt"

	"github.com/33cn/chain33/system/address/eth"

	"github.com/33cn/chain33/system/crypto/secp256k1eth"

	"github.com/33cn/chain33/common/address"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
)

// Exec_Transfer transfer of exec
func (c *Coins) Exec_Transfer(transfer *types.AssetsTransfer, tx *types.Transaction, index int) (*types.Receipt, error) {
	fmt.Println("Exec_Transfer--chain33.coins--------------------------------------sfer00000000000000000")
	from := tx.From()
	//to 是 execs 合约地址
	var receipt *types.Receipt
	var err error
	if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) {
		receipt, err = c.GetCoinsAccount().TransferToExec(from, tx.GetRealToAddr(), transfer.Amount)

	} else {
		receipt, err = c.GetCoinsAccount().Transfer(from, tx.GetRealToAddr(), transfer.Amount)
	}

	//TODO 删掉，evm统一处理
	if tx.GetSignature().GetTy() == types.EncodeSignID(secp256k1eth.ID, eth.ID) {
		if err != nil {
			if receipt == nil {
				receipt = new(types.Receipt)
				receipt.Ty = types.ExecPack
			}

		}
		// 设置账户的nonce
		key := secp256k1eth.CaculCoinsEvmAccountKey(from)
		var nonceInfo types.EvmAccountNonce
		stataV, err := c.GetStateDB().Get(key)
		if err == nil {
			fmt.Println("Exec_Transfer--------------------------,tx.GetNonce()", tx.GetNonce())
			types.Decode(stataV, &nonceInfo)
			if nonceInfo.GetNonce() == tx.GetNonce() {
				fmt.Println("Exec_Transfer--------------------------befer", nonceInfo.GetNonce())
				nonceInfo.Nonce += 1
			}

		} else {
			fmt.Println("Exec_Transfer--------------------------getstatdb", err.Error())
			nonceInfo.Nonce = 1
			nonceInfo.Addr = tx.From()
		}
		var kv types.KeyValue
		kv.Key = key
		kv.Value = types.Encode(&nonceInfo)
		receipt.KV = append(receipt.KV, &kv)
	}

	return receipt, err
}

// Exec_TransferToExec the transfer to exec address
func (c *Coins) Exec_TransferToExec(transfer *types.AssetsTransferToExec, tx *types.Transaction, index int) (*types.Receipt, error) {
	types.AssertConfig(c.GetAPI())
	cfg := c.GetAPI().GetConfig()
	if !cfg.IsFork(c.GetHeight(), "ForkTransferExec") {
		return nil, types.ErrActionNotSupport
	}
	from := tx.From()
	//to 是 execs 合约地址
	if !isExecAddrMatch(transfer.ExecName, tx.GetRealToAddr()) {
		return nil, types.ErrToAddrNotSameToExecAddr
	}
	return c.GetCoinsAccount().TransferToExec(from, tx.GetRealToAddr(), transfer.Amount)
}

// Exec_Withdraw withdraw exec
func (c *Coins) Exec_Withdraw(withdraw *types.AssetsWithdraw, tx *types.Transaction, index int) (*types.Receipt, error) {
	types.AssertConfig(c.GetAPI())
	cfg := c.GetAPI().GetConfig()
	if !cfg.IsFork(c.GetHeight(), "ForkWithdraw") {
		withdraw.ExecName = ""
	}
	from := tx.From()
	//to 是 execs 合约地址
	if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) || isExecAddrMatch(withdraw.ExecName, tx.GetRealToAddr()) {
		return c.GetCoinsAccount().TransferWithdraw(from, tx.GetRealToAddr(), withdraw.Amount)
	}
	return nil, types.ErrActionNotSupport
}

// Exec_Genesis genesis of exec
func (c *Coins) Exec_Genesis(genesis *types.AssetsGenesis, tx *types.Transaction, index int) (*types.Receipt, error) {
	if c.GetHeight() == 0 {
		if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) {
			return c.GetCoinsAccount().GenesisInitExec(genesis.ReturnAddress, genesis.Amount, tx.GetRealToAddr())
		}
		return c.GetCoinsAccount().GenesisInit(tx.GetRealToAddr(), genesis.Amount)
	}
	return nil, types.ErrReRunGenesis
}

func isExecAddrMatch(name string, to string) bool {
	toaddr := address.ExecAddress(name)
	return toaddr == to
}
