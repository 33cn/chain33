package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"gitlab.33.cn/chain33/chain33/common"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/spf13/cobra"
)

func DecodeTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decode",
		Short: "Decode a hex format transaction",
		Run:   decodeTx,
	}
	addDecodeTxFlags(cmd)
	return cmd
}

func addDecodeTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("data", "d", "", "transaction content")
	cmd.MarkFlagRequired("data")
}

func decodeTx(cmd *cobra.Command, args []string) {
	data, _ := cmd.Flags().GetString("data")
	var tx types.Transaction
	bytes, err := common.FromHex(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = types.Decode(bytes, &tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	res, err := jsonrpc.DecodeTx(&tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	result, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(result))
}
