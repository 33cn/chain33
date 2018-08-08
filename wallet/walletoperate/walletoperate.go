package walletoperate

import (
	"sync"

	"math/rand"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

type WalletOperate interface {
	GetAPI() client.QueueProtocolAPI
	GetMutex() *sync.Mutex
	GetDBStore() db.DB
	GetSignType() int
	GetPassword() string
	GetBlockHeight() int64
	GetRandom() *rand.Rand
	GetWalletDone() chan struct{}
	GetLastHeader() *types.Header

	IsWalletLocked() bool
	GetRescanFlag() int32
	SetRescanFlag(flag int32)

	CheckWalletStatus() (bool, error)
	Nonce() int64
	AddWaitGroup(delta int)
	WaitGroupDone()
}
