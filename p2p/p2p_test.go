package p2p

import (
	"bytes"
	"encoding/hex"

	"github.com/33cn/chain33/common/merkle"

	//"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

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
	q           queue.Queue
	p2pModule   *P2p
	dataDir     = "testdata"
	testChannel = int32(119)
	memTxList   []*types.Transaction
)

func init() {
	l.SetLogLevel("err")
	q = queue.New("channel")
	go q.Start()

	go func() {

		cfg, sub := types.InitCfg("../cmd/chain33/chain33.test.toml")
		wcli := wallet.New(cfg.Wallet, sub.Wallet)
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
			case types.EventTxListByHash:
				query := msg.Data.(*types.ReqTxHashList)
				var txs []*types.Transaction
				if !query.IsShortHash {
					txs = memTxList[:1]
				} else {
					txs = memTxList
				}
				msg.Reply(client.NewMessage("p2p", types.EventTxListByHash, &types.ReplyTxList{Txs: txs}))
			}
		}
	}()
	time.Sleep(time.Second)
	p2pModule = initP2p(53802, dataDir)
	p2pModule.Wait()

}

//初始化p2p模块
func initP2p(port int32, dbpath string) *P2p {
	cfg := new(types.P2P)
	cfg.Port = port
	cfg.Enable = true
	cfg.DbPath = dbpath
	cfg.DbCache = 4
	cfg.Channel = testChannel
	cfg.ServerStart = true
	cfg.Driver = "leveldb"

	p2pcli := New(cfg)
	p2pcli.node.nodeInfo.addrBook.initKey()
	privkey, _ := p2pcli.node.nodeInfo.addrBook.GetPrivPubKey()
	p2pcli.node.nodeInfo.addrBook.bookDb.Set([]byte(privKeyTag), []byte(privkey))
	p2pcli.node.nodeInfo.SetServiceTy(7)
	p2pcli.SetQueueClient(q.Client())

	return p2pcli
}

