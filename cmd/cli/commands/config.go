package commands

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet"
)

func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		ConfigTxCmd(),
		QueryConfigCmd(),
	)

	return cmd
}

// config transaction
func ConfigTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config_tx",
		Short: "Set transaction fee",
		Run:   configTx,
	}
	addConfigTxFlags(cmd)
	return cmd
}

func addConfigTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("key", "k", "", "key string")
	cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("operation", "o", "", "adding or deletion operation")
	cmd.MarkFlagRequired("operation")

	cmd.Flags().StringP("value", "v", "", "operating object")
	cmd.MarkFlagRequired("value")

	cmd.Flags().StringP("priv_key", "p", "", "private key")
	cmd.MarkFlagRequired("priv_key")
}

func configTx(cmd *cobra.Command, args []string) {
	key, _ := cmd.Flags().GetString("key")
	op, _ := cmd.Flags().GetString("operation")
	opAddr, _ := cmd.Flags().GetString("value")
	priv, _ := cmd.Flags().GetString("priv_key")

	c, _ := crypto.New(types.GetSignatureTypeName(wallet.SignType))
	a, _ := common.FromHex(priv)
	privKey, _ := c.PrivKeyFromBytes(a)
	originAddr := account.PubKeyToAddress(privKey.PubKey().Bytes()).String()

	v := &types.ModifyConfig{Key: key, Op: op, Value: opAddr, Addr: originAddr}
	modify := &types.ManageAction{
		Ty:    types.ManageActionModifyConfig,
		Value: &types.ManageAction_Modify{Modify: v},
	}
	tx := &types.Transaction{Execer: []byte("manage"), Payload: types.Encode(modify)}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()

	var err error
	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	tx.Fee += types.MinFee
	tx.Sign(int32(wallet.SignType), privKey)
	txHex := types.Encode(tx)
	fmt.Println(hex.EncodeToString(txHex))
}

// query config
func QueryConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query_config",
		Short: "Query config item",
		Run:   queryConfig,
	}
	addQueryConfigFlags(cmd)
	return cmd
}

func addQueryConfigFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("key", "k", "", "key string")
	cmd.MarkFlagRequired("key")
}

func queryConfig(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	key, _ := cmd.Flags().GetString("key")
	req := &types.ReqString{
		Data: key,
	}
	var params jsonrpc.Query4Cli
	params.Execer = "manage"
	params.FuncName = "GetConfigItem"
	params.Payload = req

	var res types.ReplyConfig
	ctx := NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.Run()
}
