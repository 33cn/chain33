package wallet

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/consensus"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/p2p"
	tickettypes "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	_ "gitlab.33.cn/chain33/chain33/system"
	"gitlab.33.cn/chain33/chain33/types"
	wcom "gitlab.33.cn/chain33/chain33/wallet/common"
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

type walletMock struct {
	client   queue.Client
	api      client.QueueProtocolAPI
	db       db.DB
	reporter wcom.MineStatusReport
	cfg      *types.Wallet
	wg       sync.WaitGroup
	done     chan struct{}
	mtx      sync.Mutex

	walletStatus       bool
	walletAccountStore []*types.WalletAccountStore
}

func (mock *walletMock) init(cfg *types.Wallet) {
	mock.cfg = cfg
	mock.done = make(chan struct{})
	mock.db = db.NewDB("wallet", cfg.Driver, cfg.DbPath, cfg.DbCache)

	wcom.RegisterMsgFunc(types.EventWalletImportprivkey, mock.onWalletImportprivkey)
	wcom.RegisterMsgFunc(types.EventAddBlock, mock.onAddBlock)
	wcom.RegisterMsgFunc(types.EventGetWalletStatus, mock.onGetWalletStatus)
	wcom.RegisterMsgFunc(types.EventWalletGetTickets, mock.onWalletGetTickets)
}

func (mock *walletMock) processMessage() {
	defer mock.wg.Done()
	for msg := range mock.client.Recv() {
		funcExisted, topic, retty, reply, err := wcom.ProcessFuncMap(&msg)
		if funcExisted {
			if err != nil {
				msg.Reply(mock.api.NewMessage(topic, retty, err))
			} else {
				msg.Reply(mock.api.NewMessage(topic, retty, reply))
			}
		} else {
			fmt.Println("msg ty ", msg.Ty)
		}
	}
}

func (mock *walletMock) onWalletImportprivkey(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventWalletImportprivkey)

	return topic, retty, &types.WalletAccount{
		Label: "no award",
		Acc: &types.Account{
			Currency: 1000000000,
			Balance:  1000000000,
		},
	}, nil
}

func (mock *walletMock) onAddBlock(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventAddBlock)

	return topic, retty, nil, nil
}

func (mock *walletMock) onGetWalletStatus(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventGetWalletStatus)

	return topic, retty, &types.WalletStatus{
		IsWalletLock: false,
		IsAutoMining: true,
		IsHasSeed:    true,
		IsTicketLock: false,
	}, nil
}

func (mock *walletMock) getTicketList() *tickettypes.ReplyTicketList {
	reqaddr := &tickettypes.TicketList{"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv", 1}
	var req types.Query
	req.Execer = []byte("ticket")
	req.FuncName = "TicketList"
	req.Payload = types.Encode(reqaddr)
	msg, _ := mock.api.Query(&req)
	return msg.(*tickettypes.ReplyTicketList)
}

func (mock *walletMock) onWalletGetTickets(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventWalletTickets)
	var privKeysdata [][]byte
	for _, privkey := range privKeys {
		privKeysdata = append(privKeysdata, privkey.Bytes())
	}
	ticklist := mock.getTicketList()
	ticklist.Tickets = ticklist.Tickets[0:5000]
	tks := &tickettypes.ReplyWalletTickets{ticklist.Tickets, privKeysdata}
	return topic, retty, tks, nil
}

func (mock *walletMock) SetQueueClient(cli queue.Client) {
	mock.client = cli
	mock.client.Sub("wallet")
	mock.api, _ = client.New(cli, nil)

	mock.wg.Add(1)
	go mock.processMessage()
}

func (mock *walletMock) Close() {

}

func (mock *walletMock) RegisterMineStatusReporter(reporter wcom.MineStatusReport) error {
	mock.reporter = reporter
	return nil
}

func (mock *walletMock) GetAPI() client.QueueProtocolAPI {
	panic("implement me")
}

func (mock *walletMock) GetMutex() *sync.Mutex {
	return &mock.mtx
}

func (mock *walletMock) GetDBStore() db.DB {
	return mock.db
}

func (mock *walletMock) GetSignType() int {
	panic("implement me")
}

func (mock *walletMock) GetPassword() string {
	panic("implement me")
}

func (mock *walletMock) GetBlockHeight() int64 {
	panic("implement me")
}

func (mock *walletMock) GetRandom() *rand.Rand {
	panic("implement me")
}

func (mock *walletMock) GetWalletDone() chan struct{} {
	return mock.done
}

func (mock *walletMock) GetLastHeader() *types.Header {
	panic("implement me")
}

func (mock *walletMock) GetTxDetailByHashs(ReqHashes *types.ReqHashes) {
	panic("implement me")
}

func (mock *walletMock) GetWaitGroup() *sync.WaitGroup {
	return &mock.wg
}

func (mock *walletMock) GetAllPrivKeys() ([]crypto.PrivKey, error) {
	panic("implement me")
}

func (mock *walletMock) GetWalletAccounts() ([]*types.WalletAccountStore, error) {
	return mock.walletAccountStore, nil
}

func (mock *walletMock) GetPrivKeyByAddr(addr string) (crypto.PrivKey, error) {
	panic("implement me")
}

func (mock *walletMock) GetConfig() *types.Wallet {
	return mock.cfg
}

func (mock *walletMock) GetBalance(addr string, execer string) (*types.Account, error) {
	panic("implement me")
}

func (mock *walletMock) IsWalletLocked() bool {
	panic("implement me")
}

func (mock *walletMock) IsClose() bool {
	panic("implement me")
}

func (mock *walletMock) IsCaughtUp() bool {
	panic("implement me")
}

func (mock *walletMock) GetRescanFlag() int32 {
	panic("implement me")
}

func (mock *walletMock) SetRescanFlag(flag int32) {
	panic("implement me")
}

func (mock *walletMock) CheckWalletStatus() (bool, error) {
	return mock.walletStatus, nil
}

func (mock *walletMock) Nonce() int64 {
	panic("implement me")
}

func (mock *walletMock) WaitTx(hash []byte) *types.TransactionDetail {
	panic("implement me")
}

func (mock *walletMock) WaitTxs(hashes [][]byte) (ret []*types.TransactionDetail) {
	panic("implement me")
}

func (mock *walletMock) SendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error) {
	panic("implement me")
}

func (mock *walletMock) SendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error) {
	panic("implement me")
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
	cfg := config.InitCfg("../../../cmd/chain33/chain33.test.toml")
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
	wallet := &walletMock{
		client: cli,
		api:    mock.api,
	}
	mock.client = cli
	mock.wallet = wallet
	mock.wallet.SetQueueClient(cli)
	wallet.init(cfg.Wallet)
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
	var req types.Query
	req.Execer = types.ExecerTicket
	req.FuncName = "TicketList"
	req.Payload = types.Encode(&tickettypes.TicketList{addr, 1})
	msg, err := mock.api.Query(&req)
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
