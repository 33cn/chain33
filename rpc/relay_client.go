package rpc

import (
	"math/rand"
	"time"

	"encoding/hex"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/types"
)

/////////////types.go///////////////////////////

//Relay Transaction
type RelaySellTx struct {
	SellAmount uint64 `json:"amount_of_BTY"`
	Coin       string `json:"symbol_to_exchange"`
	CoinAmount uint64 `json:"amount_to_exchange"`
	CoinAddr   string `json:"addr_of_coin"`
	CoinWait   uint32 `json:"blocks_to_wait_verify"`
	Fee        int64  `json:"fee"`
}

type RelayRevokeSellTx struct {
	OrderId string `json:"order_id"`
	Fee     int64  `json:"fee"`
}

type RelayBuyTx struct {
	OrderId string `json:"order_id"`
	Fee     int64  `json:"fee"`
}

type RelayRevokeBuyTx struct {
	OrderId string `json:"order_id"`
	Fee     int64  `json:"fee"`
}

type RelayVerifyTx struct {
	OrderId string `json:"order_id"`
	Txhash  string `json:"tx_hash"`
	Fee     int64  `json:"fee"`
}

type RelayVerifyBTCTx struct {
	OrderId     string `json:"order_id"`
	RawTx       string `json:"raw_tx"`
	TxIndex     uint32 `json:"tx_index"`
	MerklBranch string `json:"merkle_branch"`
	BlockHash   string `json:"block_hash"`
	Fee         int64  `json:"fee"`
}

type RelaySaveBTCHeadTx struct {
	Hash          string `json:"hash"`
	Confirmations uint64 `json:"confirmations"`
	Height        uint64 `json:"height"`
	Version       uint32 `json:"version"`
	MerkleRoot    string `json:"merkleRoot"`
	Time          int64  `json:"time"`
	Nonce         int64  `json:"nonce"`
	Bits          int64  `json:"bits"`
	Difficulty    int64  `json:"difficulty"`
	PreviousHash  string `json:"previousHash"`
	NextHash      string `json:"nextHash"`
	IsReset       bool   `json:"isReset"`
	Fee           int64  `json:"fee"`
}

///////////////cli.go/////
//func init() {
//	rootCmd.AddCommand(
//		commands.RelayCmd(),
//)
//}

//////////////client.go////////////////////////////////////

func (c *channelClient) CreateRawRelaySellTx(parm *RelaySellTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.RelaySell{
		Sellamount:     parm.SellAmount,
		Exchgcoin:      parm.Coin,
		Exchgamount:    parm.CoinAmount,
		Exchgaddr:      parm.CoinAddr,
		Waitcoinblocks: parm.CoinWait,
	}
	sell := &types.RelayAction{
		Ty:    types.RelayActionSell,
		Value: &types.RelayAction_Rsell{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("relay"),
		Payload: types.Encode(sell),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("relay").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayRvkSellTx(parm *RelayRevokeSellTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.RelayRevokeSell{Orderid: parm.OrderId}
	val := &types.RelayAction{
		Ty:    types.RelayActionRevokeSell,
		Value: &types.RelayAction_Rrevokesell{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("relay"),
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("relay").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayBuyTx(parm *RelayBuyTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.RelayBuy{Orderid: parm.OrderId}
	val := &types.RelayAction{
		Ty:    types.RelayActionBuy,
		Value: &types.RelayAction_Rbuy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("relay"),
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("relay").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayRvkBuyTx(parm *RelayRevokeBuyTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.RelayRevokeBuy{Orderid: parm.OrderId}
	val := &types.RelayAction{
		Ty:    types.RelayActionRevokeBuy,
		Value: &types.RelayAction_Rrevokebuy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("relay"),
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("relay").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayConfirmTx(parm *RelayVerifyTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.RelayConfirmTx{Orderid: parm.OrderId, Txhash: parm.Txhash}
	val := &types.RelayAction{
		Ty:    types.RelayActionConfirmTx,
		Value: &types.RelayAction_Rconfirmtx{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("relay"),
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("relay").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelayVerifyBTCTx(parm *RelayVerifyBTCTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.RelayVerifyBTC{
		Orderid:    parm.OrderId,
		Rawtx:      parm.RawTx,
		Txindex:    parm.TxIndex,
		Merkbranch: parm.MerklBranch,
		Blockhash:  parm.BlockHash}
	val := &types.RelayAction{
		Ty:    types.RelayActionVerifyBTCTx,
		Value: &types.RelayAction_Rverifybtc{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("relay"),
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("relay").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

func (c *channelClient) CreateRawRelaySaveBTCHeadTx(parm *RelaySaveBTCHeadTx) ([]byte, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	head := &types.BtcHeader{
		Hash:         parm.Hash,
		PreviousHash: parm.PreviousHash,
		MerkleRoot:   parm.MerkleRoot,
		Height:       parm.Height,
		IsReset:      parm.IsReset,
	}

	v := &types.BtcHeaders{}
	v.BtcHeader = append(v.BtcHeader, head)

	val := &types.RelayAction{
		Ty:    types.RelayActionRcvBTCHeaders,
		Value: &types.RelayAction_BtcHeaders{v},
	}
	tx := &types.Transaction{
		Execer:  []byte("relay"),
		Payload: types.Encode(val),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("relay").String(),
	}

	data := types.Encode(tx)
	return data, nil
}

/////////////////////jrpchandler.go/////////////////////////////////

func (c *Chain33) CreateRawRelaySellTx(in *RelaySellTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelaySellTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawRelayRvkSellTx(in *RelayRevokeSellTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayRvkSellTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawRelayBuyTx(in *RelayBuyTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayBuyTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}
func (c *Chain33) CreateRawRelayRvkBuyTx(in *RelayRevokeBuyTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayRvkBuyTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}
func (c *Chain33) CreateRawRelayConfirmTx(in *RelayVerifyTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayConfirmTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}
func (c *Chain33) CreateRawRelayVerifyBTCTx(in *RelayVerifyBTCTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelayVerifyBTCTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

func (c *Chain33) CreateRawRelaySaveBTCHeadTx(in *RelaySaveBTCHeadTx, result *interface{}) error {
	reply, err := c.cli.CreateRawRelaySaveBTCHeadTx(in)
	if err != nil {
		return err
	}

	*result = hex.EncodeToString(reply)
	return nil
}

//////////queryPayload//////////////
func relayPayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "GetRelayOrderByStatus":
		req = &types.ReqRelayAddrCoins{}
	case "GetSellRelayOrder":
		req = &types.ReqRelayAddrCoins{}
	case "GetBuyRelayOrder":
		req = &types.ReqRelayAddrCoins{}
	case "GetBTCHeaderList":
		req = &types.ReqRelayBtcHeaderHeightList{}
	case "GetBTCHeaderMissList":
		req = &types.ReqRelayBtcHeaderHeightList{}
	case "GetBTCHeaderCurHeight":
		req = &types.ReqRelayQryBTCHeadHeight{}
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}
