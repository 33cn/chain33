package personal

import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	ctypes "github.com/33cn/chain33/types"
)

type personalHandler struct {
	cli rpcclient.ChannelClient
	cfg *ctypes.Chain33Config
}

//NewPersonalAPI new persional object
func NewPersonalAPI(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) interface{} {
	p := &personalHandler{}
	p.cli.Init(c, api)
	p.cfg = cfg
	return p
}

//ListAccounts personal_listAccounts show all account
func (p *personalHandler) ListAccounts() ([]string, error) {
	req := &ctypes.ReqAccountList{WithoutBalance: true}
	msg, err := p.cli.ExecWalletFunc("wallet", "WalletGetAccountList", req)
	if err != nil {
		return nil, err
	}
	accountsList := msg.(*ctypes.WalletAccounts)
	var accounts []string
	for _, wallet := range accountsList.Wallets {
		accounts = append(accounts, wallet.GetAcc().GetAddr())
	}

	return accounts, nil
}

//NewAccount  personal_newaccount
func (p *personalHandler) NewAccount(label string) (string, error) {
	req := &ctypes.ReqNewAccount{Label: label, AddressID: 2}
	resp, err := p.cli.ExecWalletFunc("wallet", "NewAccount", req)
	if err != nil {
		return "", err
	}
	account := resp.(*ctypes.WalletAccount)
	return account.Acc.Addr, nil
}

//UnlockAccount personal_unlockAccount
func (p *personalHandler) UnlockAccount(account, passwd string, duration int64) bool {
	req := &ctypes.WalletUnLock{Passwd: passwd, Timeout: duration}
	resp, err := p.cli.ExecWalletFunc("wallet", "WalletUnLock", req)
	if err != nil {
		return false
	}
	result := resp.(*ctypes.Reply).GetIsOk()
	return result
}

//ImportRawKey personal_importRawKey
func (p *personalHandler) ImportRawKey(keydata, label string) (string, error) {
	req := &ctypes.ReqWalletImportPrivkey{Privkey: keydata, Label: label}
	resp, err := p.cli.ExecWalletFunc("wallet", "WalletImportPrivkey", req)
	if err != nil {
		return "", err
	}
	account := resp.(*ctypes.WalletAccount)
	return account.Acc.Addr, nil
}
