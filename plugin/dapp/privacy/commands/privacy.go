package commands

import (
	"fmt"
	"strconv"
	"time"

	"strings"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	cmdtypes "gitlab.33.cn/chain33/chain33/system/dapp/commands/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	defMixCount int32 = 16
)

func PrivacyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "privacy",
		Short: "Privacy transaction management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		ShowPrivacyKeyCmd(),
		ShowPrivacyAccountSpendCmd(),
		Public2PrivacyCmd(),
		Privacy2PrivacyCmd(),
		Privacy2PublicCmd(),
		ShowAmountsOfUTXOCmd(),
		ShowUTXOs4SpecifiedAmountCmd(),
		CreateUTXOsCmd(),
		ShowPrivacyAccountInfoCmd(),
		ListPrivacyTxsCmd(),
		RescanUtxosOptCmd(),
		EnablePrivacyCmd(),
	)

	return cmd
}

// ShowPrivacyKeyCmd show privacy key by address
func ShowPrivacyKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "showpk",
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
	params := types.ReqString{
		Data: addr,
	}
	var res pty.ReplyPrivacyPkPair
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.ShowPrivacykey", params, &res)
	ctx.Run()
}

// Public2PrivacyCmd public address to privacy address
func Public2PrivacyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pub2priv",
		Short: "Public to privacy from pubkeypair amout note",
		Run:   public2Privacy,
	}
	public2PrivacyFlag(cmd)
	return cmd
}

func public2PrivacyFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("from", "f", "", "from account address")
	cmd.MarkFlagRequired("from")

	cmd.Flags().StringP("pubkeypair", "p", "", "to view spend public key pair")
	cmd.MarkFlagRequired("pubkeypair")

	cmd.Flags().Float64P("amount", "a", 0, "transfer amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transfer note")
	cmd.Flags().Int64P("expire", "", 0, "transfer expire, default one hour")
	cmd.Flags().IntP("expiretype", "", 1, "0: height  1: time default is 1")

}

func public2Privacy(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	from, _ := cmd.Flags().GetString("from")
	pubkeypair, _ := cmd.Flags().GetString("pubkeypair")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	expire, _ := cmd.Flags().GetInt64("expire")
	expiretype, _ := cmd.Flags().GetInt("expiretype")
	if expiretype == 0 {
		if expire <= 0 {
			fmt.Println("Invalid expire. expire must large than 0 in expiretype==0, expire", expire)
			return
		}
	} else if expiretype == 1 {
		if expire <= 0 {
			expire = int64(time.Hour)
		}
	} else {
		fmt.Println("Invalid expiretype", expiretype)
		return
	}

	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	params := pty.ReqPub2Pri{
		Sender:     from,
		Pubkeypair: pubkeypair,
		Amount:     amountInt64,
		Note:       note,
		Tokenname:  types.BTY,
		Expire:     expire,
	}

	var res rpctypes.ReplyHash
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.MakeTxPublic2privacy", params, &res)
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

	cmd.Flags().StringP("pubkeypair", "p", "", "to view spend public key pair")
	cmd.MarkFlagRequired("pubkeypair")

	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transfer note")
	cmd.Flags().Int32P("mixcount", "m", defMixCount, "transfer note")
	cmd.Flags().Int64P("expire", "", 0, "transfer expire, default one hour")
	cmd.Flags().IntP("expiretype", "", 1, "0: height  1: time default is 1")
}

