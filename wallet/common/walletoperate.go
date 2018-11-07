package common

import (
	"math/rand"
	"reflect"
	"sync"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	QueryData       = types.NewQueryData("On_")
	PolicyContainer = make(map[string]WalletBizPolicy)
)

func Init(wallet WalletOperate, sub map[string][]byte) {
	for k, v := range PolicyContainer {
		v.Init(wallet, sub[k])
	}
}

func RegisterPolicy(key string, policy WalletBizPolicy) {
	if _, existed := PolicyContainer[key]; existed {
		panic("RegisterPolicy dup")
	}
	PolicyContainer[key] = policy
	QueryData.Register(key, policy)
	QueryData.SetThis(key, reflect.ValueOf(policy))
}

// WalletOperate 钱包对业务插件提供服务的操作接口
type WalletOperate interface {
	RegisterMineStatusReporter(reporter MineStatusReport) error

	GetAPI() client.QueueProtocolAPI
	GetMutex() *sync.Mutex
	GetDBStore() db.DB
	GetSignType() int
	GetPassword() string
	GetBlockHeight() int64
	GetRandom() *rand.Rand
	GetWalletDone() chan struct{}
	GetLastHeader() *types.Header
	GetTxDetailByHashs(ReqHashes *types.ReqHashes)
	GetWaitGroup() *sync.WaitGroup
	GetAllPrivKeys() ([]crypto.PrivKey, error)
	GetWalletAccounts() ([]*types.WalletAccountStore, error)
	GetPrivKeyByAddr(addr string) (crypto.PrivKey, error)
	GetConfig() *types.Wallet
	GetBalance(addr string, execer string) (*types.Account, error)

	IsWalletLocked() bool
	IsClose() bool
	IsCaughtUp() bool
	AddrInWallet(addr string) bool

	CheckWalletStatus() (bool, error)
	Nonce() int64

	WaitTx(hash []byte) *types.TransactionDetail
	WaitTxs(hashes [][]byte) (ret []*types.TransactionDetail)
	SendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error)
	SendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error)
}
