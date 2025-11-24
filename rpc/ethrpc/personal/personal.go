package personal

import (
	"errors"
	"fmt"

	"github.com/33cn/chain33/common/log/log15"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/queue"
	rpcclient "github.com/33cn/chain33/rpc/client"
	ctypes "github.com/33cn/chain33/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

var (
	log = log15.New("module", "ethrpc_persional")
)

type personalHandler struct {
	cli rpcclient.ChannelClient
	cfg *ctypes.Chain33Config
}

// NewPersonalAPI new persional object
func NewPersonalAPI(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) interface{} {
	p := &personalHandler{}
	p.cli.Init(c, api)
	p.cfg = cfg
	return p
}

// ListAccounts personal_listAccounts show all account
func (p *personalHandler) ListAccounts() ([]string, error) {
	req := &ctypes.ReqAccountList{WithoutBalance: true}
	msg, err := p.cli.ExecWalletFunc("wallet", "WalletGetAccountList", req)
	if err != nil {
		log.Error("WalletGetAccountList", "err", err)
		return nil, err
	}
	accountsList := msg.(*ctypes.WalletAccounts)
	var accounts []string
	for _, wallet := range accountsList.Wallets {
		if common.IsHex(wallet.GetAcc().GetAddr()) {
			accounts = append(accounts, wallet.GetAcc().GetAddr())
		}

	}

	return accounts, nil
}

// NewAccount  personal_newaccount
func (p *personalHandler) NewAccount(label string) (string, error) {
	req := &ctypes.ReqNewAccount{Label: label, AddressID: 2}
	resp, err := p.cli.ExecWalletFunc("wallet", "NewAccount", req)
	if err != nil {
		log.Error("personal_newaccount", "err", err)
		return "", err
	}
	account := resp.(*ctypes.WalletAccount)
	return account.Acc.Addr, nil
}

// UnlockAccount personal_unlockAccount
func (p *personalHandler) UnlockAccount(account, passwd string, duration int64) bool {
	req := &ctypes.WalletUnLock{Passwd: passwd, Timeout: duration}
	resp, err := p.cli.ExecWalletFunc("wallet", "WalletUnLock", req)
	if err != nil {
		return false
	}
	result := resp.(*ctypes.Reply).GetIsOk()
	return result
}

// ImportRawKey personal_importRawKey
func (p *personalHandler) ImportRawKey(keydata, label string) (string, error) {
	req := &ctypes.ReqWalletImportPrivkey{Privkey: keydata, Label: label, AddressID: 2}
	resp, err := p.cli.ExecWalletFunc("wallet", "WalletImportPrivkey", req)
	if err != nil {
		log.Error("personal_importRawKey", "err", err)
		return "", err
	}
	account := resp.(*ctypes.WalletAccount)
	return account.Acc.Addr, nil
}

// Sign personal_sign
func (p *personalHandler) Sign(data *hexutil.Bytes, address, passwd string) (string, error) {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(*data), *data)
	sha3Hash := common.Sha3([]byte(msg))
	//解锁钱包
	var key string
	if passwd != "" && !p.UnlockAccount("", passwd, 5) {
		return "", errors.New("unlock wallet faild")

	}
	//导出私钥
	reply, err := p.cli.ExecWalletFunc("wallet", "DumpPrivkey", &ctypes.ReqString{Data: address})
	if err != nil {
		log.Error("personal_sign", "err", err)
		return "", err
	}
	key = reply.(*ctypes.ReplyString).GetData()
	signKey, err := ethcrypto.ToECDSA(ethcommon.FromHex(key))
	if err != nil {
		return "", err
	}
	sig, err := ethcrypto.Sign(sha3Hash, signKey)
	if err != nil {
		return "", err
	}

	return ethcommon.Bytes2Hex(sig), nil
}
