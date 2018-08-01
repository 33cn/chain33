package para

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"gitlab.33.cn/chain33/chain33/types"
	typesmocks "gitlab.33.cn/chain33/chain33/types/mocks"
	qmocks "gitlab.33.cn/chain33/chain33/queue/mocks"
	"github.com/stretchr/testify/mock"
	"gitlab.33.cn/chain33/chain33/queue"
	"time"
	"gitlab.33.cn/chain33/chain33/common"
)

type suiteParaCommitMsg struct {
	// Include our basic suite logic.
	suite.Suite
	para      *ParaClient
	qClient   *qmocks.Client
	grpcCli   *typesmocks.GrpcserviceClient

}



func (s *suiteParaCommitMsg) SetupSuite() {

	cfg := &types.Consensus{
		ParaRemoteGrpcClient:"127.0.0.1:8106",
		StartHeight:345850,
		ParaAccount:"14KEKbYtKKQm4wMthSK9J4La4nAiidGozt",
	}
	s.para = New(cfg)
	s.grpcCli = &typesmocks.GrpcserviceClient{}
	s.para.grpcClient = s.grpcCli
	s.qClient = &qmocks.Client{}
	s.para.InitClient(s.qClient, func() {
	})
	msg := queue.Message{}
	s.qClient.On("NewMessage","wallet",mock.Anything,mock.Anything).Return(msg).Once()
	s.qClient.On("Send",msg,true).Return(nil).Once()
	reply := &types.ReplyStr{Replystr:"CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"}
	msg2 := queue.Message{Data:reply}
	s.qClient.On("Wait",msg).Return(msg2,nil).Once()
	go s.para.commitMsgClient.handler()



}

func (s *suiteParaCommitMsg) TestRun_1()  {
	result := &types.ReceiptParacrossDone{
		Height:3,
	}
	data := types.Encode(result)
	reply := &types.Reply{
		IsOk:true,
		Msg:data,
	}
	s.grpcCli.On("SendTransaction",mock.Anything,mock.Anything).Return(reply,nil)
	s.grpcCli.On("IsSync",mock.Anything,mock.Anything).Return(reply,nil)
	s.grpcCli.On("QueryChain",mock.Anything,mock.Anything).Return(nil,types.ErrNotFound).Once()

	s.grpcCli.On("QueryChain",mock.Anything,mock.Anything).Return(reply,nil)
	//to wait 1st consensus tick
	time.Sleep(time.Second*17)
	s.addMsg_1()


	time.Sleep(time.Second*3)
	//s.T().Log("currentTx",s.para.commitMsgClient.currentTx)
	s.NotNil(s.para.commitMsgClient.currentTx)
	currentTx1 := s.para.commitMsgClient.currentTx
	s.mainBlockAdd()
	//s.T().Log("currentTx--main block added",s.para.commitMsgClient.currentTx)
	s.addMsg_2()
	s.addMsg_3()
	s.addMsg_4()
	time.Sleep(time.Second*1)
	currentTx2 := s.para.commitMsgClient.currentTx
	s.NotEqual(currentTx1,currentTx2)
	//s.T().Log("currentTx-2",currentTx2)
	s.Assert().True(s.para.commitMsgClient.waitingTx)
	s.delMsg_4()
	time.Sleep(time.Second*1)
	currentTx3 := s.para.commitMsgClient.currentTx
	//s.T().Log("currentTx-3",currentTx3)
	s.NotEqual(currentTx3,currentTx2)
	s.mainBlockAdd()
	time.Sleep(time.Second*1)
}



func TestRunSuiteParaCommitMsg(t *testing.T) {
	log := new(suiteParaCommitMsg)
	suite.Run(t, log)
}

func (s *suiteParaCommitMsg) TearDownSuite() {
	time.Sleep(time.Second*5)
	s.qClient.On("Close").Return(nil)
	s.para.Close()
}

