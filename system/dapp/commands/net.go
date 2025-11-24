// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
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
		DialCmd(),
		CloseCmd(),
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
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetPeerInfo", &req, &res)
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
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetNetInfo", &req, &res)
	ctx.Run()
}

// NetProtocolsCmd get all prototols netinfo
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

// BlacklistCmd get all prototols netinfo
func BlacklistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blacklist",
		Short: "block peer connect p2p network",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		Add(),
		Del(),
		Show(),
	)
	return cmd
}

// Add add peer to blacklist
func Add() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add peer or IP to blacklist",
		Run:   addblacklist,
	}
	cmd.Flags().StringP("time", "t", "", "lifetime such as: 1hour,1min,1second")
	addBlackFlags(cmd)
	return cmd
}

// Del del peer from blacklist
func Del() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del",
		Short: "delete peer from blacklist",
		Run:   delblacklist,
	}
	addBlackFlags(cmd)
	return cmd
}

// Show print all peers from blacklist
func Show() *cobra.Command {
	//黑名单列表打印出来
	cmd := &cobra.Command{
		Use:   "show",
		Short: "show all peer from blacklist",
		Run:   showblacklist,
	}
	return cmd

}

func addBlackFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "", "", "add peer address to blacklist ,used only gossip network")
	cmd.Flags().StringP("pid", "", "", "add  peer id to blacklist")
}

func addblacklist(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	peerName, _ := cmd.Flags().GetString("pid")
	PeerAddr, _ := cmd.Flags().GetString("addr")
	lifetime, _ := cmd.Flags().GetString("time")

	var res interface{}
	var peer types.BlackPeer
	peer.Lifetime = lifetime
	if PeerAddr != "" {
		peer.PeerAddr = PeerAddr
	} else if peerName != "" {
		peer.PeerName = peerName
	} else {
		cmd.Usage()
		return
	}

	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.AddBlacklist", &peer, &res)
	ctx.Run()
}

func delblacklist(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	peerName, _ := cmd.Flags().GetString("pid")
	PeerAddr, _ := cmd.Flags().GetString("addr")

	var res interface{}
	var peer types.BlackPeer
	if PeerAddr != "" {
		peer.PeerAddr = PeerAddr
	} else if peerName != "" {
		peer.PeerName = peerName
	} else {
		cmd.Usage()
		return
	}

	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.DelBlacklist", &peer, &res)
	ctx.Run()

}

func showblacklist(cmd *cobra.Command, args []string) {
	var res = new([]*types.BlackInfo)
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.ShowBlacklist", nil, &res)
	ctx.Run()

}

// DialCmd dial the specified node
func DialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dial",
		Short: "dial the specified node",
		Run:   dialPeer,
	}

	cmd.Flags().StringP("peer", "p", "", "peer addr,format: /ip4/ip/tcp/port/p2p/pid")
	cmd.Flags().BoolP("seed", "", false, "set peer to seed")
	return cmd
}

func dialPeer(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	peerAddr, _ := cmd.Flags().GetString("peer")
	isseed, _ := cmd.Flags().GetBool("seed")
	if peerAddr == "" {
		cmd.Usage()
		return
	}
	var setpeer types.SetPeer
	var res interface{}
	setpeer.Seed = isseed
	setpeer.PeerAddr = peerAddr
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.DialPeer", &setpeer, &res)
	ctx.Run()
}

// CloseCmd close the specified peer
func CloseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close",
		Short: "close the specified peer",
		Run:   closePeer,
	}
	cmd.Flags().StringP("pid", "", "", "peer id,support dht network")
	return cmd
}

func closePeer(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	pid, _ := cmd.Flags().GetString("pid")
	if pid == "" {
		cmd.Usage()
		return
	}
	var setpeer types.SetPeer
	var res interface{}
	setpeer.Pid = pid
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.ClosePeer", &setpeer, &res)
	ctx.Run()
}
