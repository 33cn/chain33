// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"

	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
)

// NetCmd net command
func NetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "net",
		Short: "Net operation",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		GetPeerInfoCmd(),
		IsClockSyncCmd(),
		IsSyncCmd(),
		GetNetInfoCmd(),
		GetFatalFailureCmd(),
		GetTimeStausCmd(),
		NetProtocolsCmd(),
	)

	return cmd
}

// GetPeerInfoCmd  get peers connected info
func GetPeerInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Get remote peer nodes",
		Run:   peerInfo,
	}
	cmd.Flags().StringP("type", "t", "", "p2p type, gossip or dht")
	return cmd
}

func peerInfo(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res rpctypes.PeerList
	p2pty, _ := cmd.Flags().GetString("type")
	req := types.P2PGetPeerReq{P2PType: p2pty}
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetPeerInfo", req, &res)
	ctx.Run()
}

// IsClockSyncCmd  get ntp clock sync status
func IsClockSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "is_clock_sync",
		Short: "Get ntp clock synchronization status",
		Run:   isClockSync,
	}
	return cmd
}

func isClockSync(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res bool
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.IsNtpClockSync", nil, &res)
	ctx.Run()
}

// IsSyncCmd get local db sync status
func IsSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "is_sync",
		Short: "Get blockchain synchronization status",
		Run:   isSync,
	}
	return cmd
}

func isSync(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res bool
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.IsSync", nil, &res)
	ctx.Run()
}

// GetNetInfoCmd get net info
func GetNetInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Get net information",
		Run:   netInfo,
	}
	cmd.Flags().StringP("type", "t", "", "p2p type, gossip or dht")
	return cmd
}

func netInfo(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	p2pty, _ := cmd.Flags().GetString("type")
	req := types.P2PGetNetInfoReq{P2PType: p2pty}
	var res rpctypes.NodeNetinfo
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetNetInfo", req, &res)
	ctx.Run()
}

//NetProtocolsCmd get all prototols netinfo
func NetProtocolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protocols",
		Short: "Get netprotocols information",
		Run:   netProtocols,
	}
	cmd.Flags().StringP("type", "t", "", "p2p type, gossip or dht")
	return cmd
}

func netProtocols(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res types.NetProtocolInfos
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.NetProtocols", &types.ReqNil{}, &res)
	ctx.Run()
}

// GetFatalFailureCmd get FatalFailure
func GetFatalFailureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fault",
		Short: "Get system fault",
		Run:   fatalFailure,
	}
	return cmd
}

func fatalFailure(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res int64
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetFatalFailure", nil, &res)
	ctx.Run()
}

// GetTimeStausCmd get time status
func GetTimeStausCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time",
		Short: "Get time status",
		Run:   timestatus,
	}
	return cmd
}

func timestatus(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res rpctypes.TimeStatus
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetTimeStatus", nil, &res)
	ctx.Run()
}
