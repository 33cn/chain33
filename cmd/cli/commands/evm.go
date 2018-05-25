package commands

import (
	"encoding/hex"
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet"
	"math/rand"
	"os"
	"time"
	"github.com/golang/protobuf/proto"
)

func EvmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evm",
		Short: "EVM contracts operation",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		CreateContractCmd(),
		CallContractCmd(),
		EstimateContractCmd(),
		CheckContractAddrCmd(),
	)

	return cmd
}

// 创建EVM合约
func CreateContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new EVM contract",
		Run:   createContract,
	}
	addCreateContractFlags(cmd)
	return cmd
}

func addCreateContractFlags(cmd *cobra.Command) {
	addCommonFlags(cmd)
}

func createContract(cmd *cobra.Command, args []string) {
	code, _ := cmd.Flags().GetString("input")
	key, _ := cmd.Flags().GetString("key")
	expire, _ := cmd.Flags().GetString("expire")
	note, _ := cmd.Flags().GetString("note")
	fee, _ := cmd.Flags().GetFloat64("fee")
	feeInt64 := uint64(fee*1e4) * 1e4

	bCode, err := common.FromHex(code)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse evm code error", err)
		return
	}
	action := types.EVMContractAction{Amount: 0, Code: bCode, GasLimit: 0, GasPrice: 0, Note: note}

	data, err := createEvmTx(&action, key, "", expire, feeInt64)

	if err != nil {
		fmt.Fprintln(os.Stderr, "create contract error:", err)
		return
	}

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	params := rpc.RawParm{
		Data: data,
	}

	ctx := NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func createEvmTx(action *types.EVMContractAction, key, addr, expire string, fee uint64) (string, error) {
	tx := &types.Transaction{Execer: []byte("user.evm"), Payload: types.Encode(action), Fee: 0, To: addr}

	tx.Fee, _ = tx.GetRealFee(types.MinBalanceTransfer)
	tx.Fee += types.MinBalanceTransfer

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()

	txHex := types.Encode(tx)
	rawTx := hex.EncodeToString(txHex)

	unsignedTx := &types.ReqSignRawTx{
		Addr:    addr,
		Privkey: key,
		TxHex:   rawTx,
		Expire:  expire,
	}

	wal := &wallet.Wallet{}
	return wal.ProcSignRawTx(unsignedTx)
}

// 调用EVM合约
func CallContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call",
		Short: "Call the EVM contract",
		Run:   callContract,
	}
	addCallContractFlags(cmd)
	return cmd
}

func callContract(cmd *cobra.Command, args []string) {
	code, _ := cmd.Flags().GetString("input")
	key, _ := cmd.Flags().GetString("key")
	expire, _ := cmd.Flags().GetString("expire")
	note, _ := cmd.Flags().GetString("note")
	amount, _ := cmd.Flags().GetFloat64("amount")
	fee, _ := cmd.Flags().GetFloat64("fee")
	to, _ := cmd.Flags().GetString("to")
	name, _ := cmd.Flags().GetString("name")

	amountInt64 := uint64(amount*1e4) * 1e4
	feeInt64 := uint64(fee*1e4) * 1e4
	toAddr := to
	if len(toAddr) == 0 && len(name) > 0 {
		toAddr = account.ExecAddress(name).String()
	}
	if len(toAddr) == 0 {
		fmt.Fprintln(os.Stderr, "one of the 'to (contract address)' and 'name (contract name)' must be set")
		return
	}

	bCode, err := common.FromHex(code)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse evm code error", err)
		return
	}

	action := types.EVMContractAction{Amount: amountInt64, Code: bCode, GasLimit: 0, GasPrice: 0, Note: note}

	data, err := createEvmTx(&action, key, toAddr, expire, feeInt64)

	if err != nil {
		fmt.Fprintln(os.Stderr, "call contract error:%v", err)
		return
	}

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	params := rpc.RawParm{
		Data: data,
	}

	ctx := NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
	ctx.RunWithoutMarshal()

	fmt.Println("do nothing")
}

