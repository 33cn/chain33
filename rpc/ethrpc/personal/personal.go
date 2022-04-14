package personal

import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	ctypes "github.com/33cn/chain33/types"
)

type PersonalApi struct {
	cli rpcclient.ChannelClient
	cfg *ctypes.Chain33Config
}

func NewPersonalApi(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) interface{} {
	p := &PersonalApi{}
	p.cli.Init(c, api)
	p.cfg = cfg
	return p
}

func (p *PersonalApi) ListAccounts() ([]string, error) {
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

func (p *PersonalApi) NewAccount(label string, addrID int32) (string, error) {

	req := &ctypes.ReqNewAccount{Label: label, AddressID: addrID}
	resp, err := p.cli.ExecWalletFunc("wallet", "NewAccount", req)
	if err != nil {
		return "", err
	}
	account := resp.(*ctypes.WalletAccount)
	return account.Acc.Addr, nil
}

func (p *PersonalApi) UnlockAccount(account, passwd string, duration int64) bool {
	req := &ctypes.WalletUnLock{Passwd: passwd, Timeout: duration}
	resp, err := p.cli.ExecWalletFunc("wallet", "WalletUnLock", req)
	if err != nil {
		return false
	}
	result := resp.(*ctypes.Reply).GetIsOk()
	return result
}

func (p *PersonalApi) ImportRawKey(keydata, label string) (string, error) {
	req := &ctypes.ReqWalletImportPrivkey{Privkey: keydata, Label: label}
	resp, err := p.cli.ExecWalletFunc("wallet", "WalletImportPrivkey", req)
	if err != nil {
		return "", err
	}
	account := resp.(*ctypes.WalletAccount)
	return account.Acc.Addr, nil
}
