package wallet

import (
	"gitlab.33.cn/chain33/chain33/common/db"
	wcom "gitlab.33.cn/chain33/chain33/wallet/common"
)

func NewStore(db db.DB) *ticketStore {
	return &ticketStore{Store: wcom.NewStore(db)}
}

// ticketStore ticket合约类型数据库操作类
type ticketStore struct {
	*wcom.Store
}

func (store *ticketStore) checkAddrIsInWallet(addr string) bool {
	if len(addr) == 0 {
		return false
	}
	acc, err := store.GetAccountByAddr(addr)
	if err != nil || acc == nil {
		return false
	}
	return true
}

func (store *ticketStore) SetAutoMinerFlag(flag int32) {
	if flag == 1 {
		store.Set(CalcWalletAutoMiner(), []byte("1"))
	} else {
		store.Set(CalcWalletAutoMiner(), []byte("0"))
	}
}

func (store *ticketStore) GetAutoMinerFlag() int32 {
	flag := int32(0)
	value, err := store.Get(CalcWalletAutoMiner())
	if err != nil {
		bizlog.Debug("GetAutoMinerFlag", "Get error", err)
		return flag
	}
	if value != nil && string(value) == "1" {
		flag = 1
	}
	return flag
}
