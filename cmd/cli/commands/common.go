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
)

func CommonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "common",
		Short: "Common operation or status about chain33",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		LockCmd(),
		UnlockCmd(),
		IsSyncCmd(),
		AutoMineCmd(),
		PeerInfoCmd(),
		BindMinerCmd(),
		IsClockSync(),
	)

	return cmd
}

// lock
func LockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Lock wallet",
		Run:   lock,
	}
	return cmd
}

func lock(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res jsonrpc.Reply
	ctx := NewRPCCtx(rpcLaddr, "Chain33.Lock", nil, &res)
	ctx.Run()
}

// unlock
func UnlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock wallet",
		Run:   unLock,
	}
	addUnlockFlags(cmd)
	return cmd
}

func addUnlockFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("pwd", "p", "", "password needed to unlock")
	cmd.MarkFlagRequired("pwd")

	cmd.Flags().Int64P("time_out", "t", 0, "time out for unlock operation(0 for unlimited)")
	cmd.Flags().BoolP("scope", "s", false, "unlock scope(true:whole wallet, false:only ticket)")
}

func unLock(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	pwd, _ := cmd.Flags().GetString("pwd")
	timeOut, _ := cmd.Flags().GetInt64("time_out")
	walletOrTicket, _ := cmd.Flags().GetBool("scope")
	params := types.WalletUnLock{
		Passwd:         pwd,
		Timeout:        timeOut,
		WalletOrTicket: walletOrTicket,
	}
	var res jsonrpc.Reply
	ctx := NewRPCCtx(rpcLaddr, "Chain33.UnLock", params, &res)
	ctx.Run()
}

// issync
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
	ctx := NewRPCCtx(rpcLaddr, "Chain33.IsSync", nil, &res)
	ctx.Run()
}

// auto_mine
func AutoMineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto_mine",
		Short: "Set auto mine on/off",
		Run:   autoMine,
	}
	addAutoMineFlags(cmd)
	return cmd
}

func addAutoMineFlags(cmd *cobra.Command) {
	cmd.Flags().Int32P("flag", "f", 0, `auto mine(0: off, 1: on)`)
	cmd.MarkFlagRequired("flag")
}

func autoMine(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	flag, _ := cmd.Flags().GetInt32("flag")
	if flag != 0 && flag != 1 {
		cmd.UsageFunc()(cmd)
		return
	}
	params := types.MinerFlag{
		Flag: flag,
	}

	var res jsonrpc.Reply
	ctx := NewRPCCtx(rpcLaddr, "Chain33.SetAutoMining", params, &res)
	ctx.Run()
}

// peer_info
func PeerInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "peer_info",
		Short: "Get remote peer nodes",
		Run:   peerInfo,
	}
	return cmd
}

func peerInfo(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res jsonrpc.PeerList
	ctx := NewRPCCtx(rpcLaddr, "Chain33.GetPeerInfo", nil, &res)
	ctx.Run()
}

// bind_miner
func BindMinerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bind_miner",
		Short: "Bind private key to miner address",
		Run:   bindMiner,
	}
	addBindMinerFlags(cmd)
	return cmd
}

func addBindMinerFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", `miner address`)
	cmd.MarkFlagRequired("addr")

	cmd.Flags().StringP("key", "k", "", `private key`)
	cmd.MarkFlagRequired("key")
}

func bindMiner(cmd *cobra.Command, args []string) {
	addr, _ := cmd.Flags().GetString("addr")
	key, _ := cmd.Flags().GetString("key")

	c, _ := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	a, _ := common.FromHex(key)
	privKey, _ := c.PrivKeyFromBytes(a)
	originaddr := account.PubKeyToAddress(privKey.PubKey().Bytes()).String()
	ta := &types.TicketAction{}
	tbind := &types.TicketBind{
		MinerAddress:  addr,
		ReturnAddress: originaddr,
	}
	ta.Value = &types.TicketAction_Tbind{tbind}
	ta.Ty = types.TicketActionBind
	execer := []byte("ticket")
	to := account.ExecAddress(string(execer)).String()
	tx := &types.Transaction{Execer: execer, Payload: types.Encode(ta), To: to}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	var err error
	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	tx.Fee += types.MinFee
	tx.Sign(types.SECP256K1, privKey)
	txHex := types.Encode(tx)
	fmt.Println(hex.EncodeToString(txHex))
}

// IsClockSync
func IsClockSync() *cobra.Command {
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
	ctx := NewRPCCtx(rpcLaddr, "Chain33.IsNtpClockSync", nil, &res)
	ctx.Run()
}
