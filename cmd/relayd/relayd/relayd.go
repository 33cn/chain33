package relayd

import (
	"context"
	"encoding/hex"
	"math/rand"
	"os"
	"sync"
	"time"

	"fmt"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/inconshreveable/log15"
	"github.com/valyala/fasthttp"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
)

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
	config          *Config
	httpServer      fasthttp.Server
	db              *relaydDB
	mu              sync.Mutex
	latestBlockHash *chainhash.Hash
	latestHeight    uint64
	knownBlockHash  *chainhash.Hash
	knownHeight     uint64
	btcClient       BtcClient
	btcClientLock   sync.Mutex
	client33        *Client33
	ctx             context.Context
	cancel          context.CancelFunc
	privateKey      crypto.PrivKey
	publicKey       crypto.PubKey
}

func NewRelayd(config *Config) *Relayd {
	ctx, cancel := context.WithCancel(context.Background())
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	db := NewRelayDB("relayd", dir, 256)
	currentHeight, err := db.Get(currentBlockheightKey[:])
	// if err != nil || currentHeight == nil {
	// 	currentHeight = 0
	// }
	// value, err := chainhash.NewHash(hash)
	// TODO
	height, err := strconv.Atoi(string(currentHeight))
	if err != nil {
		log.Warn(fmt.Sprintf("NewRelayd %s", err.Error()))
		height = 0
	}
	log.Info("NewRelayd", "current btc hegiht: ", height)

	client33 := NewClient33(&config.Chain33)

	var btc BtcClient
	if config.BtcdOrWeb == 0 {
		btc, err = NewBtcd(config.Btcd.BitConnConfig(), int(config.Btcd.ReconnectAttempts))
	} else {
		btc, err = NewBtcWeb()
	}
	if err != nil {
		panic(err)
	}

	pk, err := hex.DecodeString(config.Auth.PrivateKey)
	if err != nil {
		panic(err)
	}

	secp, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}

	priKey, err := secp.PrivKeyFromBytes(pk)
	if err != nil {
		panic(err)
	}

	return &Relayd{
		config:      config,
		db:          db,
		ctx:         ctx,
		cancel:      cancel,
		client33:    client33,
		knownHeight: uint64(height),
		btcClient:   btc,
		privateKey:  priKey,
		publicKey:   priKey.PubKey(),
	}
}

func (r *Relayd) Close() {
	r.cancel()
	r.client33.Close()
}

func (r *Relayd) Start() {
	r.btcClient.Start()
	r.client33.Start(r.ctx)
	r.tick(r.ctx)
	if err := fasthttp.ListenAndServe("0.0.0.0:8080", requestHandler); err != nil {
		panic(fmt.Errorf("Error in ListenAndServe: %s", err))
	}
}

func (r *Relayd) tick(ctx context.Context) {
	tickDealTx := time.Tick(time.Second * time.Duration(r.config.Tick33))
	tickQuery := time.Tick(time.Second * time.Duration(r.config.Tick33))
	tickBtcd := time.Tick(time.Second * time.Duration(r.config.TickBTC))
	ping := time.Tick(time.Second * 2 * time.Duration(r.config.TickBTC))

out:
	for {
		select {
		case <-ctx.Done():
			break out

		case <-tickDealTx:
			log.Info("deal transaction order")
			r.dealOrder()

		case <-tickQuery:
			log.Info("check transaction order")
			// 定时查询交易是否成功

		case <-tickBtcd:
			// btc ticket
			log.Info("sync SPV")
			hash, height, err := r.btcClient.GetLatestBlock()
			if err == nil {
				r.latestBlockHash = hash
				r.latestHeight = uint64(height)
			} else {
				log.Error("tick GetBestBlock", "error: ", err)
			}
			if r.latestHeight > r.knownHeight {
				r.syncBlockHeaders()
			}

		case <-ping:
			r.btcClient.Ping()
		}
	}
}

func (r *Relayd) syncBlockHeaders() {
	// TODO
	log.Info("syncBlockHeaders", "current btc block height: ", r.knownHeight, "current btc block height: ", r.latestHeight)
	if r.knownHeight < r.config.MinHeightBTC || r.latestHeight < r.config.MinHeightBTC {
		return
	}
	totalSetup := r.latestHeight - r.knownHeight
	stage := totalSetup / r.config.SyncSetup
	little := totalSetup % r.config.SyncSetup
	var i uint64
out:
	for i = 0; i <= stage; i++ {
		log.Info("syncing BlockHeaders", "current btc block height: ", r.knownHeight, "current btc block height: ", r.latestHeight)
		var add uint64
		if i == stage {
			add = little
		} else {
			add = r.config.SyncSetup
		}
		// log.Info("syncBlockHeaders-----------------------------------1")
		headers := make([]*types.BtcHeader, 0, add)
		breakHeight := add + r.knownHeight
		for j := r.knownHeight + 1; j <= breakHeight; j++ {
			header, err := r.btcClient.GetBlockHeader(j)
			if err != nil {
				log.Error("syncBlockHeaders", "GetBlockHeader error", err)
				break out
			}
			data := types.Encode(header)
			// TODO save db
			r.db.Set(makeHeightKey(r.knownHeight), data)
			r.db.Set(currentBlockheightKey, []byte(fmt.Sprintf("%d", r.knownHeight)))
			// log.Info("syncBlockHeaders-----------------------------------2")
			r.knownHeight++
			headers = append(headers, header)
		}
		// TODO
		log.Info("syncBlockHeaders", "len: ", len(headers))
		btcHeaders := &types.BtcHeaders{BtcHeader: headers}
		relayHeaders := &types.RelayAction_BtcHeaders{btcHeaders}
		action := &types.RelayAction{
			Value: relayHeaders,
			Ty:    types.RelayActionRcvBTCHeaders,
		}
		// log.Info("syncBlockHeaders-----------------------------------3")
		tx := r.transaction(types.Encode(action))
		ret, err := r.client33.SendTransaction(r.ctx, tx)
		if err != nil {
			// panic(err)
			log.Error("syncBlockHeaders", "SendTransaction error", err)
			break out
		}
		log.Info("syncBlockHeaders end SendTransaction", "IsOk: ", ret.GetIsOk(), "msg: ", string(ret.GetMsg()))
	}
}

