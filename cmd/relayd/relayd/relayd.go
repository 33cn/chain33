package relayd

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/inconshreveable/log15"
	"github.com/valyala/fasthttp"
	"gitlab.33.cn/chain33/chain33/account"
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
	config           *Config
	db               *relaydDB
	mu               sync.Mutex
	latestBlockHash  *chainhash.Hash
	latestBtcHeight  uint64
	knownBlockHash   *chainhash.Hash
	knownBtcHeight   uint64
	btcClient        BtcClient
	btcClientLock    sync.Mutex
	client33         *Client33
	ctx              context.Context
	cancel           context.CancelFunc
	privateKey       crypto.PrivKey
	publicKey        crypto.PubKey
	isPersist        int32
	isSnyc           int32
	isResetBtcHeight bool
	quitPersist      chan struct{}
}

func NewRelayd(config *Config) *Relayd {
	ctx, cancel := context.WithCancel(context.Background())
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	db := NewRelayDB("relayd", dir, 256)
	currentHeight, err := db.Get(currentBtcBlockheightKey[:])
	h, err := strconv.Atoi(string(currentHeight))
	if err != nil {
		log.Warn("NewRelayd", "atoi height error: ", err)
	}
	var isResetBtcHeight bool
	height := uint64(h)
	if height < config.FirstBtcHeight {
		height = config.FirstBtcHeight
		isResetBtcHeight = true
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

	pubkey := priKey.PubKey()
	fmt.Println(pubkey.KeyString())
	address := account.PubKeyToAddress(pubkey.Bytes())
	fmt.Println(address.String())

	return &Relayd{
		config:           config,
		db:               db,
		ctx:              ctx,
		cancel:           cancel,
		client33:         client33,
		knownBtcHeight:   uint64(height),
		btcClient:        btc,
		privateKey:       priKey,
		publicKey:        pubkey,
		isPersist:        0,
		isSnyc:           0,
		isResetBtcHeight: isResetBtcHeight,
		quitPersist:      make(chan struct{}),
	}
}

func (r *Relayd) Close() {
	close(r.quitPersist)
	r.client33.Close()
	r.cancel()
}

func (r *Relayd) Start() {
	r.btcClient.Start()
	r.client33.Start(r.ctx)
	r.tick(r.ctx)
	if err := fasthttp.ListenAndServe("0.0.0.0:8080", requestHandler); err != nil {
		panic(fmt.Errorf("ListenAndServe error: ", err))
	}
}

func (r *Relayd) tick(ctx context.Context) {
	tickDealTx := time.Tick(time.Second * time.Duration(r.config.Tick33))
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

		case <-tickBtcd:
			// btc ticket
			log.Info("snyc btc information")
			hash, height, err := r.btcClient.GetLatestBlock()
			if err != nil {
				log.Error("tick GetBestBlock", "error ", err)
			} else {
				r.latestBlockHash = hash
				atomic.StoreUint64(&r.latestBtcHeight, uint64(height))
			}

			if atomic.LoadUint64(&r.latestBtcHeight) > atomic.LoadUint64(&r.knownBtcHeight) {
				if atomic.LoadInt32(&r.isPersist) == 0 {
					go r.persistBlockHeaders()
				}

				if atomic.LoadInt32(&r.isSnyc) == 0 {
					go r.syncBlockHeaders()
				}
			}

		case <-ping:
			r.btcClient.Ping()
		}
	}
}

func (r *Relayd) persistBlockHeaders() {
	atomic.StoreInt32(&r.isPersist, 1)
	defer atomic.StoreInt32(&r.isPersist, 0)

	if atomic.LoadUint64(&r.knownBtcHeight) < atomic.LoadUint64(&r.latestBtcHeight) {
	out:
		for i := atomic.LoadUint64(&r.knownBtcHeight); i < atomic.LoadUint64(&r.latestBtcHeight); i++ {
			select {
			case <-r.quitPersist:
				break out
			default:
				header, err := r.btcClient.GetBlockHeader(i)
				if err != nil {
					log.Error("syncBlockHeaders", "GetBlockHeader error", err)
					break out
				}
				data := types.Encode(header)
				r.db.Set(makeHeightKey(i), data)
				r.db.Set(currentBtcBlockheightKey, []byte(fmt.Sprintf("%d", i)))
				atomic.StoreUint64(&r.knownBtcHeight, i)
				if i%10 == 0 {
					log.Info("persistBlockHeaders", "current knownBtcHeight: ", i, "current latestBtcHeight: ", atomic.LoadUint64(&r.latestBtcHeight))
				}
			}
		}
	}
}

func (r *Relayd) queryChain33WithBtcHeight() (*types.ReplayRelayQryBTCHeadHeight, error) {
	payLoad := types.Encode(&types.ReqRelayQryBTCHeadHeight{})
	query := types.Query{
		Execer:   executor,
		FuncName: "GetBTCHeaderCurHeight",
		Payload:  payLoad,
	}
	ret, err := r.client33.QueryChain(r.ctx, &query)
	if err != nil {
		return nil, err
	}
	if !ret.GetIsOk() {
		log.Info("GetBTCHeaderCurHeight", "error", ret.GetMsg())
	}
	var result types.ReplayRelayQryBTCHeadHeight
	types.Decode(ret.Msg, &result)
	return &result, nil
}