func addCallContractFlags(cmd *cobra.Command) {
	addCommonFlags(cmd)
	cmd.Flags().StringP("to", "t", "", "contract address (optional)")
	cmd.Flags().StringP("name", "m", "", "contract name, like user.evm.xxxxx (optional)")

	cmd.Flags().Float64P("amount", "a", 0, "the amount transfer to the contract (optional)")
}

func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("input", "i", "", "input contract binary code")
	cmd.MarkFlagRequired("input")

	cmd.Flags().StringP("key", "k", "", "private key")
	cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("expire", "e", "120s", "transaction expire time (optional)")

	cmd.Flags().StringP("note", "n", "", "transaction note info (optional)")

	cmd.Flags().Float64P("fee", "f", 0, "contract gas fee")
	cmd.MarkFlagRequired("fee")
}

func estimateContract(cmd *cobra.Command, args []string) {
	code, _ := cmd.Flags().GetString("input")
	to, _ := cmd.Flags().GetString("to")
	name, _ := cmd.Flags().GetString("name")

	toAddr := to
	if len(toAddr) == 0 && len(name) > 0 {
		toAddr = account.ExecAddress(name).String()
	}

	bCode, err := common.FromHex(code)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse evm code error", err)
		return
	}

	var estGasReq = types.EstimateEVMGasReq{To: toAddr, Code: bCode}
	var estGasResp types.EstimateEVMGasResp
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	query := sendQuery(rpcLaddr, "EstimateGas", estGasReq, &estGasResp)

	if query {
		fmt.Fprintf(os.Stdout, "gas cost estimate %v\n", estGasResp.Gas)
	} else {
		fmt.Fprintln(os.Stderr, "gas cost estimate error")
	}
}

func addEstimateFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("input", "i", "", "input contract binary code")
	cmd.MarkFlagRequired("input")

	cmd.Flags().StringP("to", "t", "", "contract address (optional)")
	cmd.Flags().StringP("name", "m", "", "contract name, like user.evm.xxxxx (optional)")

}

// 估算合约消耗
func EstimateContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimate",
		Short: "Estimate the gas cost of calling or creating a contract",
		Run:   estimateContract,
	}
	addEstimateFlags(cmd)
	return cmd
}

// 检查地址是否为EVM合约
func CheckContractAddrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check if the address (or name) is a valid EVM contract",
		Run:   checkContractAddr,
	}
	addCheckContractAddrFlags(cmd)
	return cmd
}

func addCheckContractAddrFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("to", "t", "", "contract address (optional)")
	cmd.Flags().StringP("name", "m", "", "contract name, like user.evm.xxxxx (optional)")
}

func checkContractAddr(cmd *cobra.Command, args []string) {
	to, _ := cmd.Flags().GetString("to")
	name, _ := cmd.Flags().GetString("name")
	toAddr := to
	if len(toAddr) == 0 && len(name) > 0 {
		toAddr = account.ExecAddress(name).String()
	}
	if len(toAddr) == 0 {
		fmt.Fprintln(os.Stderr, "one of the 'to (contract address)' and 'name (contract name)' must be set")
		cmd.Help()
		return
	}

	var checkAddrReq = types.CheckEVMAddrReq{Addr: toAddr}
	var checkAddrResp types.CheckEVMAddrResp
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	query := sendQuery(rpcLaddr, "CheckAddrExists", checkAddrReq, &checkAddrResp)

	if query {
		proto.MarshalText(os.Stdout, &checkAddrResp)
	}else{
		fmt.Fprintln(os.Stderr,"error")
	}
}

func sendQuery(rpcAddr, funcName string, request, result interface{}) bool {
	params := rpc.Query4Cli{
		Execer:   "user.evm",
		FuncName: funcName,
		Payload:  request,
	}

	rpc, err := rpc.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}

	err = rpc.Call("Chain33.Query", params, &result)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}
	return true
}
