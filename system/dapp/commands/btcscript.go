package commands

import (
	"fmt"
	"os"

	"github.com/33cn/chain33/rpc/jsonclient"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
)

// BtcScriptCmd bitcoin script command
func BtcScriptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "btcscript",
		Short: "construct bitcoin script",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		getWalletRecoveryAddrCmd(),
		signWalletRecoveryTxCmd(),
	)

	return cmd
}

// walletRecoveryAddrCmd create commit delay transaction
func getWalletRecoveryAddrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recovaddr",
		Short: "get recover address",
		Run:   getWalletRecoveryAddr,
	}
	getWalletRecoveryFlags(cmd)
	return cmd
}

func getWalletRecoveryFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("controlPub", "c", "", "control address public key(hex format)")
	cmd.MarkFlagRequired("controlPub")

	cmd.Flags().StringP("recoverPub", "r", "", "recovery address public key(hex format)")
	cmd.MarkFlagRequired("recoverPub")

	cmd.Flags().Int64P("delaytime", "t", 100, "relative delay time(seconds)")
	cmd.MarkFlagRequired("delaytime")
}

func getWalletRecoveryAddr(cmd *cobra.Command, args []string) {
	ctrPub, _ := cmd.Flags().GetString("controlPub")
	recovPub, _ := cmd.Flags().GetString("recoverPub")
	delayTime, _ := cmd.Flags().GetInt64("delaytime")

	if ctrPub == "" || recovPub == "" {
		fmt.Fprintf(os.Stderr, "invalid control/recovery pubKey\n")
		return
	}

	if delayTime < 1 {
		fmt.Fprintf(os.Stderr, "negetive delay time param\n")
		return
	}

	req := &types.ReqGetWalletRecoverAddr{
		CtrPubKey:         ctrPub,
		RecoverPubKey:     recovPub,
		RelativeDelayTime: delayTime,
	}
	var res string
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcAddr, "Chain33.GetWalletRecoverAddress", req, &res)
	ctx.RunWithoutMarshal()
}

// signWalletRecoveryTxCmd sign wallet recover tx
func signWalletRecoveryTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signrecov",
		Short: "sign wallet recovery tx",
		Run:   signWalletRecoveryTx,
	}
	signWalletRecoveryTxFlags(cmd)
	return cmd
}

func signWalletRecoveryTxFlags(cmd *cobra.Command) {

	cmd.Flags().StringP("privkey", "k", "", "sign private key(hex format)")
	cmd.Flags().StringP("addr", "a", "", "sign address(must exist in wallet)")

	cmd.Flags().StringP("txdata", "d", "", "tx data for sign(hex format)")
	cmd.MarkFlagRequired("txdata")

	cmd.Flags().StringP("controlPub", "c", "", "control address public key(hex format)")
	cmd.MarkFlagRequired("controlPub")

	cmd.Flags().StringP("recoverPub", "r", "", "recovery address public key(hex format)")
	cmd.MarkFlagRequired("recoverPub")

	cmd.Flags().Int64P("delaytime", "t", 100, "relative delay time(seconds)")
	cmd.MarkFlagRequired("delaytime")
}

func signWalletRecoveryTx(cmd *cobra.Command, args []string) {

	privKey, _ := cmd.Flags().GetString("privkey")
	addr, _ := cmd.Flags().GetString("addr")
	txdata, _ := cmd.Flags().GetString("txdata")
	ctrPub, _ := cmd.Flags().GetString("controlPub")
	recovPub, _ := cmd.Flags().GetString("recoverPub")
	delayTime, _ := cmd.Flags().GetInt64("delaytime")

	if privKey == "" && addr == "" {
		fmt.Fprintf(os.Stderr, "sign private key or address must be provided\n")
		return
	}

	if ctrPub == "" || recovPub == "" {
		fmt.Fprintf(os.Stderr, "invalid control/recovery pubKey\n")
		return
	}

	if delayTime < 0 {
		fmt.Fprintf(os.Stderr, "negetive delay height param\n")
		return
	}

	walletRecov := &types.ReqGetWalletRecoverAddr{
		CtrPubKey:         ctrPub,
		RecoverPubKey:     recovPub,
		RelativeDelayTime: delayTime,
	}

	req := &types.ReqSignWalletRecoverTx{
		SignAddr:           addr,
		PrivKey:            privKey,
		RawTx:              txdata,
		WalletRecoverParam: walletRecov,
	}

	var res string
	rpcAddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcAddr, "Chain33.SignWalletRecoverTx", req, &res)
	ctx.RunWithoutMarshal()
}