func privacy2Privacy(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	from, _ := cmd.Flags().GetString("from")
	pubkeypair, _ := cmd.Flags().GetString("pubkeypair")
	amount, _ := cmd.Flags().GetFloat64("amount")
	mixcount, _ := cmd.Flags().GetInt32("mixcount")
	note, _ := cmd.Flags().GetString("note")
	expire, _ := cmd.Flags().GetInt64("expire")
	expiretype, _ := cmd.Flags().GetInt("expiretype")
	if expiretype == 0 {
		if expire <= 0 {
			fmt.Println("Invalid expire. expire must large than 0 in expiretype==0, expire", expire)
			return
		}
	} else if expiretype == 1 {
		if expire <= 0 {
			expire = int64(time.Hour)
		}
	} else {
		fmt.Println("Invalid expiretype", expiretype)
		return
	}

	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	params := pty.ReqPri2Pri{
		Sender:     from,
		Pubkeypair: pubkeypair,
		Amount:     amountInt64,
		Mixin:      mixcount,
		Note:       note,
		Tokenname:  types.BTY,
		Expire:     expire,
	}

	var res rpctypes.ReplyHash
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.MakeTxPrivacy2privacy", params, &res)
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

	cmd.Flags().StringP("note", "n", "", "transfer note")
	cmd.Flags().Int32P("mixcount", "m", defMixCount, "transfer note")
	cmd.Flags().Int64P("expire", "", 0, "transfer expire, default one hour")
	cmd.Flags().IntP("expiretype", "", 1, "0: height  1: time default is 1")

}

func privacy2Public(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	mixcount, _ := cmd.Flags().GetInt32("mixcount")
	note, _ := cmd.Flags().GetString("note")
	expire, _ := cmd.Flags().GetInt64("expire")
	expiretype, _ := cmd.Flags().GetInt("expiretype")
	if expiretype == 0 {
		if expire <= 0 {
			fmt.Println("Invalid expire. expire must large than 0 in expiretype==0, expire", expire)
			return
		}
	} else if expiretype == 1 {
		if expire <= 0 {
			expire = int64(time.Hour)
		}
	} else {
		fmt.Println("Invalid expiretype", expiretype)
		return
	}

	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	params := pty.ReqPri2Pub{
		Sender:    from,
		Receiver:  to,
		Amount:    amountInt64,
		Note:      note,
		Tokenname: types.BTY,
		Mixin:     mixcount,
		Expire:    expire,
	}

	var res rpctypes.ReplyHash
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.MakeTxPrivacy2public", params, &res)
	ctx.Run()
}

func ShowPrivacyAccountSpendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "showpas",
		Short: "Show privacy account spend command",
		Run:   showPrivacyAccountSpend,
	}
	showPrivacyAccountSpendFlag(cmd)
	return cmd
}

func showPrivacyAccountSpendFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")
}

func showPrivacyAccountSpend(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")

	params := pty.ReqPrivBal4AddrToken{
		Addr:  addr,
		Token: types.BTY,
	}

	var res pty.UTXOHaveTxHashs
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.ShowPrivacyAccountSpend", params, &res)
	ctx.SetResultCb(parseShowPrivacyAccountSpendRes)
	ctx.Run()
}

func parseShowPrivacyAccountSpendRes(arg interface{}) (interface{}, error) {
	total := float64(0)
	res := arg.(*pty.UTXOHaveTxHashs)
	rets := make([]*PrivacyAccountSpendResult, 0)
	for _, utxo := range res.UtxoHaveTxHashs {
		amount := float64(utxo.Amount) / float64(types.Coin)
		total += amount

		var isSave bool
		for _, ret := range rets {
			if utxo.TxHash == ret.Txhash {
				result := &PrivacyAccountResult{
					Txhash:   common.ToHex(utxo.UtxoBasic.UtxoGlobalIndex.Txhash),
					OutIndex: utxo.UtxoBasic.UtxoGlobalIndex.Outindex,
					Amount:   strconv.FormatFloat(amount, 'f', 4, 64),
				}
				ret.Res = append(ret.Res, result)
				isSave = true
				break
			}
		}

		if !isSave {
			result := &PrivacyAccountResult{
				//Height:   utxo.UtxoBasic.UtxoGlobalIndex.Height,
				//TxIndex:  utxo.UtxoBasic.UtxoGlobalIndex.Txindex,
				Txhash:   common.ToHex(utxo.UtxoBasic.UtxoGlobalIndex.Txhash),
				OutIndex: utxo.UtxoBasic.UtxoGlobalIndex.Outindex,
				Amount:   strconv.FormatFloat(amount, 'f', 4, 64),
			}
			var SpendResult PrivacyAccountSpendResult
			SpendResult.Txhash = utxo.TxHash
			SpendResult.Res = append(SpendResult.Res, result)
			rets = append(rets, &SpendResult)
		}
	}

	fmt.Println(fmt.Sprintf("Total Privacy spend amount is %s", strconv.FormatFloat(total, 'f', 4, 64)))

	return rets, nil
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

	reqPrivacyToken := pty.ReqPrivacyToken{Token: types.BTY}
	var params types.Query4Cli
	params.Execer = pty.PrivacyX
	params.FuncName = "ShowAmountsOfUTXO"
	params.Payload = reqPrivacyToken

	var res pty.ReplyPrivacyAmounts
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseShowAmountOfUTXORes)
	ctx.Run()
}

