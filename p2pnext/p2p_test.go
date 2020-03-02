package p2pnext

import (
	"encoding/hex"

	"testing"

	"github.com/libp2p/go-libp2p-core/peer"

	l "github.com/33cn/chain33/common/log"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/wallet"

	"github.com/stretchr/testify/assert"
)

func init() {
	l.SetLogLevel("err")
}

func processMsg(q queue.Queue) {
	go func() {
		cfg := q.GetConfig()
		wcli := wallet.New(cfg)
		client := q.Client()
		wcli.SetQueueClient(client)
		//导入种子，解锁钱包
		password := "a12345678"
		seed := "cushion canal bitter result harvest sentence ability time steel basket useful ask depth sorry area course purpose search exile chapter mountain project ranch buffalo"
		saveSeedByPw := &types.SaveSeedByPw{Seed: seed, Passwd: password}
		_, err := wcli.GetAPI().ExecWalletFunc("wallet", "SaveSeed", saveSeedByPw)
		if err != nil {
			return
		}
		walletUnLock := &types.WalletUnLock{
			Passwd:         password,
			Timeout:        0,
			WalletOrTicket: false,
		}

		_, err = wcli.GetAPI().ExecWalletFunc("wallet", "WalletUnLock", walletUnLock)
		if err != nil {
			return
		}
	}()

	go func() {
		blockchainKey := "blockchain"
		client := q.Client()
		client.Sub(blockchainKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetBlocks:
				if req, ok := msg.GetData().(*types.ReqBlocks); ok {
					if req.Start == 1 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventBlocks, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventBlocks, &types.BlockDetails{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}

			case types.EventGetHeaders:
				if req, ok := msg.GetData().(*types.ReqBlocks); ok {
					if req.Start == 10 {
						msg.Reply(client.NewMessage(blockchainKey, types.EventHeaders, &types.Transaction{}))
					} else {
						msg.Reply(client.NewMessage(blockchainKey, types.EventHeaders, &types.Headers{}))
					}
				} else {
					msg.ReplyErr("Do not support", types.ErrInvalidParam)
				}

			case types.EventGetLastHeader:
				msg.Reply(client.NewMessage("p2p", types.EventHeader, &types.Header{Height: 2019}))
			case types.EventGetBlockHeight:

				msg.Reply(client.NewMessage("p2p", types.EventReplyBlockHeight, &types.ReplyBlockHeight{Height: 2019}))

			}

		}

	}()

	go func() {
		mempoolKey := "mempool"
		client := q.Client()
		client.Sub(mempoolKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetMempoolSize:
				msg.Reply(client.NewMessage("p2p", types.EventMempoolSize, &types.MempoolSize{Size: 0}))
			}
		}
	}()
}

func NewP2p(cfg *types.Chain33Config, q queue.Queue) *P2P {
	p2p := New(cfg)
	p2p.SetQueueClient(q.Client())
	return p2p
}

func testP2PEvent(t *testing.T, qcli queue.Client) {

	msg := qcli.NewMessage("p2p", types.EventBlockBroadcast, &types.Block{})
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventTxBroadcast, &types.Transaction{})
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventFetchBlocks, &types.ReqBlocks{})
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventGetMempool, nil)
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventPeerInfo, nil)
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventGetNetInfo, nil)
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

	msg = qcli.NewMessage("p2p", types.EventFetchBlockHeaders, &types.ReqBlocks{})
	qcli.Send(msg, false)
	assert.True(t, qcli.Send(msg, false) == nil)

}

func testAddrBook(t *testing.T, p2p *P2P) {
	assert.True(t, p2p.addrbook.GetPrivkey() != nil)
	privstr, pubstr := p2p.addrbook.GetPrivPubKey()
	assert.NotEmpty(t, privstr)
	assert.NotEmpty(t, pubstr)
	pubkey, err := GenPubkey(privstr)
	assert.NoError(t, err)
	assert.Equal(t, pubkey, pubstr)
	privb, pubb, err := GenPrivPubkey()
	assert.NoError(t, err)

	genpriv := hex.EncodeToString(privb)
	genpub := hex.EncodeToString(pubb)
	p2p.addrbook.SaveKey(privstr, pubstr)
	assert.NotEqual(t, genpriv, privstr)
	assert.NotEqual(t, genpub, pubkey)
	var addrinfo peer.AddrInfo

	err = p2p.addrbook.SaveAddr([]peer.AddrInfo{addrinfo})
	assert.NoError(t, err)

}

func testRandStr(t *testing.T, n int) {
	rstr := RandStr(n)
	assert.True(t, len(rstr) == n)
}

func testgenAirDropAddr(t *testing.T, p2p *P2P) {
	err := p2p.genAirDropKeyFromWallet()
	assert.NoError(t, err)

}

func testP2PClose(t *testing.T, p2p *P2P) {
	p2p.Close()

}
func testP2PReStart(t *testing.T, p2p *P2P) {
	p2p.ReStart()
}

func testP2PWait(t *testing.T, p2p *P2P) {
	p2p.Wait()
}

func Test_p2p(t *testing.T) {

	cfg := types.NewChain33Config(types.ReadFile("../cmd/chain33/chain33.test.toml"))
	q := queue.New("channel")
	q.SetConfig(cfg)
	processMsg(q)
	p2p := NewP2p(cfg, q)
	testRandStr(t, 5)
	testP2PEvent(t, p2p.client)
	testAddrBook(t, p2p)

	testP2PClose(t, p2p)
	testP2PWait(t, p2p)
	testP2PReStart(t, p2p)

}
