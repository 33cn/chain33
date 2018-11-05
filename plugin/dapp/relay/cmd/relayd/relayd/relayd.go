package relayd

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/types"
)

type Relayd struct {
	config            *Config
	db                *relaydDB
	latestBlockHash   *chainhash.Hash
	latestBtcHeight   uint64
	knownBtcHeight    uint64
	firstHeaderHeight uint64
	btcClient         BtcClient
	client33          *Client33
	ctx               context.Context
	cancel            context.CancelFunc
	privateKey        crypto.PrivKey
	publicKey         crypto.PubKey
	isPersisting      int32
	isSnycing         int32
	isResetBtcHeight  bool
}

func NewRelayd(config *Config) *Relayd {
	ctx, cancel := context.WithCancel(context.Background())
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	db := NewRelayDB("relayd", dir, 256)

	var firstHeight int
	var isResetBtcHeight bool
	data, err := db.Get(firstHeaderKey[:])
	if err != nil || data == nil {
		firstHeight = int(config.FirstBtcHeight)
		isResetBtcHeight = true
		db.Set(firstHeaderKey, []byte(fmt.Sprintf("%d", firstHeight)))
	}

	firstHeight, err = strconv.Atoi(string(data))
	if err != nil {
		log.Warn("NewRelayd", "atoi firstHeight error: ", err)
	}
	if firstHeight != int(config.FirstBtcHeight) {
		firstHeight = int(config.FirstBtcHeight)
		isResetBtcHeight = true
	}

	currentHeight, err := db.Get(currentBtcBlockheightKey[:])
	height, err := strconv.Atoi(string(currentHeight))
	if err != nil {
		log.Warn("NewRelayd", "atoi height error: ", err)
	}

	if height < firstHeight || isResetBtcHeight {
		height = firstHeight
	}

	log.Info("NewRelayd", "current btc hegiht: ", height)

	client33 := NewClient33(&config.Chain33)
	var btc BtcClient
	if config.BtcdOrWeb == 0 {
		btc, err = NewBtcd(config.Btcd.BitConnConfig(), config.Btcd.ReconnectAttempts)
	} else {
		btc, err = NewBtcWeb()
	}
	if err != nil {
		panic(err)
	}

	var pk []byte
	private, err := db.Get(privateKey[:])
	if err != nil && private == nil && config.Auth.PrivateKey == "" {
		panic(err)
	}

	if private != nil && config.Auth.PrivateKey == "" {
		pk = private
	} else {
		pk, err = hex.DecodeString(config.Auth.PrivateKey)
		if err != nil && pk == nil && private == nil {
			panic(err)
		}
	}
	db.Set(privateKey[:], pk)

	secp, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		panic(err)
	}

	priKey, err := secp.PrivKeyFromBytes(pk)
	if err != nil {
		panic(err)
	}

	pubkey := priKey.PubKey()
	fmt.Println(pubkey.KeyString())
	address := address.PubKeyToAddress(pubkey.Bytes())
	fmt.Println(address.String())

	return &Relayd{
		config:            config,
		db:                db,
		ctx:               ctx,
		cancel:            cancel,
		client33:          client33,
		knownBtcHeight:    uint64(height),
		firstHeaderHeight: uint64(firstHeight),
		btcClient:         btc,
		privateKey:        priKey,
		publicKey:         pubkey,
		isPersisting:      0,
		isSnycing:         0,
		isResetBtcHeight:  isResetBtcHeight,
	}
}

func (r *Relayd) Close() {
	r.client33.Close()
	r.cancel()
}

func (r *Relayd) Start() {
	r.btcClient.Start()
	r.client33.Start(r.ctx)
	r.tick(r.ctx)
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
				atomic.StoreUint64(&r.latestBtcHeight, height)
			}
			log.Info("tick", "latestBtcHeight: ", height, "knownBtcHeight: ", atomic.LoadUint64(&r.knownBtcHeight))

			if atomic.LoadInt32(&r.isPersisting) == 0 {
				go r.persistBlockHeaders()
			}

			if atomic.LoadInt32(&r.isSnycing) == 0 {
				go r.syncBlockHeaders()
			}

		case <-ping:
			r.btcClient.Ping()
		}
	}
}

