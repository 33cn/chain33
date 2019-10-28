package p2p

import (
	"encoding/hex"
	"sync/atomic"
	"time"

	"os"
	"strings"
	"testing"

	l "github.com/33cn/chain33/common/log"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/wallet"

	//"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	testChannel = int32(119)
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
		msgSaveEmpty := client.NewMessage("wallet", types.EventSaveSeed, saveSeedByPw)
		client.Send(msgSaveEmpty, true)
		_, err := client.Wait(msgSaveEmpty)
		if err != nil {
			return
		}
		walletUnLock := &types.WalletUnLock{
			Passwd:         password,
			Timeout:        0,
			WalletOrTicket: false,
		}
		msgUnlock := client.NewMessage("wallet", types.EventWalletUnLock, walletUnLock)
		client.Send(msgUnlock, true)
		_, err = client.Wait(msgUnlock)
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

//new p2p
func newP2p(cfg *types.Chain33Config, port int32, dbpath string, q queue.Queue) *P2p {
	pcfg := cfg.GetModuleConfig().P2P
	pcfg.Port = port
	pcfg.Enable = true
	pcfg.DbPath = dbpath
	pcfg.DbCache = 4
	pcfg.Channel = testChannel
	pcfg.ServerStart = true
	pcfg.Driver = "leveldb"

	p2pcli := New(cfg)
	p2pcli.node.nodeInfo.addrBook.initKey()
	privkey, _ := p2pcli.node.nodeInfo.addrBook.GetPrivPubKey()
	p2pcli.node.nodeInfo.addrBook.bookDb.Set([]byte(privKeyTag), []byte(privkey))
	p2pcli.node.nodeInfo.SetServiceTy(7)
	p2pcli.SetQueueClient(q.Client())
	return p2pcli
}

//free P2p
func freeP2p(p2p *P2p) {
	p2p.Close()
	if err := os.RemoveAll(p2p.cfg.DbPath); err != nil {
		log.Error("removeTestDbErr", "err", err)
	}
}

func testP2PEvent(t *testing.T, qcli queue.Client) {
	msg := qcli.NewMessage("p2p", types.EventBlockBroadcast, &types.Block{})
	qcli.Send(msg, false)

	msg = qcli.NewMessage("p2p", types.EventTxBroadcast, &types.Transaction{})
	qcli.Send(msg, false)
	msg = qcli.NewMessage("p2p", types.EventFetchBlocks, &types.ReqBlocks{})
	qcli.Send(msg, false)

	msg = qcli.NewMessage("p2p", types.EventGetMempool, nil)
	qcli.Send(msg, false)

	msg = qcli.NewMessage("p2p", types.EventPeerInfo, nil)
	qcli.Send(msg, false)

	msg = qcli.NewMessage("p2p", types.EventGetNetInfo, nil)
	qcli.Send(msg, false)

	msg = qcli.NewMessage("p2p", types.EventFetchBlockHeaders, &types.ReqBlocks{})
	qcli.Send(msg, false)

}
func testNetInfo(t *testing.T, p2p *P2p) {
	p2p.node.nodeInfo.IsNatDone()
	p2p.node.nodeInfo.SetNatDone()
	p2p.node.nodeInfo.Get()
	p2p.node.nodeInfo.Set(p2p.node.nodeInfo)
	assert.NotNil(t, p2p.node.nodeInfo.GetListenAddr())
	assert.NotNil(t, p2p.node.nodeInfo.GetExternalAddr())
}

//测试Peer
func testPeer(t *testing.T, p2p *P2p, q queue.Queue) {
	cfg := types.NewChain33Config(types.ReadFile("../cmd/chain33/chain33.test.toml"))
	conn, err := grpc.Dial("localhost:53802", grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	assert.Nil(t, err)
	defer conn.Close()

	remote, err := NewNetAddressString("127.0.0.1:53802")
	assert.Nil(t, err)

	localP2P := newP2p(cfg, 43802, "testPeer", q)
	defer freeP2p(localP2P)

	t.Log(localP2P.node.CacheBoundsSize())
	t.Log(localP2P.node.GetCacheBounds())

	localP2P.node.RemoveCachePeer("localhost:12345")
	peer, err := P2pComm.dialPeer(remote, localP2P.node)
	assert.Nil(t, err)
	defer peer.Close()
	peer.MakePersistent()
	localP2P.node.addPeer(peer)
	var info *innerpeer
	t.Log("WaitRegisterPeerStart...")
	trytime := 0
	for info == nil || info.p2pversion == 0 {
		trytime++
		time.Sleep(time.Millisecond * 100)
		//info = p2p.node.server.p2pserver.getInBoundPeerInfo("127.0.0.1:43802")
		info = p2p.node.server.p2pserver.getInBoundPeerInfo(peer.GetPeerName())
		if trytime > 100 {
			return
		}
	}
	t.Log("WaitRegisterPeerStop...")
	p2pcli := NewNormalP2PCli()
	num, err := p2pcli.GetInPeersNum(peer)
	assert.Equal(t, 1, num)
	assert.Nil(t, err)
	tx1 := &types.Transaction{Execer: []byte("testTx1")}
	tx2 := &types.Transaction{Execer: []byte("testTx2")}
	localP2P.node.pubToPeer(&types.P2PTx{Tx: tx1}, peer.GetPeerName())
	p2p.node.server.p2pserver.pubToStream(&types.P2PTx{Tx: tx2}, info.name)
	t.Log("WaitRegisterTxFilterStart...")
	for !(txHashFilter.QueryRecvData(hex.EncodeToString(tx1.Hash())) &&
		txHashFilter.QueryRecvData(hex.EncodeToString(tx1.Hash()))) {
		time.Sleep(time.Millisecond * 10)
	}
	t.Log("WaitRegisterTxFilterStop")

	localP2P.node.AddCachePeer(peer)
	peer.GetRunning()
	peer.getIsRegister()
	peer.setIsRegister(false)
	localP2P.node.nodeInfo.FetchPeerInfo(localP2P.node)
	peers, infos := localP2P.node.GetActivePeers()
	assert.Equal(t, len(peers), len(infos))
	localP2P.node.flushNodePort(43803, 43802)

	localP2P.node.nodeInfo.peerInfos.SetPeerInfo(nil)
	localP2P.node.nodeInfo.peerInfos.GetPeerInfo("1222")
	t.Log(p2p.node.GetRegisterPeer("localhost:43802"))
	//测试发送Ping消息
	err = p2pcli.SendPing(peer, localP2P.node.nodeInfo)
	assert.Nil(t, err)

	//获取peer节点的被连接数
	pnum, err := p2pcli.GetInPeersNum(peer)
	assert.Nil(t, err)
	assert.Equal(t, 1, pnum)

	_, err = peer.GetPeerInfo()
	assert.Nil(t, err)
	//获取节点列表
	_, err = p2pcli.GetAddrList(peer)
	assert.Nil(t, err)

	_, err = p2pcli.SendVersion(peer, localP2P.node.nodeInfo)
	assert.Nil(t, err)

	t.Log(p2pcli.CheckPeerNatOk("localhost:53802", localP2P.node.nodeInfo))
	t.Log("checkself:", p2pcli.CheckSelf("loadhost:43803", localP2P.node.nodeInfo))
	_, err = p2pcli.GetAddr(peer)
	assert.Nil(t, err)

	localP2P.node.pubsub.FIFOPub(&types.P2PTx{Tx: &types.Transaction{}, Route: &types.P2PRoute{}}, "tx")
	localP2P.node.pubsub.FIFOPub(&types.P2PBlock{Block: &types.Block{}}, "block")
	//	//测试获取高度
	height, err := p2pcli.GetBlockHeight(localP2P.node.nodeInfo)
	assert.Nil(t, err)
	assert.Equal(t, int(height), 2019)
	assert.Equal(t, false, p2pcli.CheckSelf("localhost:53802", localP2P.node.nodeInfo))
	//测试下载
	job := NewDownloadJob(NewP2PCli(localP2P).(*Cli), []*Peer{peer})

	job.GetFreePeer(1)

	var ins []*types.Inventory
	var inv types.Inventory
	inv.Ty = msgBlock
	inv.Height = 2
	ins = append(ins, &inv)
	var bChan = make(chan *types.BlockPid, 256)
	job.syncDownloadBlock(peer, ins[0], bChan)
	respIns := job.DownloadBlock(ins, bChan)
	t.Log(respIns)
	job.ResetDownloadPeers([]*Peer{peer})
	t.Log(job.avalidPeersNum())
	job.setBusyPeer(peer.GetPeerName())
	job.setFreePeer(peer.GetPeerName())
	job.removePeer(peer.GetPeerName())
	job.CancelJob()

	peer.Close()
	localP2P.node.remove(peer.peerAddr.String())
}

//测试grpc 多连接
func testGrpcConns(t *testing.T) {
	var conns []*grpc.ClientConn

	for i := 0; i < maxSamIPNum; i++ {
		conn, err := grpc.Dial("localhost:53802", grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
		assert.Nil(t, err)

		cli := types.NewP2PgserviceClient(conn)
		_, err = cli.GetHeaders(context.Background(), &types.P2PGetHeaders{
			StartHeight: 0, EndHeight: 0, Version: 1002}, grpc.FailFast(true))
		assert.Equal(t, false, strings.Contains(err.Error(), "no authorized"))
		conns = append(conns, conn)
	}

	conn, err := grpc.Dial("localhost:53802", grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	assert.Nil(t, err)
	cli := types.NewP2PgserviceClient(conn)
	_, err = cli.GetHeaders(context.Background(), &types.P2PGetHeaders{
		StartHeight: 0, EndHeight: 0, Version: 1002}, grpc.FailFast(true))
	assert.Equal(t, true, strings.Contains(err.Error(), "no authorized"))

	conn.Close()
	for _, conn := range conns {
		conn.Close()
	}

}

//测试grpc 流多连接
func testGrpcStreamConns(t *testing.T, p2p *P2p) {

	conn, err := grpc.Dial("localhost:53802", grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	assert.Nil(t, err)
	cli := types.NewP2PgserviceClient(conn)
	var p2pdata types.P2PGetData
	resp, err := cli.GetData(context.Background(), &p2pdata)
	assert.Nil(t, err)
	_, err = resp.Recv()
	assert.Equal(t, true, strings.Contains(err.Error(), "no authorized"))

	ping, err := P2pComm.NewPingData(p2p.node.nodeInfo)
	assert.Nil(t, err)

	_, err = cli.ServerStreamSend(context.Background(), ping)
	assert.Nil(t, err)

	_, err = cli.ServerStreamRead(context.Background())
	assert.Nil(t, err)
	var emptyBlock types.P2PBlock

	_, err = cli.BroadCastBlock(context.Background(), &emptyBlock)
	assert.Equal(t, true, strings.Contains(err.Error(), "no authorized"))

	conn.Close()

}

func testP2pComm(t *testing.T, p2p *P2p) {

	addrs := P2pComm.AddrRouteble([]string{"localhost:53802"}, calcChannelVersion(testChannel))
	t.Log(addrs)
	i32 := P2pComm.BytesToInt32([]byte{0xff})
	t.Log(i32)
	_, _, err := P2pComm.GenPrivPubkey()
	assert.Nil(t, err)
	ping, err := P2pComm.NewPingData(p2p.node.nodeInfo)
	assert.Nil(t, err)
	assert.Equal(t, true, P2pComm.CheckSign(ping))
	assert.IsType(t, "string", P2pComm.GetLocalAddr())
	assert.Equal(t, 5, len(P2pComm.RandStr(5)))
}

func testAddrBook(t *testing.T, p2p *P2p) {

	prv, pub, err := P2pComm.GenPrivPubkey()
	if err != nil {
		t.Log(err.Error())
		return
	}

	t.Log("priv:", hex.EncodeToString(prv), "pub:", hex.EncodeToString(pub))

	pubstr, err := P2pComm.Pubkey(hex.EncodeToString(prv))
	if err != nil {
		t.Log(err.Error())
		return
	}
	t.Log("GenPubkey:", pubstr)

	addrBook := p2p.node.nodeInfo.addrBook
	addrBook.Size()
	addrBook.saveToDb()
	addrBook.GetPeerStat("locolhost:43802")
	addrBook.genPubkey(hex.EncodeToString(prv))
	assert.Equal(t, addrBook.genPubkey(hex.EncodeToString(prv)), pubstr)
	addrBook.Save()
	addrBook.GetAddrs()
	addrBook.ResetPeerkey("", "")
	privkey, _ := addrBook.GetPrivPubKey()
	assert.NotEmpty(t, privkey)
	addrBook.ResetPeerkey(hex.EncodeToString(prv), pubstr)
	resetkey, _ := addrBook.GetPrivPubKey()
	assert.NotEqual(t, resetkey, privkey)
}

func testRestart(t *testing.T, p2p *P2p) {
	client := p2p.client
	assert.False(t, p2p.isRestart())
	p2p.txFactory <- struct{}{}
	p2p.processEvent(client.NewMessage("p2p", types.EventTxBroadcast, &types.Transaction{}), 128, p2p.p2pCli.BroadCastTx)
	atomic.StoreInt32(&p2p.restart, 1)
	p2p.ReStart()
	atomic.StoreInt32(&p2p.restart, 0)
	p2p.ReStart()
}

func Test_p2p(t *testing.T) {
	cfg := types.NewChain33Config(types.ReadFile("../cmd/chain33/chain33.test.toml"))
	q := queue.New("channel")
	q.SetConfig(cfg)
	go q.Start()
	processMsg(q)
	p2p := newP2p(cfg, 53802, "testP2p", q)
	p2p.Wait()
	defer freeP2p(p2p)
	defer q.Close()
	testP2PEvent(t, q.Client())
	testNetInfo(t, p2p)
	testPeer(t, p2p, q)
	testGrpcConns(t)
	testGrpcStreamConns(t, p2p)
	testP2pComm(t, p2p)
	testAddrBook(t, p2p)
	testRestart(t, p2p)
}