func (s *suiteParaCommitMsg) mainBlockAdd()  {

	tx2 := types.Transaction {
		Execer:[]byte("user.p.guodun.token"),
		Payload:[]byte{4,4},
		Nonce:2,
	}

	data, err := common.FromHex(s.para.commitMsgClient.currentTx)
	if err != nil {
		plog.Error("mainBlockAdd targetTx", "tx", s.para.commitMsgClient.currentTx, "err", err.Error())
		return
	}
	var decodeTx types.Transaction
	types.Decode(data, &decodeTx)

	block := &types.Block{
		Height:4,
		Txs:[]*types.Transaction{&decodeTx, &tx2},
	}

	recep1 := &types.ReceiptData{
		Ty:types.ExecOk,
	}
	recep2 := &types.ReceiptData{
		Ty:types.ExecOk,
	}
	blockDetail := &types.BlockDetail{
		Block:block,
		Receipts:[]*types.ReceiptData{recep1,recep2},
		PrevStatusHash:[]byte("1234"),

	}


	s.para.commitMsgClient.onMainBlockAdded(blockDetail)
}


func (s *suiteParaCommitMsg) addMsg_1()  {
	s.addMsg(int64(1))
}

func (s *suiteParaCommitMsg) addMsg_2()  {
	s.addMsg(int64(2))
}

func (s *suiteParaCommitMsg) addMsg_3()  {
	s.addMsg(int64(3))
}


func (s *suiteParaCommitMsg) addMsg_4()  {
	s.addMsg(int64(4))
}

func (s *suiteParaCommitMsg) addMsg(height int64)  {
	tx1 := types.Transaction {
		Execer:[]byte("user.p.guodun.token"),
		Payload:[]byte{1,2},
		Nonce:1,
	}

	tx2 := types.Transaction {
		Execer:[]byte("user.p.guodun.token"),
		Payload:[]byte{3,4},
		Nonce:2,
	}

	block := &types.Block{
		Height:height,
		Txs:[]*types.Transaction{&tx1, &tx2},
	}
	var oriTxHashs [][]byte
	for _, tx := range block.Txs {
		oriTxHashs = append(oriTxHashs, tx.Hash())
	}

	recep1 := &types.ReceiptData{
		Ty:types.ExecOk,
	}
	recep2 := &types.ReceiptData{
		Ty:types.ExecOk,
	}
	blockDetail := &types.BlockDetail{
		Block:block,
		Receipts:[]*types.ReceiptData{recep1,recep2},
		PrevStatusHash:[]byte("1234"),

	}

	commitMsg := &CommitMsg{
		initTxHashs:   oriTxHashs,
		mainBlockHash: []byte("1234"),
		blockDetail:   blockDetail,
	}
	s.para.commitMsgClient.onBlockAdded(commitMsg)
}


func (s *suiteParaCommitMsg) delMsg_4()  {
	tx1 := types.Transaction {
		Execer:[]byte("user.p.guodun.token"),
		Payload:[]byte{1,2},
		Nonce:1,
	}

	tx2 := types.Transaction {
		Execer:[]byte("user.p.guodun.token"),
		Payload:[]byte{3,4},
		Nonce:2,
	}

	block := &types.Block{
		Height:4,
		Txs:[]*types.Transaction{&tx1, &tx2},
	}
	var oriTxHashs [][]byte
	for _, tx := range block.Txs {
		oriTxHashs = append(oriTxHashs, tx.Hash())
	}

	recep1 := &types.ReceiptData{
		Ty:types.ExecOk,
	}
	recep2 := &types.ReceiptData{
		Ty:types.ExecOk,
	}
	blockDetail := &types.BlockDetail{
		Block:block,
		Receipts:[]*types.ReceiptData{recep1,recep2},
		PrevStatusHash:[]byte("1234"),

	}

	commitMsg := &CommitMsg{
		initTxHashs:   oriTxHashs,
		mainBlockHash: []byte("1234"),
		blockDetail:   blockDetail,
	}
	s.para.commitMsgClient.onBlockDeleted(commitMsg)
}

