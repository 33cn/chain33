package relayd

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

// confirmed checks whether a transaction at height txHeight has met minconf
// confirmations for a blockchain at height curHeight.
func confirmed(minconf, txHeight, curHeight int32) bool {
	return confirms(txHeight, curHeight) >= minconf
}

func confirms(txHeight, curHeight int32) int32 {
	switch {
	case txHeight == -1, txHeight > curHeight:
		return 0
	default:
		return curHeight - txHeight + 1
	}
}

type Relayd struct {
	config     *Config
	httpServer http.Server
	db         *relaydDB
	mu         sync.Mutex

	latestBestBlock btcjson.GetBestBlockResult
	knownBlockHash  *chainhash.Hash

	btcClient     *BTCClient
	btcClientLock sync.Mutex
	client33      *Client33
	ctx           context.Context
	cancel        context.CancelFunc
}

var (
	currentBlockHashKey chainhash.Hash
	zeroBlockHeader     = []byte("")
)

const SETUP = 1000

func NewRelayd(config *Config) *Relayd {
	ctx, cancel := context.WithCancel(context.Background())
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	db := NewRelayDB("relayd", dir, 256)
	hash, err := db.Get(currentBlockHashKey[:])
	if err != nil || hash == nil {
		hash = zeroBlockHeader
	}
	value, err := chainhash.NewHash(hash)
	// TODO
	if err != nil {
		log.Info("NewRelayd", err)
		// panic(err)
	}
	client33 := NewClient33(&config.Chain33)
	btc, err := NewBTCClient(config.BitCoin.BitConnConfig(), int(config.ReconnectAttempts))

	return &Relayd{
		config:         config,
		ctx:            ctx,
		cancel:         cancel,
		client33:       client33,
		knownBlockHash: value,
		btcClient:      btc,
	}
}

func (r *Relayd) Start() {
	r.btcClient.Start()
	r.client33.Start(r.ctx)
}

func (r *Relayd) Heartbeat() {
	go func(ctx context.Context) {
	out:
		for {
			select {
			case <-ctx.Done():
				break out

			case <-time.After(time.Second * time.Duration(3)):
				err := r.client33.Ping(ctx)
				if err != nil {
					log.Info("Heartbeat chain33", err.Error())
					r.client33.AutoReconnect(ctx)
					r.config.ReconnectAttempts--
				}

				// TODO
				// err = r.btcClient.Ping()
				// if err != nil {
				// 	log.Info("Heartbeat btc", err)
				// }

				if r.config.ReconnectAttempts < 0 {
					break out
				}
			}
		}
	}(r.ctx)
}

func (r *Relayd) Loop() {
	// chain33 ticket
	go func(ctx context.Context) {
	out:
		for {
			select {
			case <-ctx.Done():
				break out

			case <-time.After(time.Second * time.Duration(r.config.Heartbeat33)):
				log.Info("deal transaction order")
				// r.dealOrder()

			case <-time.After(time.Second * 2 * time.Duration(r.config.Heartbeat33)):
				log.Info("check transaction order")
				// 定时查询交易是否成功

			}
		}

	}(r.ctx)

	// btc ticket
	go func(ctx context.Context) {
	out:
		for {
			select {
			case <-ctx.Done():
				break out

			case <-time.After(time.Second * time.Duration(r.config.HeartbeatBTC)):
				log.Info("sync SPV")
				// 同步区块头到chain33以及本地
				// r.btcClient.get

			}

		}
	}(r.ctx)

}

// Ping
func (r *Relayd) Ping() error {
	return nil
}

func (r *Relayd) Close() {
	r.cancel()
	r.client33.Close()
}

func (r *Relayd) SyncBlockHeaders() {
	// TODO
	// latestHash, latestHeight,err := r.btcClient.GetBestBlock()
	// if err != nil {
	// 	log.Info("SyncBlockHeaders", err)
	// }
	//
	// header, err := r.btcClient.GetBlockHeaderVerbose(latestHash)
	// if err != nil{
	// 	log.Info("SyncBlockHeaders", err)
	// }
	// r.btcClient.Getblo

	//  header, err := r.btcClient.GetBlockHeaderVerbose()
	// if err != nil {
	//
	//
	// }
}

func (r *Relayd) dealOrder() {
	result, err := r.queryRelayOrders(types.RelayOrderStatus_confirming)
	if err != nil {
		log.Info("dealOrder", err)
	}

	txs := make([][]byte, len(result.GetOrders()))
	for _, value := range result.GetOrders() {
		hash, err := chainhash.NewHashFromStr(value.Orderid)
		if err != nil {
			log.Error("dealOrder", err)
			continue
		}
		tx, err := r.btcClient.GetTransaction(hash)
		if err != nil {
			log.Error("dealOrder", err)
			continue
		}
		data, err := json.Marshal(tx)
		if err != nil {
			log.Error("dealOrder json.Marshal", err)
			continue
		}
		txs = append(txs, data)
	}

	// TODO
	// r.client33.SendTransaction()

}

var execer = []byte("relay")

func (r *Relayd) queryRelayOrders(status types.RelayOrderStatus) (*types.QueryRelayOrderResult, error) {
	payLoad := types.Encode(&types.QueryRelayOrderParam{
		Status: status,
	})
	query := types.Query{
		Execer:   execer,
		FuncName: "queryRelayOrder",
		Payload:  payLoad,
	}
	ret, err := r.client33.QueryChain(r.ctx, &query)
	if err != nil {
		// log.Info("queryRelayOrders", err)
		return nil, err

	}
	if !ret.GetIsOk() {
		log.Info("queryRelayOrders", "error")
	}
	var result types.QueryRelayOrderResult
	types.Decode(ret.Msg, &result)
	return &result, nil
}