func (r *Relayd) transaction(payload []byte) *types.Transaction {
	tx := &types.Transaction{
		Execer:  executor,
		Payload: payload,
		Nonce:   rand.Int63(),
		Fee:     r.config.Fee,
	}
	tx.Sign(types.SECP256K1, r.privateKey)
	return tx
}

func (r *Relayd) dealOrder() {
	result, err := r.requestRelayOrders(types.RelayOrderStatus_confirming)
	if err != nil {
		log.Error("dealOrder", "requestRelayOrders error: ", err)
	}

	// txSpv := make([][]byte, len(result.GetOrders()))
	for _, value := range result.GetOrders() {
		// TODO save db ???
		tx, err := r.btcClient.GetTransaction(value.Exchgtxhash)
		if err != nil {
			log.Error("dealOrder", "dealOrder GetTransaction error: ", err)
			continue
		}

		spv, err := r.btcClient.GetSPV(tx.BlockHeight, tx.Hash)
		if err != nil {
			log.Error("dealOrder", "GetSPV error: ", err)
			continue
		}
		// var data []byte
		// err = types.Decode(data, tx)
		// if err != nil {
		// 	log.Error("dealOrder json.Marshal", err)
		// }
		verify := &types.RelayVerify{
			Orderid: value.Orderid,
			Tx:      tx,
			Spv:     spv,
		}
		rr := &types.RelayAction_Rverify{
			verify,
		}
		action := &types.RelayAction{
			Value: rr,
			Ty:    types.RelayActionVerifyTx,
		}
		t := r.transaction(types.Encode(action))
		r.client33.SendTransaction(r.ctx, t, nil)
		// TODO save db ???
		// txSpv = append(txSpv, data)
	}
	// TODO
	// r.client33.SendTransaction(r.ctx, r.transaction(txs))
	// t := r.transaction(types.Encode(action))
}

func (r *Relayd) requestRelayOrders(status types.RelayOrderStatus) (*types.QueryRelayOrderResult, error) {
	payLoad := types.Encode(&types.ReqRelayAddrCoins{
		Status: status,
	})
	query := types.Query{
		Execer:   executor,
		FuncName: "GetRelayOrderByStatus",
		Payload:  payLoad,
	}
	ret, err := r.client33.QueryChain(r.ctx, &query)
	if err != nil {
		return nil, err

	}
	if !ret.GetIsOk() {
		log.Info("requestRelayOrders", "error")
	}
	var result types.QueryRelayOrderResult
	types.Decode(ret.Msg, &result)
	return &result, nil
}

// triggerOrderStatus Trigger order status for BlockChain
func (r *Relayd) triggerOrderStatus() {
	panic("unimplemented")
}

func (r *Relayd) isExist(id string) {
	panic("unimplemented")
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "Hello, world!\n\n")

	fmt.Fprintf(ctx, "Request method is %q\n", ctx.Method())
	fmt.Fprintf(ctx, "RequestURI is %q\n", ctx.RequestURI())
	fmt.Fprintf(ctx, "Requested path is %q\n", ctx.Path())
	fmt.Fprintf(ctx, "Host is %q\n", ctx.Host())
	fmt.Fprintf(ctx, "Query string is %q\n", ctx.QueryArgs())
	fmt.Fprintf(ctx, "User-Agent is %q\n", ctx.UserAgent())
	fmt.Fprintf(ctx, "Connection has been established at %s\n", ctx.ConnTime())
	fmt.Fprintf(ctx, "Request has been started at %s\n", ctx.Time())
	fmt.Fprintf(ctx, "Serial request number for the current connection is %d\n", ctx.ConnRequestNum())
	fmt.Fprintf(ctx, "Your ip is %q\n\n", ctx.RemoteIP())

	fmt.Fprintf(ctx, "Raw request is:\n---CUT---\n%s\n---CUT---", &ctx.Request)

	ctx.SetContentType("text/plain; charset=utf8")

	// Set arbitrary headers
	ctx.Response.Header.Set("X-My-Header", "my-header-value")

	// Set cookies
	var c fasthttp.Cookie
	c.SetKey("cookie-name")
	c.SetValue("cookie-value")
	ctx.Response.Header.SetCookie(&c)
}