func parseShowAmountOfUTXORes(arg interface{}) (interface{}, error) {
	res := arg.(*pty.ReplyPrivacyAmounts)
	for _, amount := range res.AmountDetail {
		amount.Amount = amount.Amount / types.Coin
	}
	return res, nil
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

	reqPrivacyToken := pty.ReqPrivacyToken{
		Token:  types.BTY,
		Amount: amountInt64,
	}
	var params types.Query4Cli
	params.Execer = pty.PrivacyX
	params.FuncName = "ShowUTXOs4SpecifiedAmount"
	params.Payload = reqPrivacyToken

	var res pty.ReplyUTXOsOfAmount
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseShowUTXOs4SpecifiedAmountRes)
	ctx.Run()
}

func parseShowUTXOs4SpecifiedAmountRes(arg interface{}) (interface{}, error) {
	res := arg.(*pty.ReplyUTXOsOfAmount)
	ret := make([]*PrivacyAccountResult, 0)
	for _, item := range res.LocalUTXOItems {
		result := &PrivacyAccountResult{
			Txhash:        common.ToHex(item.Txhash),
			OutIndex:      item.Outindex,
			OnetimePubKey: common.Bytes2Hex(item.Onetimepubkey),
		}
		ret = append(ret, result)
	}

	return ret, nil
}

func CreateUTXOsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "createutxos",
		Short: "Create specified count UTXOs with specified amount",
		Run:   createUTXOs,
	}
	createUTXOsFlag(cmd)
	return cmd
}

func createUTXOsFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("from", "f", "", "from account address")
	cmd.MarkFlagRequired("from")
	cmd.Flags().StringP("pubkeypair", "p", "", "to view spend public key pair")
	cmd.MarkFlagRequired("pubkeypair")
	cmd.Flags().Float64P("amount", "a", 0, "amount")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().Int32P("count", "c", 0, "mix count, default 0")
	cmd.Flags().StringP("note", "n", "", "transfer note")
	cmd.Flags().Int64P("expire", "", int64(time.Hour), "transfer expire, default one hour")
}

func createUTXOs(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	from, _ := cmd.Flags().GetString("from")
	pubkeypair, _ := cmd.Flags().GetString("pubkeypair")
	note, _ := cmd.Flags().GetString("note")
	count, _ := cmd.Flags().GetInt32("count")
	amount, _ := cmd.Flags().GetFloat64("amount")
	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4
	expire, _ := cmd.Flags().GetInt64("expire")
	if expire <= 0 {
		expire = int64(time.Hour)
	}

	params := &pty.ReqCreateUTXOs{
		Tokenname:  types.BTY,
		Sender:     from,
		Pubkeypair: pubkeypair,
		Amount:     amountInt64,
		Count:      count,
		Note:       note,
		Expire:     expire,
	}

	var res rpctypes.ReplyHash
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.CreateUTXOs", params, &res)
	ctx.Run()
}

// ShowPrivacyAccountInfoCmd
func ShowPrivacyAccountInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "showpai",
		Short: "Show privacy account information",
		Run:   showPrivacyAccountInfo,
	}
	showPrivacyAccountInfoFlag(cmd)
	return cmd
}

func showPrivacyAccountInfoFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")

	cmd.Flags().StringP("token", "t", types.BTY, "coins token, BTY supported.")
	cmd.Flags().Int32P("displaymode", "d", 0, "display mode.(0: display collect. 1:display avaliable detail. 2:display frozen detail. 3:display all")
}

