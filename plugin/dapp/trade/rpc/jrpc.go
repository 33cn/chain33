package rpc

import (
	"context"
	"encoding/hex"

	"gitlab.33.cn/chain33/chain33/types"

	ptypes "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
)

func (this *Jrpc) CreateRawTradeSellTx(in *ptypes.TradeSellTx, result *interface{}) error {
	if in == nil {
		return types.ErrInvalidParam
	}
	param := &ptypes.TradeForSell{
		TokenSymbol:       in.TokenSymbol,
		AmountPerBoardlot: in.AmountPerBoardlot,
		MinBoardlot:       in.MinBoardlot,
		PricePerBoardlot:  in.PricePerBoardlot,
		TotalBoardlot:     in.TotalBoardlot,
		Starttime:         0,
		Stoptime:          0,
		Crowdfund:         false,
		AssetExec:         in.AssetExec,
	}

	reply, err := this.cli.CreateRawTradeSellTx(context.Background(), param)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply.Data)
	return nil
}

func (this *Jrpc) CreateRawTradeBuyTx(in *ptypes.TradeBuyTx, result *interface{}) error {
	if in == nil {
		return types.ErrInvalidParam
	}
	param := &ptypes.TradeForBuy{
		SellID:      in.SellID,
		BoardlotCnt: in.BoardlotCnt,
	}

	reply, err := this.cli.CreateRawTradeBuyTx(context.Background(), param)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply.Data)
	return nil
}

func (this *Jrpc) CreateRawTradeRevokeTx(in *ptypes.TradeRevokeTx, result *interface{}) error {
	if in == nil {
		return types.ErrInvalidParam
	}
	param := &ptypes.TradeForRevokeSell{
		SellID: in.SellID,
	}

	reply, err := this.cli.CreateRawTradeRevokeTx(context.Background(), param)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply.Data)
	return nil
}

func (this *Jrpc) CreateRawTradeBuyLimitTx(in *ptypes.TradeBuyLimitTx, result *interface{}) error {
	if in == nil {
		return types.ErrInvalidParam
	}
	param := &ptypes.TradeForBuyLimit{
		TokenSymbol:       in.TokenSymbol,
		AmountPerBoardlot: in.AmountPerBoardlot,
		MinBoardlot:       in.MinBoardlot,
		PricePerBoardlot:  in.PricePerBoardlot,
		TotalBoardlot:     in.TotalBoardlot,
		AssetExec:         in.AssetExec,
	}

	reply, err := this.cli.CreateRawTradeBuyLimitTx(context.Background(), param)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply.Data)
	return nil
}

func (this *Jrpc) CreateRawTradeSellMarketTx(in *ptypes.TradeSellMarketTx, result *interface{}) error {
	if in == nil {
		return types.ErrInvalidParam
	}
	param := &ptypes.TradeForSellMarket{
		BuyID:       in.BuyID,
		BoardlotCnt: in.BoardlotCnt,
	}

	reply, err := this.cli.CreateRawTradeSellMarketTx(context.Background(), param)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply.Data)
	return nil
}

func (this *Jrpc) CreateRawTradeRevokeBuyTx(in *ptypes.TradeRevokeBuyTx, result *interface{}) error {
	if in == nil {
		return types.ErrInvalidParam
	}
	param := &ptypes.TradeForRevokeBuy{
		BuyID: in.BuyID,
	}

	reply, err := this.cli.CreateRawTradeRevokeBuyTx(context.Background(), param)
	if err != nil {
		return err
	}
	*result = hex.EncodeToString(reply.Data)
	return nil
}
