package walletoperate

import (
	"math/rand"
	"sync"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	FuncMap = queue.FuncMap{}
)

// WalletOperate 钱包对业务插件提供服务的操作接口
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
	GetWalletAccounts() ([]*types.WalletAccountStore, error)
	GetTxDetailByHashs(ReqHashes *types.ReqHashes)
	GetWaitGroup() *sync.WaitGroup

	IsWalletLocked() bool
	GetRescanFlag() int32
	SetRescanFlag(flag int32)

	CheckWalletStatus() (bool, error)
	Nonce() int64
	RegisterMsgFunc(msgid int, fn queue.FN_MsgCallback)
}
