// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"encoding/json"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	wcom "github.com/33cn/chain33/wallet/common"
)

var (
	storelog = walletlog.New("submodule", "store")
)

func newStore(db db.DB) *walletStore {
	return &walletStore{Store: wcom.NewStore(db)}
}

type walletStore struct {
	*wcom.Store
}

// SetFeeAmount 设置钱包的手续费，本函数需要跟钱包中的费率一起改变，否则会有问题
func (ws *walletStore) SetFeeAmount(FeeAmount int64) error {
	FeeAmountbytes, err := json.Marshal(FeeAmount)
	if err != nil {
		storelog.Error("SetFeeAmount", "marshal FeeAmount error", err)
		return types.ErrMarshal
	}

	err = ws.GetDB().SetSync(CalcWalletPassKey(), FeeAmountbytes)
	if err != nil {
		storelog.Error("SetFeeAmount", "SetSync error", err)
		return err
	}
	return nil
}

// GetFeeAmount 获取手续费
func (ws *walletStore) GetFeeAmount(minFee int64) int64 {
	FeeAmountbytes, err := ws.Get(CalcWalletPassKey())
	if FeeAmountbytes == nil || err != nil {
		storelog.Debug("GetFeeAmount", "Get from db error", err)
		return minFee
	}
	var FeeAmount int64
	err = json.Unmarshal(FeeAmountbytes, &FeeAmount)
	if err != nil {
		storelog.Error("GetFeeAmount", "json unmarshal error", err)
		return minFee
	}
	return FeeAmount
}

// SetWalletPassword 设置钱包的密码
func (ws *walletStore) SetWalletPassword(newpass string) {
	err := ws.GetDB().SetSync(CalcWalletPassKey(), []byte(newpass))
	if err != nil {
		storelog.Error("SetWalletPassword", "SetSync error", err)
	}
}

// GetWalletPassword 获取钱包的密码
func (ws *walletStore) GetWalletPassword() string {
	passwordbytes, err := ws.Get(CalcWalletPassKey())
	if passwordbytes == nil || err != nil {
		storelog.Error("GetWalletPassword", "Get from db error", err)
		return ""
	}
	return string(passwordbytes)
}
