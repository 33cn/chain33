package ethrpc

import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	ctypes "github.com/33cn/chain33/types"
	rpcclient "github.com/33cn/chain33/rpc/client"
)

type PersonalApi struct {
	cli rpcclient.ChannelClient
	cfg *ctypes.Chain33Config

}

func NewPersonalApi( cfg *ctypes.Chain33Config,c queue.Client,api client.QueueProtocolAPI) *PersonalApi {
	p :=&PersonalApi{}
	p.cli.Init(c,api)
	p.cfg=cfg
	return p
}

/**
#personal_listAccounts

req

```
{"jsonrpc":"2.0","method": "personal_listAccounts", "params": [],"id":1}
```

response

```
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        "19B5wna26upMNHpRf8vMeDMEDx3bkx72be",
        "1AGZT88duvemtxG1Fd37PcsEyRkt3gi5GP"
    ]
}
```
 */
func (p *PersonalApi) ListAccounts() ([]string, error){
	req := &ctypes.ReqAccountList{WithoutBalance: true}
	msg,err:= p.cli.ExecWalletFunc("wallet","WalletGetAccountList",req)
	if err!=nil{
		return nil,err
	}
	accountsList := msg.(*ctypes.WalletAccounts)
	var accounts  []string
	for _, wallet := range accountsList.Wallets {
		accounts = append(accounts, wallet.GetAcc().GetAddr())
	}

	return accounts, nil
}

/**
personal_newAccount

req

```
{
  "method":"personal_newAccount",
  "params":["test"],
  "id":1,
  "jsonrpc":"2.0"
}
```

response

```
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "1DuYwg7eVrBEYKaiZBFtsqNZWT4K923JwN"
}
```
 */
func (p *PersonalApi) NewAccount(lable string) (string, error){
	req := &ctypes.ReqNewAccount{Label:lable}
	resp, err := p.cli.ExecWalletFunc("wallet", "NewAccount", req)
	if err != nil {
		return "", err
	}
	account := resp.(*ctypes.WalletAccount)
	return account.Acc.Addr, nil
}

/**
# personal_unlockAccount

req

```
{
  "method":"personal_unlockAccount",
  "params":["1AGZT88duvemtxG1Fd37PcsEyRkt3gi5GP", "bty123456",10000],
  "id":1,
  "jsonrpc":"2.0"
}
```

response

```
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": true
}
 */
func (p *PersonalApi)UnlockAccount(account, passwd  string,  duration int64) bool {
	req := &ctypes.WalletUnLock{Passwd:passwd, Timeout: duration}
	resp, err := p.cli.ExecWalletFunc("wallet", "WalletUnLock", req)
	if err != nil {
		return false
	}
	result := resp.(*ctypes.Reply).GetIsOk()
	return result
}


/**
#personal_importRawKey

req

```
{
  "method":"personal_importRawKey",
  "params":["07b6e23c7d0e9385eafb7bf8127a63340533fbd4b89c**********99ba1f94", "label"],
  "id":1,
  "jsonrpc":"2.0"
}

```

response

```
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "12dyEw2De26DZH6bnWg1pNx72FiPirNkTU"
}
 */
func (p *PersonalApi)ImportRawKey(keydata, label string) (string, error) {
	req := &ctypes.ReqWalletImportPrivkey{Privkey: keydata, Label:label}
	resp, err := p.cli.ExecWalletFunc("wallet", "WalletImportPrivkey", req)
	if err != nil {
		return "", err
	}
	account := resp.(*ctypes.WalletAccount)
	return account.Acc.Addr, nil
}