func showPrivacyAccountInfo(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	token, _ := cmd.Flags().GetString("token")
	mode, _ := cmd.Flags().GetInt32("displaymode")
	if mode < 0 || mode > 3 {
		fmt.Println("display mode only support 0-3")
		return
	}

	params := pty.ReqPPrivacyAccount{
		Addr:        addr,
		Token:       token,
		Displaymode: mode,
	}

	var res pty.ReplyPrivacyAccount
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.ShowPrivacyAccountInfo", params, &res)
	ctx.SetResultCb(parseshowPrivacyAccountInfo)
	ctx.Run()
}

func parseshowPrivacyAccountInfo(arg interface{}) (interface{}, error) {
	total := float64(0)
	totalFrozen := float64(0)
	res := arg.(*pty.ReplyPrivacyAccount)

	var availableAmount, frozenAmount, totalAmount string

	utxos := make([]*PrivacyAccountResult, 0)
	for _, utxo := range res.Utxos.Utxos {
		amount := float64(utxo.Amount) / float64(types.Coin)
		total += amount

		if res.Displaymode == 1 || res.Displaymode == 3 {
			result := &PrivacyAccountResult{
				Txhash:   common.ToHex(utxo.UtxoBasic.UtxoGlobalIndex.Txhash),
				OutIndex: utxo.UtxoBasic.UtxoGlobalIndex.Outindex,
				Amount:   strconv.FormatFloat(amount, 'f', 4, 64),
			}
			utxos = append(utxos, result)
		}
	}
	availableAmount = strconv.FormatFloat(total, 'f', 4, 64)

	ftxos := make([]*PrivacyAccountResult, 0)
	for _, utxo := range res.Ftxos.Utxos {
		amount := float64(utxo.Amount) / float64(types.Coin)
		totalFrozen += amount

		if res.Displaymode == 2 || res.Displaymode == 3 {
			result := &PrivacyAccountResult{
				Txhash:   common.ToHex(utxo.UtxoBasic.UtxoGlobalIndex.Txhash),
				OutIndex: utxo.UtxoBasic.UtxoGlobalIndex.Outindex,
				Amount:   strconv.FormatFloat(amount, 'f', 4, 64),
			}
			ftxos = append(ftxos, result)
		}
	}
	frozenAmount = strconv.FormatFloat(totalFrozen, 'f', 4, 64)
	totalAmount = strconv.FormatFloat(total+totalFrozen, 'f', 4, 64)

	ret := &PrivacyAccountInfoResult{
		AvailableDetail: utxos,
		FrozenDetail:    ftxos,
		AvailableAmount: availableAmount,
		FrozenAmount:    frozenAmount,
		TotalAmount:     totalAmount,
	}
	return ret, nil
}

func ListPrivacyTxsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list_txs",
		Short: "List privacy transactions in wallet",
		Run:   listPrivacyTxsFlags,
	}
	addListPrivacyTxsFlags(cmd)
	return cmd
}

func addListPrivacyTxsFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")
	//
	cmd.Flags().Int32P("sendrecv", "", 0, "send or recv flag (0: send, 1: recv), default 0")
	cmd.Flags().Int32P("count", "c", 10, "number of transactions, default 10")
	cmd.Flags().StringP("token", "", types.BTY, "token name.(BTY supported)")
	cmd.Flags().Int32P("direction", "d", 1, "query direction (0: pre page, 1: next page), valid with seedtxhash param")
	cmd.Flags().StringP("seedtxhash", "", "", "seed trasnaction hash")
}

func listPrivacyTxsFlags(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	count, _ := cmd.Flags().GetInt32("count")
	direction, _ := cmd.Flags().GetInt32("direction")
	addr, _ := cmd.Flags().GetString("addr")
	sendRecvFlag, _ := cmd.Flags().GetInt32("sendrecv")
	tokenname, _ := cmd.Flags().GetString("token")
	seedtxhash, _ := cmd.Flags().GetString("seedtxhash")
	params := pty.ReqPrivacyTransactionList{
		Tokenname:    tokenname,
		SendRecvFlag: sendRecvFlag,
		Direction:    direction,
		Count:        count,
		Address:      addr,
		Seedtxhash:   []byte(seedtxhash),
	}
	var res rpctypes.WalletTxDetails
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.PrivacyTxList", params, &res)
	ctx.SetResultCb(parseWalletTxListRes)
	ctx.Run()
}

