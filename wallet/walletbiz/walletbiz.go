package walletbiz

import (
	"sync"

	"math/rand"

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
	AddWaitGroup(delta int)
	WaitGroupDone()

	IsWalletLocked() bool
	GetBlockHeight() int64
	GetRandom() *rand.Rand
}
