package game

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"

	"bytes"
	"context"
	"sync"
	"time"
)

var (
	clog            = log.New("module", "txCache")
	grpcRecSize int = 30 * 1024 * 1024 //the size should be limited in server

	client *CacheClient
)

const (
	Pending = int32(5)
)

type CacheClient struct {
	conn         *grpc.ClientConn
	grpcClient   types.GrpcserviceClient
	CacheGameIds []string
	sync.Mutex
	wg sync.WaitGroup
}

func NewClient() *CacheClient {
	clog.Debug("New Cache client")

	msgRecvOp := grpc.WithMaxMsgSize(grpcRecSize)
	conn, err := grpc.Dial(types.GetParaRemoteGrpcClient(), grpc.WithInsecure(), msgRecvOp)

	if err != nil {
		panic(err)
	}
	grpcClient := types.NewGrpcserviceClient(conn)
	client := CacheClient{
		conn,
		grpcClient,
		[]string{},
		sync.Mutex{},
		sync.WaitGroup{},
	}
	return &client
}
func Start() {
	if types.IsPara() {
		client = NewClient()
		client.wg.Add(1)
		go func() {
			tick := time.Tick(time.Second)
			for {
				<-tick
				clog.Debug(" =============Update Cache GameId List============")
				err := client.UpdateGameList()
				if err != nil {
					clog.Error("******Update Cache have err:", err.Error())
				}
			}
			client.wg.Done()
		}()
		client.wg.Wait()
	}
}
func (client *CacheClient) UpdateGameList() error {
	client.Lock()
	defer client.Unlock()
	replyTxList, err := client.grpcClient.GetLastMemPool(context.Background(), &types.ReqNil{})
	if err != nil {
		clog.Error("GetLastMemPool have err:", err.Error())
		return err
	}
	var matchGameIds []string
	for _, tx := range replyTxList.GetTxs() {
		if bytes.Equal(tx.Execer, []byte(types.GetParaName()+types.GameX)) {
			var action types.GameAction
			err := types.Decode(tx.Payload, &action)
			if err != nil {
				continue
			}
			clog.Debug("exec Game tx=", "tx=", action)
			if action.GetTy() == types.GameActionMatch && action.GetMatch() != nil {
				matchGameIds = append(matchGameIds, action.GetMatch().GetGameId())
			}
		}
	}
	client.CacheGameIds = matchGameIds
	return nil
}
func (client *CacheClient) filterStatus(game *types.Game) int32 {
	for _, id := range client.CacheGameIds {
		if game.GetGameId() == id {
			return Pending
		}
	}
	return game.GetStatus()
}
