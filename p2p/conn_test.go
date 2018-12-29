package p2p

import (
	"os"
	"strings"
	"testing"

	l "github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/queue"
	pb "github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func initP2p() *P2p {
	l.SetLogLevel("err")
	cfg := new(pb.P2P)
	cfg.Port = 33802
	cfg.Enable = true
	cfg.DbPath = "testdata"
	cfg.DbCache = 10240
	cfg.Version = 216
	cfg.ServerStart = true
	cfg.Driver = "leveldb"
	q := queue.New("channel")
	p2pcli := New(cfg)
	p2pcli.SetQueueClient(q.Client())
	return p2pcli
}

func TestGrpcConns(t *testing.T) {
	p2pcli := initP2p()
	defer os.RemoveAll("testdata")
	defer p2pcli.Close()
	for i := 0; i < maxSamIPNum; i++ {
		conn, err := grpc.Dial("localhost:33802", grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
		assert.Nil(t, err)
		defer conn.Close()
		cli := pb.NewP2PgserviceClient(conn)
		_, err = cli.GetHeaders(context.Background(), &pb.P2PGetHeaders{
			StartHeight: 0, EndHeight: 0, Version: 1002}, grpc.FailFast(true))
		assert.Equal(t, false, strings.Contains(err.Error(), "not authorized"))
	}
	conn, err := grpc.Dial("localhost:33802", grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	assert.Nil(t, err)
	defer conn.Close()
	cli := pb.NewP2PgserviceClient(conn)
	_, err = cli.GetHeaders(context.Background(), &pb.P2PGetHeaders{
		StartHeight: 0, EndHeight: 0, Version: 1002}, grpc.FailFast(true))
	assert.Equal(t, true, strings.Contains(err.Error(), "not authorized"))
}

func TestGrpcStreamConns(t *testing.T) {
	p2pcli := initP2p()
	defer os.RemoveAll("testdata")
	defer p2pcli.Close()
	for i := 0; i < maxSamIPNum; i++ {
		conn, err := grpc.Dial("localhost:33802", grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
		assert.Nil(t, err)
		defer conn.Close()
		cli := pb.NewP2PgserviceClient(conn)
		var p2pdata pb.P2PGetData
		resp, err := cli.GetData(context.Background(), &p2pdata)
		assert.Nil(t, err)
		_, err = resp.Recv()
		assert.Equal(t, false, strings.Contains(err.Error(), "not authorized"))
	}
	conn, err := grpc.Dial("localhost:33802", grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")))
	assert.Nil(t, err)
	defer conn.Close()
	cli := pb.NewP2PgserviceClient(conn)
	var p2pdata pb.P2PGetData
	resp, err := cli.GetData(context.Background(), &p2pdata)
	assert.Nil(t, err)
	_, err = resp.Recv()
	assert.Equal(t, true, strings.Contains(err.Error(), "not authorized"))
}
