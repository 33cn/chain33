package commands

import (
	"encoding/hex"
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func PrivacyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "privacy",
		Short: "Privacy transaction management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		ShowPrivacyKeyCmd(),
		ShowPrivacyAccountCmd(),
		ShowPrivacyAccountDetailCmd(),
		Public2PrivacyCmd(),
		Privacy2PrivacyCmd(),
		Privacy2PublicCmd(),
		ShowAmountsOfUTXOCmd(),
		ShowUTXOs4SpecifiedAmountCmd(),
		CreateUTXOsCmd(),
	)

	return cmd
}

// show privacy key by address
func ShowPrivacyKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Show privacy key by address",
		Run:   showPrivacyKey,
	}
	showPrivacyKeyFlag(cmd)
	return cmd
}

func showPrivacyKeyFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")

}

func showPrivacyKey(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")

	params := types.ReqStr{
		ReqStr: addr,
	}

	var res types.ReplyPrivacyPkPair
	ctx := NewRpcCtx(rpcLaddr, "Chain33.ShowPrivacykey", params, &res)
	ctx.Run()
}

// public address to privacy address
func Public2PrivacyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pub2priv",
		Short: "Public to privacy from toviewpubkey tospendpubkey amout note",
		Run:   public2Privacy,
	}
	public2PrivacyFlag(cmd)
	return cmd
}

func public2PrivacyFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("from", "f", "", "from account address")
	cmd.MarkFlagRequired("from")

	cmd.Flags().StringP("toviewpubkey", "v", "", "to view public key")
	cmd.MarkFlagRequired("toviewpubkey")

	cmd.Flags().StringP("tospendpubkey", "s", "", "to spend public key")
	cmd.MarkFlagRequired("tospendpubkey")

	cmd.Flags().Float64P("amount", "a", 0, "transfer amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transfer note")
	cmd.MarkFlagRequired("note")

}

func public2Privacy(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	from, _ := cmd.Flags().GetString("from")
	toviewpubkey, _ := cmd.Flags().GetString("toviewpubkey")
	tospendpubkey, _ := cmd.Flags().GetString("tospendpubkey")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")

	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	params := types.ReqPub2Pri{
		Sender:      from,
		ViewPublic:  toviewpubkey,
		SpendPublic: tospendpubkey,
		Amount:      amountInt64,
		Note:        note,
		Tokenname:   types.BTY,
	}

	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcLaddr, "Chain33.MakeTxPublic2privacy", params, &res)
	ctx.Run()
}

// privacy address to privacy address
func Privacy2PrivacyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "priv2priv",
		Short: "Privacy to privacy from toviewpubkey tospendpubkey amout note",
		Run:   privacy2Privacy,
	}
	privacy2PrivacyFlag(cmd)
	return cmd
}

func privacy2PrivacyFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("from", "f", "", "from account address")
	cmd.MarkFlagRequired("from")

	cmd.Flags().StringP("toviewpubkey", "v", "", "to view public key")
	cmd.MarkFlagRequired("toviewpubkey")

	cmd.Flags().StringP("tospendpubkey", "s", "", "to spend public key")
	cmd.MarkFlagRequired("tospendpubkey")

	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().Int32P("mixcount", "m", 0, "mix count")
	cmd.MarkFlagRequired("mixcount")

	cmd.Flags().StringP("note", "n", "", "transfer note")
	cmd.MarkFlagRequired("note")

}

func privacy2Privacy(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	from, _ := cmd.Flags().GetString("from")
	toviewpubkey, _ := cmd.Flags().GetString("toviewpubkey")
	tospendpubkey, _ := cmd.Flags().GetString("tospendpubkey")
	amount, _ := cmd.Flags().GetFloat64("amount")
	mixcount, _ := cmd.Flags().GetInt32("txhash")
	note, _ := cmd.Flags().GetString("note")

	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	params := types.ReqPri2Pri{
		Sender:      from,
		ViewPublic:  toviewpubkey,
		SpendPublic: tospendpubkey,
		Amount:      amountInt64,
		Mixin:       mixcount,
		Note:        note,
		Tokenname:   types.BTY,
	}

	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcLaddr, "Chain33.MakeTxPrivacy2privacy", params, &res)
	ctx.Run()
}

// privacy address to public address
func Privacy2PublicCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "priv2pub",
		Short: "Public to privacy from toviewpubkey tospendpubkey amout note",
		Run:   privacy2Public,
	}
	privacy2Publiclag(cmd)
	return cmd
}

func privacy2Publiclag(cmd *cobra.Command) {
	cmd.Flags().StringP("from", "f", "", "from account address")
	cmd.MarkFlagRequired("from")

	cmd.Flags().StringP("to", "t", "", "to account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().Int32P("mixcount", "m", 0, "mix count")
	cmd.MarkFlagRequired("mixcount")

	cmd.Flags().StringP("note", "n", "", "transfer note")
	cmd.MarkFlagRequired("note")

}

func privacy2Public(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	mixcount, _ := cmd.Flags().GetInt32("txhash")
	note, _ := cmd.Flags().GetString("note")

	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	params := types.ReqPri2Pub{
		Sender:    from,
		Receiver:  to,
		Amount:    amountInt64,
		Note:      note,
		Tokenname: types.BTY,
		Mixin:     mixcount,
	}

	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcLaddr, "Chain33.MakeTxPrivacy2public", params, &res)
	ctx.Run()
}

func ShowPrivacyAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "showpa",
		Short: "Show privacy account command",
		Run:   showPrivacyAccount,
	}
	showPrivacyAccountFlag(cmd)
	return cmd
}

func showPrivacyAccountFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")
}

func showPrivacyAccount(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")

	params := types.ReqPrivBal4AddrToken{
		Addr:  addr,
		Token: types.BTY,
	}

	var res []*types.UTXO
	ctx := NewRpcCtx(rpcLaddr, "Chain33.ShowPrivacyAccount", params, &res)
	ctx.Run()
}

func ShowPrivacyAccountDetailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "showpad",
		Short: "Show privacy account detail command",
		Run:   showPrivacyAccountDetail,
	}
	showPrivacyAccountDetailFlag(cmd)
	return cmd
}

func showPrivacyAccountDetailFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")
}

func showPrivacyAccountDetail(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")

	params := types.ReqPrivBal4AddrToken{
		Addr:  addr,
		Token: types.BTY,
	}

	var res []*types.UTXO
	ctx := NewRpcCtx(rpcLaddr, "Chain33.ShowPrivacyAccount", params, &res)
	ctx.Run()
}

func ShowAmountsOfUTXOCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "showau",
		Short: "Show Amount of UTXO",
		Run:   showAmountOfUTXO,
	}
	showAmountOfUTXOFlag(cmd)
	return cmd
}

func showAmountOfUTXOFlag(cmd *cobra.Command) {
}

func showAmountOfUTXO(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")

	reqPrivacyToken := types.ReqPrivacyToken{Token: types.BTY}
	var params jsonrpc.Query4Cli
	params.Execer = types.PrivacyX
	params.FuncName = "ShowAmountsOfUTXO"
	params.Payload = hex.EncodeToString(types.Encode(&reqPrivacyToken))

	var res types.ReplyPrivacyAmounts
	ctx := NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.Run()
}

func ShowUTXOs4SpecifiedAmountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "showutxo4a",
		Short: "Show specified amount UTXOs",
		Run:   showUTXOs4SpecifiedAmount,
	}
	showUTXOs4SpecifiedAmountFlag(cmd)
	return cmd
}

func showUTXOs4SpecifiedAmountFlag(cmd *cobra.Command) {
	cmd.Flags().Float64P("amount", "a", 0, "amount")
	cmd.MarkFlagRequired("amount")
}

func showUTXOs4SpecifiedAmount(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	amount, _ := cmd.Flags().GetFloat64("amount")
	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4

	reqPrivacyToken := types.ReqPrivacyToken{
		Token:  types.BTY,
		Amount: amountInt64,
	}
	var params jsonrpc.Query4Cli
	params.Execer = types.PrivacyX
	params.FuncName = "ShowUTXOs4SpecifiedAmount"
	params.Payload = hex.EncodeToString(types.Encode(&reqPrivacyToken))

	var res types.ReplyUTXOsOfAmount
	ctx := NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.Run()
}

func CreateUTXOsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "createutxos",
		Short: "Show specified amount UTXOs",
		Run:   createUTXOs,
	}
	createUTXOsFlag(cmd)
	return cmd
}

func createUTXOsFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("from", "f", "", "from account address")
	cmd.MarkFlagRequired("from")
	cmd.Flags().StringP("toviewpubkey", "v", "", "to view public key")
	cmd.MarkFlagRequired("toviewpubkey")
	cmd.Flags().StringP("tospendpubkey", "s", "", "to spend public key")
	cmd.MarkFlagRequired("tospendpubkey")
	cmd.Flags().Float64P("amount", "a", 0, "amount")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().Int32P("count", "c", 0, "mix count")
	cmd.MarkFlagRequired("count")
	cmd.Flags().StringP("note", "n", "", "transfer note")
	cmd.MarkFlagRequired("note")
}

func createUTXOs(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	from, _ := cmd.Flags().GetString("from")
	toviewpubkey, _ := cmd.Flags().GetString("toviewpubkey")
	tospendpubkey, _ := cmd.Flags().GetString("tospendpubkey")
	note, _ := cmd.Flags().GetString("note")
	count, _ := cmd.Flags().GetInt32("count")
	amount, _ := cmd.Flags().GetFloat64("amount")
	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4

	params := &types.ReqCreateUTXOs{
		Sender:      from,
		ViewPublic:  toviewpubkey,
		SpendPublic: tospendpubkey,
		Amount:      amountInt64,
		Count:       count,
		Note:        note,
	}

	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateUTXOs", params, &res)
	ctx.Run()
}
