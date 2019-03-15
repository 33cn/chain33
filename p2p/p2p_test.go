package p2p

import (
	"os"
	"strings"
	"testing"
	"time"

	l "github.com/33cn/chain33/common/log"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	//"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var q queue.Queue
var p2pModule *P2p
var dataDir = "testdata"

func init() {

	l.SetLogLevel("err")
	q = queue.New("channel")
	go q.Start()

	p2pModule = initP2p(33802, dataDir)

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

//初始化p2p模块
func initP2p(port int32, dbpath string) *P2p {
	cfg := new(types.P2P)
	cfg.Port = port
	cfg.Enable = true
	cfg.DbPath = dbpath
	cfg.DbCache = 4
	cfg.Version = 216
	cfg.ServerStart = true
	cfg.Driver = "leveldb"

	p2pcli := New(cfg)
	p2pcli.SetQueueClient(q.Client())
	return p2pcli
}

//测试grpc 多连接
func TestGrpcConns(t *testing.T) {
	var conns []*grpc.ClientConn

	for i := 0; i < maxSamIPNum; i++ {
		conn, err := grpc.Dial("localhost:33802", grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
		assert.Nil(t, err)

		cli := types.NewP2PgserviceClient(conn)
		_, err = cli.GetHeaders(context.Background(), &types.P2PGetHeaders{
			StartHeight: 0, EndHeight: 0, Version: 1002}, grpc.FailFast(true))
		assert.Equal(t, false, strings.Contains(err.Error(), "no authorized"))
		conns = append(conns, conn)
	}

	conn, err := grpc.Dial("localhost:33802", grpc.WithInsecure(),
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

	conn, err := grpc.Dial("localhost:33802", grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	assert.Nil(t, err)
	cli := types.NewP2PgserviceClient(conn)
	var p2pdata types.P2PGetData
	resp, err := cli.GetData(context.Background(), &p2pdata)
	assert.Nil(t, err)
	_, err = resp.Recv()
	assert.Equal(t, true, strings.Contains(err.Error(), "no authorized"))
	conn.Close()

}

//测试Peer
func TestPeer(t *testing.T) {
	p2pModule.Close()
	p2pModule = initP2p(33802, dataDir)
	conn, err := grpc.Dial("localhost:33802", grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	assert.Nil(t, err)
	defer conn.Close()

	remote, err := NewNetAddressString("127.0.0.1:33802")
	assert.Nil(t, err)

	localP2P := initP2p(43802, "testdata2")
	defer os.RemoveAll("testdata2")
	defer localP2P.Close()

	peer, err := P2pComm.dialPeer(remote, localP2P.node)
	assert.Nil(t, err)
	defer peer.Close()
	peer.MakePersistent()
	peer.Start()
	time.Sleep(time.Second * 5)
	t.Log(peer.GetInBouns())
	assert.IsType(t, "string", peer.GetPeerName())
	p2pcli := NewNormalP2PCli()

	//测试发送Ping消息
	err = p2pcli.SendPing(peer, localP2P.node.nodeInfo)
	assert.Nil(t, err)

	//获取peer节点的被连接数
	pnum, err := p2pcli.GetInPeersNum(peer)
	assert.Nil(t, err)
	assert.Equal(t, 1, pnum)

	_, err = peer.GetPeerInfo(216)
	assert.Nil(t, err)
	//获取节点列表
	_, err = p2pcli.GetAddrList(peer)
	assert.Nil(t, err)

	_, err = p2pcli.SendVersion(peer, localP2P.node.nodeInfo)
	assert.Nil(t, err)

	t.Log(p2pcli.CheckPeerNatOk("localhost:33802"))

	_, err = p2pcli.GetAddr(peer)
	assert.Nil(t, err)

	//	//测试获取高度
	height, err := p2pcli.GetBlockHeight(localP2P.node.nodeInfo)
	assert.Nil(t, err)
	assert.Equal(t, int(height), 2019)
	assert.Equal(t, false, p2pcli.CheckSelf("localhost:33802", localP2P.node.nodeInfo))
	//测试下载
	job := NewDownloadJob(NewP2PCli(localP2P).(*Cli), []*Peer{peer})

	freePeer := job.GetFreePeer(1)
	t.Log(freePeer)
	var ins []*types.Inventory
	var bChan = make(chan *types.BlockPid, 256)
	respIns := job.DownloadBlock(ins, bChan)
	t.Log(respIns)
	os.Remove(dataDir)
}

//func TestP2PEvent(t *testing.T) {

//	qcli := q.Client()
//	msg := qcli.NewMessage("p2p", types.EventBlockBroadcast, &types.Block{})
//	err := qcli.Send(msg, false)
//	assert.Nil(t, err)
//	msg = qcli.NewMessage("p2p", types.EventTxBroadcast, &types.Transaction{})
//	err = qcli.Send(msg, false)
//	assert.Nil(t, err)

//	msg = qcli.NewMessage("p2p", types.EventFetchBlocks, &types.ReqBlocks{})
//	err = qcli.Send(msg, false)
//	assert.Nil(t, err)

//	msg = qcli.NewMessage("p2p", types.EventGetMempool, nil)
//	err = qcli.Send(msg, false)
//	assert.Nil(t, err)

//	msg = qcli.NewMessage("p2p", types.EventPeerInfo, nil)
//	err = qcli.Send(msg, false)
//	assert.Nil(t, err)
//	msg = qcli.NewMessage("p2p", types.EventGetNetInfo, nil)
//	err = qcli.Send(msg, false)
//	assert.Nil(t, err)

//	msg = qcli.NewMessage("p2p", types.EventFetchBlockHeaders, &types.ReqBlocks{})
//	err = qcli.Send(msg, false)
//	assert.Nil(t, err)

//}

func TestP2pComm(t *testing.T) {

	addrs := P2pComm.AddrRouteble([]string{"localhost:33802"})
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
	go Filter.ManageRecvFilter()
	defer Filter.Close()
	assert.Equal(t, true, Filter.RegRecvData("key"))
	assert.Equal(t, true, Filter.QueryRecvData("key"))
	Filter.RemoveRecvData("key")
	assert.Equal(t, false, Filter.QueryRecvData("key"))

}

func TestP2pClose(t *testing.T) {
	p2pModule.Close()
	os.RemoveAll(dataDir)
}