func parseWalletTxListRes(arg interface{}) (interface{}, error) {
	res := arg.(*rpctypes.WalletTxDetails)
	var result cmdtypes.WalletTxDetailsResult
	for _, v := range res.TxDetails {
		amountResult := strconv.FormatFloat(float64(v.Amount)/float64(types.Coin), 'f', 4, 64)
		wtxd := &cmdtypes.WalletTxDetailResult{
			Tx:         cmdtypes.DecodeTransaction(v.Tx),
			Receipt:    v.Receipt,
			Height:     v.Height,
			Index:      v.Index,
			Blocktime:  v.BlockTime,
			Amount:     amountResult,
			Fromaddr:   v.FromAddr,
			Txhash:     v.TxHash,
			ActionName: v.ActionName,
		}
		result.TxDetails = append(result.TxDetails, wtxd)
	}
	return result, nil
}

func RescanUtxosOptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rescanOpt",
		Short: "rescan Utxos in wallet and query rescan utxos status",
		Run:   RescanUtxosOpt,
	}
	RescanUtxosOptFlags(cmd)
	return cmd
}

func RescanUtxosOptFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "privacy rescanOpt -a [all-addr0-addr1] (all indicate all wallet address)")
	cmd.MarkFlagRequired("addr")
	//
	cmd.Flags().Int32P("flag", "f", 0, "Rescan or query rescan flag (0: Rescan, 1: query rescan)")
}

func RescanUtxosOpt(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	address, _ := cmd.Flags().GetString("addr")
	flag, _ := cmd.Flags().GetInt32("flag")

	var params pty.ReqRescanUtxos

	params.Flag = flag
	if "all" != address {
		if len(address) > 0 {
			addrs := strings.Split(address, "-")
			params.Addrs = append(params.Addrs, addrs...)
		}
	}

	var res pty.RepRescanUtxos
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.RescanUtxos", params, &res)
	ctx.SetResultCb(parseRescanUtxosOpt)
	ctx.Run()
}

func parseRescanUtxosOpt(arg interface{}) (interface{}, error) {
	res := arg.(*pty.RepRescanUtxos)
	var result showRescanResults
	if 0 == res.Flag {
		str := "start rescan UTXO"
		return str, nil
	} else {
		for _, v := range res.RepRescanResults {
			str, ok := pty.RescanFlagMapint2string[v.Flag]
			if ok {
				showRescanResult := &ShowRescanResult{
					Addr:       v.Addr,
					FlagString: str,
				}
				result.RescanResults = append(result.RescanResults, showRescanResult)
			}
		}
		return &result, nil
	}
}

func EnablePrivacyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "enable privacy address in wallet",
		Run:   EnablePrivacy,
	}
	EnablePrivacyFlags(cmd)
	return cmd
}

func EnablePrivacyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "privacy enable -a [all-addr0-addr1] (all indicate enable all wallet address)")
	cmd.MarkFlagRequired("addr")
}

func EnablePrivacy(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	address, _ := cmd.Flags().GetString("addr")

	var params pty.ReqEnablePrivacy

	if "all" != address {
		if len(address) > 0 {
			addrs := strings.Split(address, "-")
			params.Addrs = append(params.Addrs, addrs...)
		}
	}

	var res pty.RepEnablePrivacy
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.EnablePrivacy", params, &res)
	ctx.SetResultCb(parseEnablePrivacy)
	ctx.Run()
}

func parseEnablePrivacy(arg interface{}) (interface{}, error) {
	res := arg.(*pty.RepEnablePrivacy)

	var result ShowEnablePrivacy
	for _, v := range res.Results {
		showPriAddrResult := &ShowPriAddrResult{
			Addr: v.Addr,
			IsOK: v.IsOK,
			Msg:  v.Msg,
		}
		result.Results = append(result.Results, showPriAddrResult)
	}
	return &result, nil
}
