package commands

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func TokenAssetsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "assets",
		// TODO
		Short: "查询地址下的token/trace合约下的token资产",
		Run:   tokenAssets,
	}
	addTokenAssetsFlags(cmd)
	return cmd
}

func addTokenAssetsFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("exec", "e", "", "execer name")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")
}

func tokenAssets(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	execer, _ := cmd.Flags().GetString("exec")
	req := types.ReqAccountTokenAssets{
		Address: addr,
		Execer:  execer,
	}

	var params jsonrpc.Query
	params.Execer = "token"
	params.FuncName = "GetAccountTokenAssets"
	params.Payload = hex.EncodeToString(types.Encode(&req))
	rpc, err := jsonrpc.NewJsonClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res *types.ReplyAccountTokenAssets
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for _, result := range res.TokenAssets {
		balanceResult := strconv.FormatFloat(float64(result.Account.Balance)/float64(types.TokenPrecision), 'f', 4, 64)
		frozenResult := strconv.FormatFloat(float64(result.Account.Frozen)/float64(types.TokenPrecision), 'f', 4, 64)
		result := &TokenAccountResult{
			Token:    result.Symbol,
			Addr:     result.Account.Addr,
			Currency: result.Account.Currency,
			Balance:  balanceResult,
			Frozen:   frozenResult,
		}

		data, err := json.MarshalIndent(result, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		fmt.Println(string(data))
	}
}
