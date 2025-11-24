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
	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/system/dapp/commands"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
)

// Run :
func Run(RPCAddr, ParaName, name string) {
	// cli 命令只打印错误级别到控制台
	log.SetLogLevel("error")
	configPath := ""
	for i, arg := range os.Args[:] {
		if arg == "--conf" && i+1 <= len(os.Args)-1 { // --conf chain33.toml 可以配置读入cli配置文件路径
			configPath = os.Args[i+1]
			break
		}
		if strings.HasPrefix(arg, "--conf=") { // --conf="chain33.toml"
			configPath = strings.TrimPrefix(arg, "--conf=")
			break
		}
	}
	if configPath == "" {
		if name == "" {
			configPath = "chain33.toml"
		} else {
			configPath = name + ".toml"
		}
	}

	exist, _ := pathExists(configPath)
	var chain33Cfg *types.Chain33Config
	if exist {
		chain33Cfg = types.NewChain33Config(types.ReadFile(configPath))
	} else {
		cfgstring := types.GetDefaultCfgstring()
		if ParaName != "" {
			cfgstring = strings.Replace(cfgstring, "Title=\"local\"", fmt.Sprintf("Title=\"%s\"", ParaName), 1)
			cfgstring = strings.Replace(cfgstring, "FixTime=false", "CoinSymbol=\"para\"", 1)
		}
		chain33Cfg = types.NewChain33Config(cfgstring)
	}

	types.SetCliSysParam(chain33Cfg.GetTitle(), chain33Cfg)

	rootCmd := &cobra.Command{
		Use:     chain33Cfg.GetTitle() + "-cli",
		Short:   chain33Cfg.GetTitle() + " client tools",
		Version: fmt.Sprintf("%s %s", version.GetVersion(), version.BuildTime),
	}

	closeCmd := &cobra.Command{
		Use:   "close",
		Short: "Close " + chain33Cfg.GetTitle(),
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

	rootCmd.AddCommand(
		commands.CertCmd(),
		commands.AccountCmd(),
		commands.BlockCmd(),
		commands.CoinsCmd(),
		commands.ExecCmd(),
		commands.MempoolCmd(),
		commands.NetCmd(),
		commands.SeedCmd(),
		commands.StatCmd(),
		commands.TxCmd(),
		commands.WalletCmd(),
		commands.VersionCmd(),
		commands.SystemCmd(),
		commands.OneStepSendCmd(),
		commands.OneStepSendCertTxCmd(),
		commands.BlacklistCmd(),
		closeCmd,
		commands.AssetCmd(),
		commands.NoneCmd(),
		commands.BtcScriptCmd(),
	)

	//test tls is enable
	RPCAddr = testTLS(RPCAddr)
	pluginmgr.AddCmd(rootCmd)
	log.SetLogLevel("error")
	chain33Cfg.S("RPCAddr", RPCAddr)
	chain33Cfg.S("ParaName", ParaName)
	rootCmd.PersistentFlags().String("rpc_laddr", chain33Cfg.GStr("RPCAddr"), "http url")
	rootCmd.PersistentFlags().String("paraName", chain33Cfg.GStr("ParaName"), "parachain")
	rootCmd.PersistentFlags().String("title", chain33Cfg.GetTitle(), "get title name")
	rootCmd.PersistentFlags().MarkHidden("title")
	rootCmd.PersistentFlags().String("conf", "", "cli config")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func testTLS(RPCAddr string) string {
	rpcaddr := RPCAddr
	if !strings.HasPrefix(rpcaddr, "http://") {
		return RPCAddr
	}
	// if http://
	if rpcaddr[len(rpcaddr)-1] != '/' {
		rpcaddr += "/"
	}
	rpcaddr += "test"
	/* #nosec */
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

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
