package common

import (
	"math/rand"
	"reflect"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	funcMap         = make(map[string]map[string]reflect.Method)
	typeMap         = make(map[string]map[string]reflect.Type)
	valueMap        = make(map[string]reflect.Value)
	PolicyContainer = make(map[string]WalletBizPolicy)
)

func RegisterPolicy(key string, policy WalletBizPolicy) {
	if _, existed := PolicyContainer[key]; existed {
		panic("RegisterPolicy dup")
	}
	PolicyContainer[key] = policy
	RegisterEventCB(key, policy)
}

func RegisterEventCB(key string, obj interface{}) {
	if _, existed := funcMap[key]; existed {
		panic("RegisterEventCB dup")
	}
	funcMap[key], typeMap[key] = buildType(types.ListMethod(obj))
	valueMap[key] = reflect.ValueOf(obj)
}

func GetFunc(driver, name string) (reflect.Method, error) {
	funclist, ok := funcMap[driver]
	if !ok {
		return reflect.Method{}, types.ErrActionNotSupport
	}
	if f, ok := funclist[name]; ok {
		return f, nil
	}
	return reflect.Method{}, types.ErrActionNotSupport
}

func GetType(driver, name string) (reflect.Type, error) {
	typelist, ok := typeMap[driver]
	if !ok {
		return nil, types.ErrActionNotSupport
	}
	if t, ok := typelist[name]; ok {
		return t, nil
	}
	return nil, types.ErrActionNotSupport
}

func DecodeParam(driver, name string, in []byte) (reply types.Message, err error) {
	ty, err := GetType(driver, name)
	if err != nil {
		return nil, err
	}
	p := reflect.New(ty.In(1).Elem())
	queryin := p.Interface()
	if paramIn, ok := queryin.(proto.Message); ok {
		err = types.Decode(in, paramIn)
		return paramIn, err
	}
	return nil, types.ErrActionNotSupport
}

func CallFunc(driver, name string, in types.Message) (reply types.Message, err error) {
	f, err := GetFunc(driver, name)
	if err != nil {
		return nil, err
	}
	valueret := f.Func.Call([]reflect.Value{valueMap[driver], reflect.ValueOf(in)})
	if len(valueret) != 2 {
		return nil, types.ErrMethodNotFound
	}
	if !valueret[0].CanInterface() {
		return nil, types.ErrMethodNotFound
	}
	if !valueret[1].CanInterface() {
		return nil, types.ErrMethodNotFound
	}
	r1 := valueret[0].Interface()
	if r1 != nil {
		if r, ok := r1.(types.Message); ok {
			reply = r
		} else {
			return nil, types.ErrMethodReturnType
		}
	}
	//参数2
	r2 := valueret[1].Interface()
	if r2 != nil {
		if r, ok := r2.(error); ok {
			err = r
		} else {
			return nil, types.ErrMethodReturnType
		}
	}
	if reply == nil && err == nil {
		return nil, types.ErrActionNotSupport
	}
	return reply, err
}

func buildType(methods map[string]reflect.Method) (map[string]reflect.Method, map[string]reflect.Type) {
	tys := make(map[string]reflect.Type)
	ms := make(map[string]reflect.Method)
	for name, method := range methods {
		if !strings.HasPrefix(name, "On_") {
			continue
		}
		ty := method.Type
		if ty.NumIn() != 2 {
			continue
		}
		paramIn := ty.In(1)
		if paramIn.Kind() != reflect.Ptr {
			continue
		}
		p := reflect.New(ty.In(1).Elem())
		queryin := p.Interface()
		if _, ok := queryin.(proto.Message); !ok {
			continue
		}
		if ty.NumOut() != 2 {
			continue
		}
		if !ty.Out(0).AssignableTo(reflect.TypeOf((*proto.Message)(nil)).Elem()) {
			continue
		}
		if !ty.Out(1).AssignableTo(reflect.TypeOf((*error)(nil)).Elem()) {
			continue
		}
		tys[name] = ty
		ms[name] = method
	}
	return ms, tys
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
	GetRescanFlag() int32
	SetRescanFlag(flag int32)

	CheckWalletStatus() (bool, error)
	Nonce() int64

	WaitTx(hash []byte) *types.TransactionDetail
	WaitTxs(hashes [][]byte) (ret []*types.TransactionDetail)
	SendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error)
	SendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error)
}
