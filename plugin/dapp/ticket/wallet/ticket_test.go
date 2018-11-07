package wallet_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	ticketwallet "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/wallet"
	"gitlab.33.cn/chain33/chain33/util/testnode"

	_ "gitlab.33.cn/chain33/chain33/plugin"
	_ "gitlab.33.cn/chain33/chain33/system"
)

func Test_WalletTicket(t *testing.T) {
	t.Log("Begin wallet ticket test")

	cfg, sub := testnode.GetDefaultConfig()
	cfg.Consensus.Name = "ticket"
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	defer mock33.Close()
	err := mock33.WaitHeight(0)
	assert.Nil(t, err)
	msg, err := mock33.GetAPI().Query(ty.TicketX, "TicketList", &ty.TicketList{"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv", 1})
	assert.Nil(t, err)
	ticketList := msg.(*ty.ReplyTicketList)
	assert.NotNil(t, ticketList)
	return
	ticketwallet.FlushTicket(mock33.GetAPI())
	err = mock33.WaitHeight(2)
	assert.Nil(t, err)
	header, err := mock33.GetAPI().GetLastHeader()
	require.Equal(t, err, nil)
	require.Equal(t, header.Height >= 2, true)
	t.Log("End wallet ticket test")
}
