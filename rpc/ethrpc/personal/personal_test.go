package personal

import (
	clientMocks "github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/queue"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

var (
	personalObj *personalHandler
	qapi   *clientMocks.QueueProtocolAPI
	q      = queue.New("test")

)
func init() {
	qapi = &clientMocks.QueueProtocolAPI{}
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	q.SetConfig(cfg)
	personalObj = &personalHandler{}
	personalObj.cfg = cfg
	personalObj.cli.Init(q.Client(), qapi)

}

func TestPersonalHandler_ListAccounts(t *testing.T) {
	var account = &ctypes.WalletAccount{Acc: &ctypes.Account{Addr: "0xa42431Da868c58877a627CC71Dc95F01bf40c196"}, Label: "test1"}
	var account2 = &ctypes.WalletAccount{Acc: &ctypes.Account{Addr: "0xBe807Dddb074639cD9fA61b47676c064fc50D62C"}, Label: "test2"}
	var resp = &ctypes.WalletAccounts{
		Wallets: []*ctypes.WalletAccount{account, account2},
	}
	req := &ctypes.ReqAccountList{WithoutBalance: true}
	qapi.On("ExecWalletFunc", "wallet", "WalletGetAccountList", req).Return(resp, nil)
	accs,err:= personalObj.ListAccounts()
	if err!=nil{
		t.Log(err)
		return
	}

	t.Log("accs",accs)
	assert.Equal(t, 2,len(accs))


}

func TestPersonalHandler_NewAccount(t *testing.T) {
	req := &ctypes.ReqNewAccount{Label: "test", AddressID: 2}
	resp:=&ctypes.WalletAccount{Acc: &ctypes.Account{Addr: "0xa42431Da868c58877a627CC71Dc95F01bf40c196"}, Label: "test1"}
	qapi.On("ExecWalletFunc", "wallet", "NewAccount",req ).Return(resp, nil)
	acc,err:= personalObj.NewAccount("test")
	if err!=nil{
		t.Log(err)
		return
	}

	t.Log("acc",acc)
	assert.Equal(t, "0xa42431Da868c58877a627CC71Dc95F01bf40c196",acc)
}

func TestPersonalHandler_UnlockAccount(t *testing.T) {
	req := &ctypes.WalletUnLock{Passwd: "passwd", Timeout:60}
	var resp ctypes.Reply
	resp.IsOk=true
	qapi.On("ExecWalletFunc", "wallet", "WalletUnLock", req).Return(&resp, nil)
	ok:= personalObj.UnlockAccount("0xa42431Da868c58877a627CC71Dc95F01bf40c196","passwd",60)
	assert.Equal(t, true,ok)
}


func TestPersonalHandler_ImportRawKey(t *testing.T) {
	var key [32]byte
	rd:= rand.New(rand.NewSource(time.Now().UnixNano()))
	rd.Read(key[:])
	label:="test"
	req := &ctypes.ReqWalletImportPrivkey{Privkey: common.Bytes2Hex(key[:]), Label: label}
	qapi.On("ExecWalletFunc", "wallet", "WalletImportPrivkey", req).Return(&ctypes.WalletAccount{
		Acc: &ctypes.Account{Addr: "0xa42431Da868c58877a627CC71Dc95F01bf40c196"}, Label: label,
	}, nil)

	acc,err:= personalObj.ImportRawKey(common.Bytes2Hex(key[:]),label)
	assert.Nil(t, err)
	assert.Equal(t, "0xa42431Da868c58877a627CC71Dc95F01bf40c196",acc)
}