package commands

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/rpc"
	jsonrpc "gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/types"
)

func TicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket",
		Short: "Ticket management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		BindMinerCmd(),
		CountTicketCmd(),
		CloseTicketCmd(),
		GetColdAddrByMinerCmd(),
	)

	return cmd
}

// bind miner
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
	cmd.Flags().StringP("bind_addr", "b", "", "miner address")
	cmd.MarkFlagRequired("bind_addr")

	cmd.Flags().StringP("origin_addr", "o", "", "origin address")
	cmd.MarkFlagRequired("origin_addr")
}

func bindMiner(cmd *cobra.Command, args []string) {
	bindAddr, _ := cmd.Flags().GetString("bind_addr")
	originAddr, _ := cmd.Flags().GetString("origin_addr")
	//c, _ := crypto.New(types.GetSignatureTypeName(wallet.SignType))
	//a, _ := common.FromHex(key)
	//privKey, _ := c.PrivKeyFromBytes(a)
	//originAddr := account.PubKeyToAddress(privKey.PubKey().Bytes()).String()
	ta := &types.TicketAction{}
	tBind := &types.TicketBind{
		MinerAddress:  bindAddr,
		ReturnAddress: originAddr,
	}
	ta.Value = &types.TicketAction_Tbind{Tbind: tBind}
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
	//tx.Sign(int32(wallet.SignType), privKey)
	txHex := types.Encode(tx)
	fmt.Println(hex.EncodeToString(txHex))
}

// get ticket count
func CountTicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "count",
		Short: "Get ticket count",
		Run:   countTicket,
	}
	return cmd
}

func countTicket(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res int64
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetTicketCount", nil, &res)
	ctx.Run()
}

// close all accessible tickets
func CloseTicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close",
		Short: "Close ticket ",
		Run:   closeTicket,
	}
	return cmd
}

func closeTicket(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	status, err := getWalletStatus(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	isAutoMining := status.(jsonrpc.WalletStatus).IsAutoMining
	if isAutoMining {
		fmt.Fprintln(os.Stderr, types.ErrMinerNotClosed)
		return
	}

	var res types.ReplyHashes
	rpc, err := rpc.NewJSONClient(rpcLaddr)
	err = rpc.Call("Chain33.CloseTickets", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if len(res.Hashes) == 0 {
		fmt.Println("no ticket to be close")
		return
	}

	data, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(data))
}

func getWalletStatus(rpcAddr string) (interface{}, error) {
	rpc, err := rpc.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	var res jsonrpc.WalletStatus
	err = rpc.Call("Chain33.GetWalletStatus", nil, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	return res, nil
}

// get cold address by miner
func GetColdAddrByMinerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cold",
		Short: "Get cold wallet address of miner",
		Run:   coldAddressOfMiner,
	}
	addColdAddressOfMinerFlags(cmd)
	return cmd
}

func addColdAddressOfMinerFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("miner", "m", "", "miner address")
	cmd.MarkFlagRequired("miner")
}

func coldAddressOfMiner(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("miner")
	reqaddr := &types.ReqString{
		Data: addr,
	}
	var params jsonrpc.Query4Cli
	params.Execer = "ticket"
	params.FuncName = "MinerSourceList"
	params.Payload = reqaddr

	var res types.Message
	ctx := NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.Run()
}
