package rpc

import (
	"context"
	"math/rand"
	"time"

	"gitlab.33.cn/chain33/chain33/common/address"

	ptypes "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (this *channelClient) CreateRawTradeSellTx(ctx context.Context, in *ptypes.TradeForSell) (*types.UnsignTx, error) {
	if in == nil {
		return nil, types.ErrInvalidParam
	}
	sell := &ptypes.Trade{
		Ty:    ptypes.TradeSellLimit,
		Value: &ptypes.Trade_SellLimit{SellLimit: in},
	}
	tx := &types.Transaction{
		Execer:  []byte(ptypes.TradeX),
		Payload: types.Encode(sell),
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(ptypes.TradeX),
	}
	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}

func (this *channelClient) CreateRawTradeBuyTx(ctx context.Context, in *ptypes.TradeForBuy) (*types.UnsignTx, error) {
	if in == nil {
		return nil, types.ErrInvalidParam
	}
	buy := &ptypes.Trade{
		Ty:    ptypes.TradeBuyMarket,
		Value: &ptypes.Trade_BuyMarket{in},
	}
	tx := &types.Transaction{
		Execer:  []byte(ptypes.TradeX),
		Payload: types.Encode(buy),
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(ptypes.TradeX),
	}
	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}

func (this *channelClient) CreateRawTradeRevokeTx(ctx context.Context, in *ptypes.TradeForRevokeSell) (*types.UnsignTx, error) {
	if in == nil {
		return nil, types.ErrInvalidParam
	}
	buy := &ptypes.Trade{
		Ty:    ptypes.TradeRevokeSell,
		Value: &ptypes.Trade_RevokeSell{in},
	}
	tx := &types.Transaction{
		Execer:  []byte(ptypes.TradeX),
		Payload: types.Encode(buy),
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(ptypes.TradeX),
	}
	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}

func (this *channelClient) CreateRawTradeBuyLimitTx(ctx context.Context, in *ptypes.TradeForBuyLimit) (*types.UnsignTx, error) {
	if in == nil {
		return nil, types.ErrInvalidParam
	}
	buy := &ptypes.Trade{
		Ty:    ptypes.TradeBuyLimit,
		Value: &ptypes.Trade_BuyLimit{in},
	}
	tx := &types.Transaction{
		Execer:  []byte(ptypes.TradeX),
		Payload: types.Encode(buy),
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(ptypes.TradeX),
	}
	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}

func (this *channelClient) CreateRawTradeSellMarketTx(ctx context.Context, in *ptypes.TradeForSellMarket) (*types.UnsignTx, error) {
	if in == nil {
		return nil, types.ErrInvalidParam
	}
	buy := &ptypes.Trade{
		Ty:    ptypes.TradeSellMarket,
		Value: &ptypes.Trade_SellMarket{in},
	}
	tx := &types.Transaction{
		Execer:  []byte(ptypes.TradeX),
		Payload: types.Encode(buy),
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(ptypes.TradeX),
	}
	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}

func (this *channelClient) CreateRawTradeRevokeBuyTx(ctx context.Context, in *ptypes.TradeForRevokeBuy) (*types.UnsignTx, error) {
	if in == nil {
		return nil, types.ErrInvalidParam
	}
	buy := &ptypes.Trade{
		Ty:    ptypes.TradeRevokeBuy,
		Value: &ptypes.Trade_RevokeBuy{in},
	}
	tx := &types.Transaction{
		Execer:  []byte(ptypes.TradeX),
		Payload: types.Encode(buy),
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(ptypes.TradeX),
	}
	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	data := types.Encode(tx)
	return &types.UnsignTx{Data: data}, nil
}
