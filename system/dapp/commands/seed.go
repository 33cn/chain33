// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"fmt"

	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
)

// SeedCmd seed command
func SeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		GenSeedCmd(),
		GetSeedCmd(),
		SaveSeedCmd(),
	)

	return cmd
}

// GenSeedCmd generate seed
func GenSeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate seed",
		Run:   genSeed,
	}
	addGenSeedFlags(cmd)
	return cmd
}

func addGenSeedFlags(cmd *cobra.Command) {
	cmd.Flags().Int32P("lang", "l", 0, "seed language(0:English, 1:简体中文)")
	cmd.MarkFlagRequired("lang")
}

func genSeed(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	lang, _ := cmd.Flags().GetInt32("lang")
	params := types.GenSeedLang{
		Lang: lang,
	}
	var res types.ReplySeed
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GenSeed", params, &res)
	_, err := ctx.RunResult()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(res.Seed)
}

// GetSeedCmd get seed
func GetSeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get seed by password",
		Run:   getSeed,
	}
	addGetSeedFlags(cmd)
	return cmd
}

func addGetSeedFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("pwd", "p", "", "password used to fetch seed")
	cmd.MarkFlagRequired("pwd")
}

func getSeed(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	pwd, _ := cmd.Flags().GetString("pwd")
	params := types.GetSeedByPw{
		Passwd: pwd,
	}
	var res types.ReplySeed
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetSeed", params, &res)
	ctx.Run()
}

// SaveSeedCmd save seed
func SaveSeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save seed and encrypt with passwd",
		Run:   saveSeed,
	}
	addSaveSeedFlags(cmd)
	return cmd
}

func addSaveSeedFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("seed", "s", "", "15 seed characters separated by space")
	cmd.MarkFlagRequired("seed")

	cmd.Flags().StringP("pwd", "p", "", "password used to encrypt seed")
	cmd.MarkFlagRequired("pwd")
}

func saveSeed(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	seed, _ := cmd.Flags().GetString("seed")
	pwd, _ := cmd.Flags().GetString("pwd")
	params := types.SaveSeedByPw{
		Seed:   seed,
		Passwd: pwd,
	}
	var res rpctypes.Reply
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.SaveSeed", params, &res)
	ctx.Run()
}