func (r *Relayd) persistBlockHeaders() {
	atomic.StoreInt32(&r.isPersisting, 1)
	defer atomic.StoreInt32(&r.isPersisting, 0)
out:
	for i := atomic.LoadUint64(&r.knownBtcHeight); i <= atomic.LoadUint64(&r.latestBtcHeight); i++ {
		header, err := r.btcClient.GetBlockHeader(i)
		if err != nil {
			log.Error("syncBlockHeaders", "GetBlockHeader error", err)
			break out
		}
		data := types.Encode(header)
		r.db.Set(makeHeightKey(i), data)
		r.db.Set(currentBtcBlockheightKey, []byte(fmt.Sprintf("%d", i)))
		atomic.StoreUint64(&r.knownBtcHeight, i)
		if i%10 == 0 || (atomic.LoadUint64(&r.latestBtcHeight)-i) < 10 {
			log.Info("persistBlockHeaders", "current knownBtcHeight: ", i, "current latestBtcHeight: ", atomic.LoadUint64(&r.latestBtcHeight))
		}
	}
}

func (r *Relayd) queryChain33WithBtcHeight() (*ty.ReplayRelayQryBTCHeadHeight, error) {
	payLoad := types.Encode(&ty.ReqRelayQryBTCHeadHeight{})
	query := types.ChainExecutor{
		Driver:   ty.RelayX,
		FuncName: "GetBTCHeaderCurHeight",
		Param:    payLoad,
	}
	ret, err := r.client33.QueryChain(r.ctx, &query)
	if err != nil {
		return nil, err
	}
	if !ret.GetIsOk() {
		log.Info("GetBTCHeaderCurHeight", "error", ret.GetMsg())
	}
	var result ty.ReplayRelayQryBTCHeadHeight
	types.Decode(ret.Msg, &result)
	return &result, nil
}

func (r *Relayd) syncBlockHeaders() {
	atomic.StoreInt32(&r.isSnycing, 1)
	defer atomic.StoreInt32(&r.isSnycing, 0)

	knownBtcHeight := atomic.LoadUint64(&r.knownBtcHeight)
	if knownBtcHeight > r.firstHeaderHeight {
		ret, err := r.queryChain33WithBtcHeight()
		if err != nil {
			log.Error("syncBlockHeaders", "queryChain33WithBtcHeight error: ", err)
			return
		}

		log.Info("syncBlockHeaders", "queryChain33WithBtcHeight result: ", ret)
		var initIterHeight uint64
		if r.firstHeaderHeight != uint64(ret.BaseHeight) && !r.isResetBtcHeight {
			r.isResetBtcHeight = true
		}

		var total uint64
		if r.isResetBtcHeight {
			initIterHeight = r.firstHeaderHeight
			total = knownBtcHeight - initIterHeight
		} else {
			initIterHeight = uint64(ret.CurHeight) + 1
			if initIterHeight >= knownBtcHeight {
				return
			}
			total = knownBtcHeight - initIterHeight + 1
		}

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
			headers := make([]*ty.BtcHeader, 0, add)
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
			btcHeaders := &ty.BtcHeaders{BtcHeader: headers}
			relayHeaders := &ty.RelayAction_BtcHeaders{btcHeaders}
			action := &ty.RelayAction{
				Value: relayHeaders,
				Ty:    ty.RelayActionRcvBTCHeaders,
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
		Execer:  []byte(ty.RelayX),
		Payload: payload,
		Nonce:   rand.Int63(),
		To:      address.ExecAddress(ty.RelayX),
	}

	fee, _ := tx.GetRealFee(types.GInt("MinFee"))
	tx.Fee = fee
	tx.Sign(types.SECP256K1, r.privateKey)
	log.Info("transaction", "fee : ", fee)
	return tx
}

func (r *Relayd) dealOrder() {
	result, err := r.requestRelayOrders(ty.RelayOrderStatus_confirming)
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
		verify := &ty.RelayVerify{
			OrderId: value.Id,
			Tx:      tx,
			Spv:     spv,
		}
		rr := &ty.RelayAction_Verify{
			verify,
		}
		action := &ty.RelayAction{
			Value: rr,
			Ty:    ty.RelayActionVerifyTx,
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

func (r *Relayd) requestRelayOrders(status ty.RelayOrderStatus) (*ty.QueryRelayOrderResult, error) {
	payLoad := types.Encode(&ty.ReqRelayAddrCoins{
		Status: status,
	})
	query := types.ChainExecutor{
		Driver:   ty.RelayX,
		FuncName: "GetRelayOrderByStatus",
		Param:    payLoad,
	}
	ret, err := r.client33.QueryChain(r.ctx, &query)
	if err != nil {
		return nil, err
	}
	if !ret.GetIsOk() {
		log.Info("requestRelayOrders", "error")
	}
	var result ty.QueryRelayOrderResult
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
