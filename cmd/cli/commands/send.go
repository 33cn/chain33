package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func SendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send transaction in one move",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		SendTransferCmd(),
		SendWithdrawCmd(),
		SendTokenTransferCmd(),
		SendTokenWithdrawCmd(),
		SendTokenPreCreateTxCmd(),
		SendTokenFinishTxCmd(),
		SendTokenRevokeTxCmd(),
		SendTradeSellTxCmd(),
		SendTradeBuyTxCmd(),
		SendTradeRevokeTxCmd(),
	)

	return cmd
}

func SendTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bty_transfer",
		Short: "Send bty transfer transaction",
		Run:   sendTransaction,
	}
	addCreateTransferFlags(cmd)
	addKeyFlag(cmd)
	return cmd
}

func SendWithdrawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bty_withdraw",
		Short: "Send bty withdraw transaction",
		Run:   sendTransaction,
	}
	addCreateWithdrawFlags(cmd)
	addKeyFlag(cmd)
	return cmd
}

func SendTokenTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token_transfer",
		Short: "Send token transfer transaction",
		Run:   sendTransaction,
	}
	addCreateTokenTransferFlags(cmd)
	addKeyFlag(cmd)
	return cmd
}

func SendTokenWithdrawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token_withdraw",
		Short: "Send token withdraw transaction",
		Run:   sendTransaction,
	}
	addCreateTokenWithdrawFlags(cmd)
	addKeyFlag(cmd)
	return cmd
}

func SendTokenPreCreateTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token_precreate",
		Short: "Send token precreate transaction",
		Run:   sendTransaction,
	}
	addTokenPrecreatedFlags(cmd)
	addKeyFlag(cmd)
	return cmd
}

func SendTokenFinishTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token_finish",
		Short: "Send token finish transaction",
		Run:   sendTransaction,
	}
	addTokenFinishFlags(cmd)
	addKeyFlag(cmd)
	return cmd
}

func SendTokenRevokeTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token_revoke",
		Short: "Send token revoke transaction",
		Run:   sendTransaction,
	}
	addTokenRevokeFlags(cmd)
	addKeyFlag(cmd)
	return cmd
}

func SendTradeSellTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trade_sell",
		Short: "Send trade sell transaction",
		Run:   sendTransaction,
	}
	addTokenSellFlags(cmd)
	addKeyFlag(cmd)
	return cmd
}

func SendTradeBuyTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trade_buy",
		Short: "Send trade buy transaction",
		Run:   sendTransaction,
	}
	addTokenBuyFlags(cmd)
	addKeyFlag(cmd)
	return cmd
}

func SendTradeRevokeTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trade_revoke",
		Short: "Send trade revoke transaction",
		Run:   sendTransaction,
	}
	addTokenSellRevokeFlags(cmd)
	addKeyFlag(cmd)
	return cmd
}

func addKeyFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("key", "k", "", "private key")
	cmd.MarkFlagRequired("key")
}

func signTx(cmd *cobra.Command, data string) (string, error) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	key, _ := cmd.Flags().GetString("key")
	params := types.ReqSignRawTx{
		Privkey: key,
		TxHex:   data,
		Expire:  "120s",
	}
	ctx := NewRpcCtx(rpcLaddr, "Chain33.SignRawTx", params, nil)
	return ctx.RunWithoutMarshal()
}

