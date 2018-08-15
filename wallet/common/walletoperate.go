package common

import (
	"math/rand"
	"sync"

	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	FuncMap         = queue.FuncMap{}
	PolicyContainer = map[string]WalletBizPolicy{}
)

func RegisterPolicy(key string, policy WalletBizPolicy) error {
	if _, existed := PolicyContainer[key]; existed {
		return errors.New("PolicyTypeExisted")
	}
	PolicyContainer[key] = policy
	return nil
}

func RegisterMsgFunc(msgid int, fn queue.FN_MsgCallback) {
	if !FuncMap.IsInited() {
		FuncMap.Init()
	}
	FuncMap.Register(msgid, fn)
}

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
	GetAllPrivKeys() ([]crypto.PrivKey, error)

	IsWalletLocked() bool
	IsClose() bool
	GetRescanFlag() int32
	SetRescanFlag(flag int32)

	CheckWalletStatus() (bool, error)
	Nonce() int64

	SendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error)
}
