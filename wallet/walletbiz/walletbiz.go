package walletbiz

import (
	"sync"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/db"
)

type WalletBiz interface {
	GetAPI() client.QueueProtocolAPI
	GetMutex() *sync.Mutex
	GetDBStore() db.DB
	GetSignType() int
	GetPassword() string
	CheckWalletStatus() (bool, error)
	Nonce() int64

	IsWalletLocked() bool
}