func sendTransaction(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var txHex string
	switch cmd.Use {
	case "bty_transfer":
		toAddr, _ := cmd.Flags().GetString("to")
		amount, _ := cmd.Flags().GetFloat64("amount")
		note, _ := cmd.Flags().GetString("note")
		txHex, _ = CreateRawTx(toAddr, amount, note, false, false, "")
	case "bty_withdraw":
		exec, _ := cmd.Flags().GetString("exec")
		amount, _ := cmd.Flags().GetFloat64("amount")
		note, _ := cmd.Flags().GetString("note")
		execAddr, err := GetExecAddr(exec)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		txHex, _ = CreateRawTx(execAddr, amount, note, true, false, "")
	case "token_transfer":
		toAddr, _ := cmd.Flags().GetString("to")
		amount, _ := cmd.Flags().GetFloat64("amount")
		note, _ := cmd.Flags().GetString("note")
		symbol, _ := cmd.Flags().GetString("symbol")
		txHex, _ = CreateRawTx(toAddr, amount, note, false, true, symbol)
	case "token_withdraw":
		exec, _ := cmd.Flags().GetString("exec")
		amount, _ := cmd.Flags().GetFloat64("amount")
		note, _ := cmd.Flags().GetString("note")
		symbol, _ := cmd.Flags().GetString("symbol")
		execAddr, err := GetExecAddr(exec)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		txHex, _ = CreateRawTx(execAddr, amount, note, true, true, symbol)
	case "token_precreate":
		name, _ := cmd.Flags().GetString("name")
		symbol, _ := cmd.Flags().GetString("symbol")
		introduction, _ := cmd.Flags().GetString("introduction")
		ownerAddr, _ := cmd.Flags().GetString("owner_addr")
		price, _ := cmd.Flags().GetFloat64("price")
		fee, _ := cmd.Flags().GetFloat64("fee")
		total, _ := cmd.Flags().GetInt64("total")
		priceInt64 := int64(price * 1e4)
		feeInt64 := int64(fee * 1e4)
		params := &jsonrpc.TokenPreCreateTx{
			Price:        priceInt64 * 1e4,
			Name:         name,
			Symbol:       symbol,
			Introduction: introduction,
			OwnerAddr:    ownerAddr,
			Total:        total * types.TokenPrecision,
			Fee:          feeInt64 * 1e4,
		}
		ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTokenPreCreateTx", params, nil)
		txHex, _ = ctx.RunWithoutMarshal()
	case "token_finish":
		symbol, _ := cmd.Flags().GetString("symbol")
		ownerAddr, _ := cmd.Flags().GetString("owner_addr")
		fee, _ := cmd.Flags().GetFloat64("fee")
		feeInt64 := int64(fee * 1e4)
		params := &jsonrpc.TokenFinishTx{
			Symbol:    symbol,
			OwnerAddr: ownerAddr,
			Fee:       feeInt64 * 1e4,
		}
		ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTokenFinishTx", params, nil)
		txHex, _ = ctx.RunWithoutMarshal()
	case "token_revoke":
		symbol, _ := cmd.Flags().GetString("symbol")
		ownerAddr, _ := cmd.Flags().GetString("owner_addr")
		fee, _ := cmd.Flags().GetFloat64("fee")
		feeInt64 := int64(fee * 1e4)
		params := &jsonrpc.TokenRevokeTx{
			Symbol:    symbol,
			OwnerAddr: ownerAddr,
			Fee:       feeInt64 * 1e4,
		}
		ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTokenRevokeTx", params, nil)
		txHex, _ = ctx.RunWithoutMarshal()
	case "trade_sell":
		symbol, _ := cmd.Flags().GetString("symbol")
		min, _ := cmd.Flags().GetInt64("min")
		price, _ := cmd.Flags().GetFloat64("price")
		fee, _ := cmd.Flags().GetFloat64("fee")
		total, _ := cmd.Flags().GetFloat64("total")
		priceInt64 := int64(price * 1e4)
		feeInt64 := int64(fee * 1e4)
		totalInt64 := int64(total * 1e8 / 1e6)
		params := &jsonrpc.TradeSellTx{
			TokenSymbol:       symbol,
			AmountPerBoardlot: 1e6,
			MinBoardlot:       min,
			PricePerBoardlot:  priceInt64 * 1e4,
			TotalBoardlot:     totalInt64,
			Fee:               feeInt64 * 1e4,
		}
		ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTradeSellTx", params, nil)
		txHex, _ = ctx.RunWithoutMarshal()
	case "trade_buy":
		sellID, _ := cmd.Flags().GetString("sell_id")
		fee, _ := cmd.Flags().GetFloat64("fee")
		count, _ := cmd.Flags().GetInt64("count")
		feeInt64 := int64(fee * 1e4)
		params := &jsonrpc.TradeBuyTx{
			SellID:      sellID,
			BoardlotCnt: count,
			Fee:         feeInt64 * 1e4,
		}
		ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTradeBuyTx", params, nil)
		txHex, _ = ctx.RunWithoutMarshal()
	case "trade_revoke":
		sellID, _ := cmd.Flags().GetString("sell_id")
		fee, _ := cmd.Flags().GetFloat64("fee")
		feeInt64 := int64(fee * 1e4)
		params := &jsonrpc.TradeRevokeTx{
			SellID: sellID,
			Fee:    feeInt64 * 1e4,
		}
		ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateRawTradeRevokeTx", params, nil)
		txHex, _ = ctx.RunWithoutMarshal()
	default:
		fmt.Fprintln(os.Stderr, "ErrCLICode")
		return
	}

	txSigned, err := signTx(cmd, txHex)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := jsonrpc.RawParm{
		Data: txSigned,
	}
	ctx := NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
	txHash, _ := ctx.RunWithoutMarshal()
	fmt.Println(txHash)
}
