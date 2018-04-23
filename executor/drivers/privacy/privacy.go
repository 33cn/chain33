package privacy

/*
privacy执行器支持隐私交易的执行，

主要提供操作有以下几种：
1）公开地址转账到一次性地址中，即：public address -> one-time addrss
2）隐私转账，隐私余额会被继续转账到一次性地址中 one-time address -> one-time address；
3）隐私余额转账到公开地址中， 即：one-time address -> public address

操作流程：
1）如果需要进行coin或token的隐私转账，则需要首先将其balance转账至privacy合约账户中；
2）此时用户可以发起隐私交易，在交易信息中指定接收的公钥对(A,B),执行成功之后，balance会被存入到一次性地址中；
3）发起交易，

*/

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var privacylog = log.New("module", "execs.privacy")

func init() {
	t := newPrivacy()
	//drivers.Register(t.GetName(), t, types.ForkV5_add_privacy)
	drivers.Register(t.GetName(), t, 0)
}

type privacy struct {
	drivers.DriverBase
}

func newPrivacy() *privacy {
	t := &privacy{}
	t.SetChild(t)
	return t
}

func (p *privacy) GetName() string {
	return "privacy"
}

func (p *privacy) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	_, err := p.DriverBase.Exec(tx, index)
	if err != nil {
		return nil, err
	}
	var action types.PrivacyAction
	err = types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}

	privacylog.Info("Privacy exec", "action type", action.Ty)
	if action.Ty == types.ActionPublic2Privacy && action.GetPublic2Privacy() != nil {
		public2Privacy := action.GetPublic2Privacy()
		if !public2Privacy.Op4Token {
			coinsAccount := p.GetCoinsAccount()
			from := account.From(tx).String()
			return coinsAccount.ExecTransfer(from, tx.To, p.GetAddr(), public2Privacy.Amount)
		} else {
			//token 转账操作

		}

	} else if action.Ty == types.ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		privacy2Privacy := action.GetPrivacy2Privacy()
		if !privacy2Privacy.Op4Token {
			from := account.From(tx).String()
			coinsAccount := p.GetCoinsAccount()
			return coinsAccount.ExecTransfer(from, tx.To, p.GetAddr(), privacy2Privacy.Amount)
		} else {			//token 转账操作

		}

	} else if action.Ty == types.ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		privacy2public := action.GetPrivacy2Public()
		if !privacy2public.Op4Token {
			from := account.From(tx).String()
			coinsAccount := p.GetCoinsAccount()
			return coinsAccount.ExecTransfer(from, tx.To, p.GetAddr(), privacy2public.Amount)
		} else {			//token 转账操作

		}

	}
	return nil, types.ErrActionNotSupport
}


func (p *privacy) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := p.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	return set, nil
}

func (c *privacy) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := c.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	return set, nil
}