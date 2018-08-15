package ticket

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
		bizlog.Error("checkAddrIsInWallet", "GetAccountByAddr error", err)
		return false
	}
	return true
}
