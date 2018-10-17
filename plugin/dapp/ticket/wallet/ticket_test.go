package wallet_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/consensus"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/p2p"
	_ "gitlab.33.cn/chain33/chain33/plugin"
	tickettypes "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	_ "gitlab.33.cn/chain33/chain33/system"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet"
)

var (
	strPrivKeys []string
	privKeys    []crypto.PrivKey
)

func init() {
	log.SetLogLevel("info")
	strPrivKeys = append(strPrivKeys,
		"4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01",
		"CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944",
		"B0BB75BC49A787A71F4834DA18614763B53A18291ECE6B5EDEC3AD19D150C3E7",
		"56942AD84CCF4788ED6DACBC005A1D0C4F91B63BCF0C99A02BE03C8DEAE71138",
		"2AFF1981291355322C7A6308D46A9C9BA311AA21D94F36B43FC6A6021A1334CF",
		"2116459C0EC8ED01AA0EEAE35CAC5C96F94473F7816F114873291217303F6989")

	for _, key := range strPrivKeys {
		privKeys = append(privKeys, getprivkey(key))
	}
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}

type chain33Mock struct {
	random *rand.Rand
	client queue.Client

	api     client.QueueProtocolAPI
	chain   *blockchain.BlockChain
	mem     *mempool.Mempool
	cs      queue.Module
	exec    *executor.Executor
	wallet  queue.Module
	network *p2p.P2p
	store   queue.Module
}

func (mock *chain33Mock) start() {
	q := queue.New("channel")
	cfg := config.InitCfg("testdata/chain33.test.toml")
	cfg.Title = "local"
	cfg.TestNet = true
	cfg.Consensus.Name = "ticket"
	types.SetTestNet(cfg.TestNet)
	types.SetTitle(cfg.Title)

	mock.random = rand.New(rand.NewSource(types.Now().UnixNano()))

	mock.chain = blockchain.New(cfg.BlockChain)
	mock.chain.SetQueueClient(q.Client())
	mock.exec = executor.New(cfg.Exec)
	mock.exec.SetQueueClient(q.Client())
	types.SetMinFee(0)
	mock.store = store.New(cfg.Store)
	mock.store.SetQueueClient(q.Client())
	mock.cs = consensus.New(cfg.Consensus)
	mock.cs.SetQueueClient(q.Client())
	mock.mem = mempool.New(cfg.MemPool)
	mock.mem.SetQueueClient(q.Client())
	mock.network = p2p.New(cfg.P2P)
	mock.network.SetQueueClient(q.Client())
	mock.api, _ = client.New(q.Client(), nil)
	cli := q.Client()

	mock.client = cli
	mock.wallet = wallet.New(cfg.Wallet)
	mock.wallet.SetQueueClient(cli)
}

func (mock *chain33Mock) stop() {
	mock.chain.Close()
	mock.store.Close()
	mock.mem.Close()
	mock.cs.Close()
	mock.exec.Close()
	mock.wallet.Close()
	mock.network.Close()
	mock.client.Close()
}

func (mock *chain33Mock) waitHeight(height int64, t *testing.T) {
	for {
		header, err := mock.api.GetLastHeader()
		require.Equal(t, err, nil)
		// 区块出块以后再执行下面的动作
		if header.Height >= height {
			break
		}
		time.Sleep(time.Second)
	}
}

func (mock *chain33Mock) getTicketList(addr string, t *testing.T) *tickettypes.ReplyTicketList {
	msg, err := mock.api.Query(types.TicketX, "TicketList", &tickettypes.TicketList{addr, 1})
	require.Equal(t, err, nil)
	return msg.(*tickettypes.ReplyTicketList)
}

func Test_WalletTicket(t *testing.T) {
	t.Log("Begin wallet ticket test")
	// 启动虚拟即
	mock := &chain33Mock{}
	mock.start()
	defer mock.stop()

	mock.waitHeight(0, t)
	ticketList := mock.getTicketList("12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv", t)
	if ticketList == nil {
		t.Error("getTicketList return nil")
		return
	}
	mock.api.Notify("consensus", types.EventFlushTicket, nil)

	mock.waitHeight(2, t)
	header, err := mock.api.GetLastHeader()
	require.Equal(t, err, nil)
	require.Equal(t, header.Height >= 2, true)
	t.Log("End wallet ticket test")
}
