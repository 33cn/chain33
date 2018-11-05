package para

import (
	"math/rand"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/p2p"

	//_ "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross"
	pp "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/executor"
	//"gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/rpc"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	_ "gitlab.33.cn/chain33/chain33/system"
	"gitlab.33.cn/chain33/chain33/types"
	typesmocks "gitlab.33.cn/chain33/chain33/types/mocks"
)

var random *rand.Rand

func init() {
	types.Init("user.p.para.", nil)
	pp.Init("paracross", nil)
	random = rand.New(rand.NewSource(types.Now().UnixNano()))
	consensusInterval = 2
	log.SetLogLevel("debug")
}

type suiteParaCommitMsg struct {
	// Include our basic suite logic.
	suite.Suite
	para    *ParaClient
	grpcCli *typesmocks.Chain33Client
	q       queue.Queue
	block   *blockchain.BlockChain
	exec    *executor.Executor
	store   queue.Module
	mem     *mempool.Mempool
	network *p2p.P2p
}

func initConfigFile() (*types.Config, *types.ConfigSubModule) {
	cfg, sub := types.InitCfg("../../../plugin/dapp/paracross/cmd/build/chain33.para.test.toml")
	return cfg, sub
}

func (s *suiteParaCommitMsg) initEnv(cfg *types.Config, sub *types.ConfigSubModule) {
	q := queue.New("channel")
	s.q = q
	//api, _ = client.New(q.Client(), nil)

	s.block = blockchain.New(cfg.BlockChain)
	s.block.SetQueueClient(q.Client())

	s.exec = executor.New(cfg.Exec, sub.Exec)
	s.exec.SetQueueClient(q.Client())

	s.store = store.New(cfg.Store, sub.Store)
	s.store.SetQueueClient(q.Client())
	s.para = New(cfg.Consensus, sub.Consensus["para"]).(*ParaClient)
	s.grpcCli = &typesmocks.Chain33Client{}
	//data := &types.Int64{1}
	s.grpcCli.On("GetLastBlockSequence", mock.Anything, mock.Anything).Return(nil, errors.New("nil"))
	reply := &types.Reply{IsOk: true}
	s.grpcCli.On("IsSync", mock.Anything, mock.Anything).Return(reply, nil)
	result := &pt.ParacrossStatus{Height: -1}
	data := types.Encode(result)
	ret := &types.Reply{IsOk: true, Msg: data}
	s.grpcCli.On("QueryChain", mock.Anything, mock.Anything).Return(ret, nil).Maybe()
	s.grpcCli.On("SendTransaction", mock.Anything, mock.Anything).Return(reply, nil).Maybe()
	s.para.grpcClient = s.grpcCli
	s.para.SetQueueClient(q.Client())

	s.mem = mempool.New(cfg.MemPool)
	s.mem.SetQueueClient(q.Client())
	s.mem.SetSync(true)
	s.mem.WaitPollLastHeader()

	s.network = p2p.New(cfg.P2P)
	s.network.SetQueueClient(q.Client())

	s.para.wg.Add(1)
	go walletProcess(q, s.para)

}

func walletProcess(q queue.Queue, para *ParaClient) {
	defer para.wg.Done()

	client := q.Client()
	client.Sub("wallet")

	for {
		select {
		case <-para.commitMsgClient.quit:
			return
		case msg := <-client.Recv():
			if msg.Ty == types.EventDumpPrivkey {
				msg.Reply(client.NewMessage("", types.EventHeader, &types.ReplyString{"6da92a632ab7deb67d38c0f6560bcfed28167998f6496db64c258d5e8393a81b"}))
			}
		}
	}

}

func (s *suiteParaCommitMsg) SetupSuite() {
	s.initEnv(initConfigFile())
}

func (s *suiteParaCommitMsg) TestRun_1() {
	//s.testGetBlock()
	lastBlock, err := s.para.RequestLastBlock()
	if err != nil {
		plog.Error("para test", "err", err.Error())
	}
	plog.Info("para test---------", "last height", lastBlock.Height)
	s.para.createBlock(lastBlock, nil, 0, getMainBlock(1, lastBlock.BlockTime+1))
	lastBlock, err = s.para.RequestLastBlock()
	if err != nil {
		plog.Error("para test--2", "err", err.Error())
	}
	plog.Info("para test---------", "last height", lastBlock.Height)
	s.para.createBlock(lastBlock, nil, 1, getMainBlock(2, lastBlock.BlockTime+1))
	time.Sleep(time.Second * 3)
	lastBlock, err = s.para.RequestLastBlock()
	s.para.DelBlock(lastBlock, 2)
	time.Sleep(time.Second * 3)
}

func TestRunSuiteParaCommitMsg(t *testing.T) {
	log := new(suiteParaCommitMsg)
	suite.Run(t, log)
}

func (s *suiteParaCommitMsg) TearDownSuite() {
	time.Sleep(time.Second * 5)
	s.block.Close()
	s.para.Close()
	s.exec.Close()
	s.store.Close()
	s.mem.Close()
	s.network.Close()
	s.q.Close()

}

func getMainBlock(height int64, BlockTime int64) *types.Block {
	return &types.Block{
		Height:    height,
		BlockTime: BlockTime,
	}
}