func TestP2PEvent(t *testing.T) {
	qcli := q.Client()
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
func TestNetInfo(t *testing.T) {
	p2pModule.node.nodeInfo.IsNatDone()
	p2pModule.node.nodeInfo.SetNatDone()
	p2pModule.node.nodeInfo.Get()
	p2pModule.node.nodeInfo.Set(p2pModule.node.nodeInfo)
	assert.NotNil(t, p2pModule.node.nodeInfo.GetListenAddr())
	assert.NotNil(t, p2pModule.node.nodeInfo.GetExternalAddr())
}

//测试Peer
func TestPeer(t *testing.T) {

	conn, err := grpc.Dial("localhost:53802", grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	assert.Nil(t, err)
	defer conn.Close()

	remote, err := NewNetAddressString("127.0.0.1:53802")
	assert.Nil(t, err)

	localP2P := initP2p(43802, "testdata2")
	defer os.RemoveAll("testdata2")
	defer localP2P.Close()

	t.Log(localP2P.node.CacheBoundsSize())
	t.Log(localP2P.node.GetCacheBounds())

	localP2P.node.RemoveCachePeer("localhost:12345")
	peer, err := P2pComm.dialPeer(remote, localP2P.node)
	assert.Nil(t, err)
	defer peer.Close()
	peer.MakePersistent()
	localP2P.node.addPeer(peer)
	time.Sleep(time.Second * 5)
	t.Log(peer.GetInBouns())
	t.Log(peer.version.GetVersion())
	assert.IsType(t, "string", peer.GetPeerName())

	localP2P.node.AddCachePeer(peer)
	peer.GetRunning()
	localP2P.node.natOk()
	localP2P.node.flushNodePort(43803, 43802)
	p2pcli := NewNormalP2PCli()
	localP2P.node.nodeInfo.peerInfos.SetPeerInfo(nil)
	localP2P.node.nodeInfo.peerInfos.GetPeerInfo("1222")
	t.Log(p2pModule.node.GetRegisterPeer("localhost:43802"))
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

	localP2P.node.pubsub.FIFOPub(&types.P2PTx{Tx: &types.Transaction{}}, "tx")
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
	os.Remove(dataDir)

}

func TestSortArr(t *testing.T) {
	var Inventorys = make(Invs, 0)
	for i := 100; i >= 0; i-- {
		var inv types.Inventory
		inv.Ty = 111
		inv.Height = int64(i)
		Inventorys = append(Inventorys, &inv)
	}
	sort.Sort(Inventorys)
}

//测试grpc 多连接
func TestGrpcConns(t *testing.T) {
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
func TestGrpcStreamConns(t *testing.T) {

	conn, err := grpc.Dial("localhost:53802", grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	assert.Nil(t, err)
	cli := types.NewP2PgserviceClient(conn)
	var p2pdata types.P2PGetData
	resp, err := cli.GetData(context.Background(), &p2pdata)
	assert.Nil(t, err)
	_, err = resp.Recv()
	assert.Equal(t, true, strings.Contains(err.Error(), "no authorized"))

	ping, err := P2pComm.NewPingData(p2pModule.node.nodeInfo)
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

func TestP2pComm(t *testing.T) {

	addrs := P2pComm.AddrRouteble([]string{"localhost:53802"}, calcChannelVersion(testChannel))
	t.Log(addrs)

	i32 := P2pComm.BytesToInt32([]byte{0xff})
	t.Log(i32)

	_, _, err := P2pComm.GenPrivPubkey()
	assert.Nil(t, err)

	ping, err := P2pComm.NewPingData(p2pModule.node.nodeInfo)
	assert.Nil(t, err)

	assert.Equal(t, true, P2pComm.CheckSign(ping))
	assert.IsType(t, "string", P2pComm.GetLocalAddr())
	assert.Equal(t, 5, len(P2pComm.RandStr(5)))

}

func TestFilter(t *testing.T) {
	filter := NewFilter(10)
	go filter.ManageRecvFilter()
	defer filter.Close()
	filter.GetLock()

	assert.Equal(t, true, filter.RegRecvData("key"))
	assert.Equal(t, true, filter.QueryRecvData("key"))
	filter.RemoveRecvData("key")
	assert.Equal(t, false, filter.QueryRecvData("key"))
	filter.ReleaseLock()

}

func TestAddrRouteble(t *testing.T) {
	resp := P2pComm.AddrRouteble([]string{"114.55.101.159:13802"}, calcChannelVersion(testChannel))
	t.Log(resp)
}

func TestRandStr(t *testing.T) {
	t.Log(P2pComm.RandStr(5))
}

func TestGetLocalAddr(t *testing.T) {
	t.Log(P2pComm.GetLocalAddr())
}

func TestAddrBook(t *testing.T) {

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

	addrBook := p2pModule.node.nodeInfo.addrBook
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

func TestBytesToInt32(t *testing.T) {

	t.Log(P2pComm.BytesToInt32([]byte{0xff}))
	t.Log(P2pComm.Int32ToBytes(255))
}

func TestNetAddress(t *testing.T) {
	tcpAddr := new(net.TCPAddr)
	tcpAddr.IP = net.ParseIP("localhost")
	tcpAddr.Port = 2223
	nad := NewNetAddress(tcpAddr)
	nad1 := nad.Copy()
	nad.Equals(nad1)
	nad2s, err := NewNetAddressStrings([]string{"localhost:3306"})
	if err != nil {
		return
	}
	nad.Less(nad2s[0])

}

func TestP2pListen(t *testing.T) {
	var node Node
	node.listenPort = 3333
	listen1 := NewListener("tcp", &node)
	assert.Equal(t, true, listen1 != nil)
	listen2 := NewListener("tcp", &node)
	assert.Equal(t, true, listen2 != nil)

	listen1.Close()
	listen2.Close()
}

func TestP2pRestart(t *testing.T) {

	assert.Equal(t, false, p2pModule.isClose())
	assert.Equal(t, false, p2pModule.isRestart())

	for i := 0; i < 100; i++ {
		client := p2pModule.client
		msg := client.NewMessage("p2p", types.EventTxBroadcast, &types.Transaction{})
		client.Send(msg, false)
	}

	p2pModule.ReStart()
}

type versionData struct {
	rawData interface{}
	version int32
}

func Test_processP2P(t *testing.T) {

	node := p2pModule.node
	pid := "testPid"
	sendChan := make(chan interface{}, 1)
	recvChan := make(chan *types.BroadCastData, 1)
	testDone := make(chan struct{})

	payload := []byte("testpayload")
	minerTx := &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 14600, Expire: 200}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 4600, Expire: 2}
	tx1 := &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 460000000, Expire: 0}
	tx2 := &types.Transaction{Execer: []byte("coins"), Payload: payload, Fee: 100, Expire: 1}
	txGroup, _ := types.CreateTxGroup([]*types.Transaction{tx1, tx2})
	gtx := txGroup.Tx()
	txList := append([]*types.Transaction{}, minerTx, tx, tx1, tx2)
	memTxList = append([]*types.Transaction{}, tx, gtx)

	block := &types.Block{
		TxHash: []byte("123"),
		Height: 10,
		Txs:    txList,
	}
	txHash := hex.EncodeToString(tx.Hash())
	blockHash := hex.EncodeToString(block.Hash())
	rootHash := merkle.CalcMerkleRoot(txList)

	//测试发送
	go func() {
		for data := range sendChan {
			verData, ok := data.(*versionData)
			assert.True(t, ok)
			sendData, doSend := node.processSendP2P(verData.rawData, verData.version)
			txHashFilter.regRData.Remove(txHash)
			blockHashFilter.regRData.Remove(blockHash)
			assert.True(t, doSend)
			recvChan <- sendData
		}
	}()
	//测试接收
	go func() {
		for data := range recvChan {
			txHashFilter.regRData.Remove(txHash)
			blockHashFilter.regRData.Remove(blockHash)
			handled := node.processRecvP2P(data, pid, node.pubToPeer)
			assert.True(t, handled)
		}
	}()

	//data test
	go func() {
		//normal
		sendChan <- &versionData{rawData: &types.P2PTx{Tx: tx}, version: lightBroadCastVersion - 1}
		sendChan <- &versionData{rawData: &types.P2PBlock{Block: block}, version: lightBroadCastVersion - 1}
		//light broadcast
		node.nodeInfo.cfg.StartLightTxTTL = 0
		sendChan <- &versionData{rawData: &types.P2PTx{Tx: tx}, version: lightBroadCastVersion}
		sendChan <- &versionData{rawData: &types.P2PBlock{Block: block}, version: lightBroadCastVersion}

		for !ltBlockCache.contains(blockHash) {
		}
		ltBlock := ltBlockCache.get(blockHash).(*types.Block)
		assert.True(t, bytes.Equal(rootHash, merkle.CalcMerkleRoot(ltBlock.Txs)))

		//query
		subChan := node.pubsub.Sub(pid)
		sendChan <- &versionData{rawData: &types.P2PQueryData{Value: &types.P2PQueryData_TxReq{TxReq: &types.P2PTxReq{TxHash: tx.Hash()}}}}
		_, ok := (<-subChan).(*types.P2PTx)
		assert.True(t, ok)
		sendChan <- &versionData{rawData: &types.P2PQueryData{Value: &types.P2PQueryData_BlockTxReq{BlockTxReq: &types.P2PBlockTxReq{
			BlockHash: blockHash,
			TxIndices: []int32{1, 2},
		}}}}
		rep, ok := (<-subChan).(*types.P2PBlockTxReply)
		assert.True(t, ok)
		assert.Equal(t, 2, int(rep.TxIndices[1]))
		sendChan <- &versionData{rawData: &types.P2PQueryData{Value: &types.P2PQueryData_BlockTxReq{BlockTxReq: &types.P2PBlockTxReq{
			BlockHash: blockHash,
			TxIndices: nil,
		}}}}
		rep, ok = (<-subChan).(*types.P2PBlockTxReply)
		assert.True(t, ok)
		assert.Nil(t, rep.TxIndices)

		//query reply
		sendChan <- &versionData{rawData: &types.P2PBlockTxReply{
			BlockHash: blockHash,
			TxIndices: []int32{1},
			Txs:       txList[1:2],
		}}
		<-subChan
		assert.True(t, ltBlockCache.contains(blockHash))
		assert.Nil(t, ltBlock.Txs)

		ltBlock.TxHash = rootHash
		sendChan <- &versionData{rawData: &types.P2PBlockTxReply{
			BlockHash: blockHash,
			Txs:       txList[0:],
		}}
		for ltBlockCache.contains(blockHash) {
		}
		//max ttl
		node.nodeInfo.cfg.MaxTTL = 0
		_, doSend := node.processSendP2P(&types.P2PTx{Tx: tx}, lightBroadCastVersion)
		assert.True(t, doSend)
		close(testDone)
	}()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-testDone:
			return
		case <-ticker.C:
			t.Error("TestP2PProcessTimeout")
			return
		}
	}
}

func TestP2pClose(t *testing.T) {
	p2pModule.Close()
	os.RemoveAll(dataDir)
}
