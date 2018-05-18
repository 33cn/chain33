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
	log.Info("NewRelayd", "current hegiht: ", height)

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

	pr, err := hex.DecodeString(config.Auth.PrivateKey)
	if err != nil {
		panic(err)
	}

	secp, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}

	priKey, err := secp.PrivKeyFromBytes(pr)
	if err != nil {
		panic(err)
	}

	return &Relayd{
		config:      config,
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
			// r.dealOrder()

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
				log.Error("tick GetBestBlock", err)
			}
			if r.latestHeight > r.knownHeight {
				go r.syncBlockHeaders()
			}

		case <-ping:
			r.btcClient.Ping()
		}
	}
}

func (r *Relayd) syncBlockHeaders() {
	// TODO
	if r.knownHeight <= r.config.MinHeightBTC || r.latestHeight <= r.config.MinHeightBTC {
		return
	}
	totalSetup := r.latestHeight - r.knownHeight
	stage := totalSetup / SETUP
	little := totalSetup % SETUP
	var i uint64
out:
	for ; i <= stage; i++ {
		var add uint64
		if i == stage {
			add += little
		} else {
			add += SETUP
		}
		headers := make([]*types.BtcHeader, add)
		add += r.knownHeight
		for j := r.knownHeight; j <= add; j++ {
			header, err := r.btcClient.GetBlockHeader(j)
			if err != nil {
				break out
			}
			data := types.Encode(header)
			// save db
			err = r.db.Set(makeHeightKey(r.knownHeight), data)
			if err != nil {
				break out
			}
			r.knownHeight++
			headers = append(headers, header)
		}
		// TODO
		h := &types.BtcHeaders{BtcHeader: headers}
		hh := &types.RelayAction_BtcHeaders{h}
		action := &types.RelayAction{
			Value: hh,
			Ty:    8,
		}
		tx := r.transaction(types.Encode(action))
		ret, err := r.client33.SendTransaction(r.ctx, tx, nil)
		if err != nil {
			panic(err)
		}
		log.Info("syncBlockHeaders", ret)
	}
}

func (r *Relayd) transaction(payload []byte) *types.Transaction {
	tx := &types.Transaction{
		Execer:  executor,
		Payload: payload,
		Nonce:   rand.Int63(),
	}
	tx.Sign(types.SECP256K1, r.privateKey)
	return tx
}

func (r *Relayd) dealOrder() {
	result, err := r.requestRelayOrders(types.RelayOrderStatus_confirming)
	if err != nil {
		log.Info("dealOrder", err)
	}

	// txSpv := make([][]byte, len(result.GetOrders()))
	for _, value := range result.GetOrders() {
		// TODO save db ???
		tx, err := r.btcClient.GetTransaction(value.Exchgtxhash)
		if err != nil {
			log.Error("dealOrder", err)
			continue
		}

		spv, err := r.btcClient.GetSPV(tx.BlockHeight, tx.Hash)
		if err != nil {
			log.Error("spv error", err)
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
			Ty:    6,
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
	payLoad := types.Encode(&types.QueryRelayOrderParam{
		Status: status,
	})
	query := types.Query{
		Execer:   executor,
		FuncName: "queryRelayOrder",
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
