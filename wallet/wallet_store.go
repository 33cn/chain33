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

func NewStore(db db.DB) *walletStore {
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

	ws.GetDB().SetSync(CalcWalletPassKey(), FeeAmountbytes)
	return nil
}

func (store *walletStore) GetFeeAmount(minFee int64) int64 {
	FeeAmountbytes, err := store.Get(CalcWalletPassKey())
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

func (store *walletStore) SetWalletPassword(newpass string) {
	store.GetDB().SetSync(CalcWalletPassKey(), []byte(newpass))
}

func (store *walletStore) GetWalletPassword() string {
	passwordbytes, err := store.Get(CalcWalletPassKey())
	if passwordbytes == nil || err != nil {
		storelog.Error("GetWalletPassword", "Get from db error", err)
		return ""
	}
	return string(passwordbytes)
}
