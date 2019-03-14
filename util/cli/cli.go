// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/system/dapp/commands"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   types.GetTitle() + "-cli",
	Short: types.GetTitle() + " client tools",
}

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send transaction in one step",
	Run:   func(cmd *cobra.Command, args []string) {},
}

var closeCmd = &cobra.Command{
	Use:   "close",
	Short: "Close " + types.GetTitle(),
	Run: func(cmd *cobra.Command, args []string) {
		rpcLaddr, err := cmd.Flags().GetString("rpc_laddr")
		if err != nil {
			panic(err)
		}
		//		rpc, _ := jsonrpc.NewJSONClient(rpcLaddr)
		//		rpc.Call("Chain33.CloseQueue", nil, nil)
		var res rpctypes.Reply
		ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.CloseQueue", nil, &res)
		ctx.Run()
	},
}

func init() {
	rootCmd.AddCommand(
		commands.CertCmd(),
		commands.AccountCmd(),
		commands.BlockCmd(),
		commands.BTYCmd(),
		commands.CoinsCmd(),
		commands.ExecCmd(),
		commands.MempoolCmd(),
		commands.NetCmd(),
		commands.SeedCmd(),
		commands.StatCmd(),
		commands.TxCmd(),
		commands.WalletCmd(),
		commands.VersionCmd(),
		sendCmd,
		closeCmd,
		commands.AssetCmd(),
	)
}

func testTLS(RPCAddr string) string {
	rpcaddr := RPCAddr
	if strings.HasPrefix(rpcaddr, "https://") {
		return RPCAddr
	}
	if !strings.HasPrefix(rpcaddr, "http://") {
		return RPCAddr
	}
	//test tls ok
	if rpcaddr[len(rpcaddr)-1] != '/' {
		rpcaddr += "/"
	}
	rpcaddr += "test"
	resp, err := http.Get(rpcaddr)
	if err != nil {
		return "https://" + RPCAddr[7:]
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return RPCAddr
	}
	return "https://" + RPCAddr[7:]
}

//Run :
func Run(RPCAddr, ParaName string) {
	//test tls is enable
	RPCAddr = testTLS(RPCAddr)
	pluginmgr.AddCmd(rootCmd)
	log.SetLogLevel("error")
	types.S("RPCAddr", RPCAddr)
	types.S("ParaName", ParaName)
	rootCmd.PersistentFlags().String("rpc_laddr", types.GStr("RPCAddr"), "http url")
	rootCmd.PersistentFlags().String("paraName", types.GStr("ParaName"), "parachain")
	if len(os.Args) > 1 {
		if os.Args[1] == "send" {
			commands.OneStepSend(os.Args)
			return
		}
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