func (r *Relayd) syncBlockHeaders() {
	atomic.StoreInt32(&r.isSnyc, 1)
	defer atomic.StoreInt32(&r.isSnyc, 0)

	knownBtcHeight := atomic.LoadUint64(&r.knownBtcHeight)
	if knownBtcHeight > r.config.FirstBtcHeight {
		ret, err := r.queryChain33WithBtcHeight()
		if err != nil {
			log.Error("syncBlockHeaders", "queryChain33WithBtcHeight error: ", err)
			return
		}

		log.Info("syncBlockHeaders", "queryChain33WithBtcHeight result: ", ret)
		var initIterHeight uint64
		if (ret.CurHeight <= 0 || r.config.FirstBtcHeight != uint64(ret.BaseHeight)) && !r.isResetBtcHeight {
			r.isResetBtcHeight = true
			r.quitPersist <- struct{}{}
			atomic.StoreUint64(&r.knownBtcHeight, r.config.FirstBtcHeight)
			return
		}

		if r.isResetBtcHeight {
			initIterHeight = r.config.FirstBtcHeight
		} else {
			initIterHeight = uint64(ret.CurHeight) + 1
			if initIterHeight > r.knownBtcHeight {
				return
			}
		}

		total := knownBtcHeight - initIterHeight
		totalConfig := r.config.SyncSetupCount * r.config.SyncSetup

		if total > totalConfig {
			total = totalConfig
		}

		stage := total / r.config.SyncSetup
		little := total % r.config.SyncSetup
		var i uint64
	out:
		for i = 0; i <= stage; i++ {
			var add uint64
			if i == stage {
				if little <= 0 {
					break out
				}
				add = little
			} else {
				add = r.config.SyncSetup
			}
			headers := make([]*types.BtcHeader, 0, add)
			breakHeight := add + initIterHeight
			for j := initIterHeight; j < breakHeight; j++ {
				// TODO betach request headers
				header, err := r.db.BlockHeader(j)
				if err != nil {
					log.Error("syncBlockHeaders", "GetBlockHeader error", err)
					break out
				}

				header.IsReset = r.isResetBtcHeight
				r.isResetBtcHeight = false
				headers = append(headers, header)
			}
			initIterHeight = breakHeight
			log.Info("syncBlockHeaders", "len: ", len(headers))
			btcHeaders := &types.BtcHeaders{BtcHeader: headers}
			relayHeaders := &types.RelayAction_BtcHeaders{btcHeaders}
			action := &types.RelayAction{
				Value: relayHeaders,
				Ty:    types.RelayActionRcvBTCHeaders,
			}
			tx := r.transaction(types.Encode(action))
			ret, err := r.client33.SendTransaction(r.ctx, tx)
			if err != nil {
				log.Error("syncBlockHeaders", "SendTransaction error", err)
				break out
			}
			log.Info("syncBlockHeaders end SendTransaction", "IsOk: ", ret.GetIsOk(), "msg: ", string(ret.GetMsg()))
		}
	}
}

func (r *Relayd) transaction(payload []byte) *types.Transaction {
	tx := &types.Transaction{
		Execer:  executor,
		Payload: payload,
		Nonce:   rand.Int63(),
		To:      account.ExecAddress(string(executor)),
	}

	fee, _ := tx.GetRealFee(types.MinFee)
	tx.Fee = fee
	tx.Sign(types.SECP256K1, r.privateKey)
	log.Info("transaction", "fee : ", fee)
	return tx
}

func (r *Relayd) dealOrder() {
	result, err := r.requestRelayOrders(types.RelayOrderStatus_confirming)
	if err != nil {
		log.Error("dealOrder", "requestRelayOrders error: ", err)
	}

	for _, value := range result.GetOrders() {
		// TODO save db ???
		tx, err := r.btcClient.GetTransaction(value.CoinTxHash)
		if err != nil {
			log.Error("dealOrder", "dealOrder GetTransaction error: ", err)
			continue
		}
		spv, err := r.btcClient.GetSPV(tx.BlockHeight, tx.Hash)
		if err != nil {
			log.Error("dealOrder", "GetSPV error: ", err)
			continue
		}
		verify := &types.RelayVerify{
			OrderId: value.Id,
			Tx:      tx,
			Spv:     spv,
		}
		rr := &types.RelayAction_Verify{
			verify,
		}
		action := &types.RelayAction{
			Value: rr,
			Ty:    types.RelayActionVerifyTx,
		}
		t := r.transaction(types.Encode(action))
		ret, err := r.client33.SendTransaction(r.ctx, t)
		if err != nil {
			log.Error("syncBlockHeaders", "SendTransaction error", err)
			break
		}
		log.Info("syncBlockHeaders end SendTransaction", "IsOk: ", ret.GetIsOk(), "msg: ", string(ret.GetMsg()))
	}
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
