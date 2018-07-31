package commands

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"time"

	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
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
		EvmDebugCmd(),
		EvmTransferCmd(),
		EvmWithdrawCmd(),
		GetEvmBalanceCmd(),
	)

	return cmd
}

// get balance of an execer
func GetEvmBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Get balance of a evm contract address",
		Run:   evmBalance,
	}
	addEvmBalanceFlags(cmd)
	return cmd
}

func addEvmBalanceFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account addr")
	cmd.MarkFlagRequired("addr")

	cmd.Flags().StringP("exec", "e", "", "evm contract name (like user.evm.xxx)")
	cmd.MarkFlagRequired("exec")
}

func evmBalance(cmd *cobra.Command, args []string) {
	// 直接复用coins的查询余额命令
	balance(cmd, args)
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

	cmd.Flags().StringP("alias", "s", "", "human readable contract alias name")
}

func createContract(cmd *cobra.Command, args []string) {
	code, _ := cmd.Flags().GetString("input")
	caller, _ := cmd.Flags().GetString("caller")
	expire, _ := cmd.Flags().GetString("expire")
	note, _ := cmd.Flags().GetString("note")
	alias, _ := cmd.Flags().GetString("alias")
	fee, _ := cmd.Flags().GetFloat64("fee")
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	paraName, _ := cmd.Flags().GetString("paraName")

	feeInt64 := uint64(fee*1e4) * 1e4

	bCode, err := common.FromHex(code)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse evm code error", err)
		return
	}
	action := types.EVMContractAction{Amount: 0, Code: bCode, GasLimit: 0, GasPrice: 0, Note: note, Alias: alias}

	data, err := createEvmTx(&action, types.ExecName(paraName+"evm"), caller, address.ExecAddress(types.ExecName(paraName+"evm")), expire, rpcLaddr, feeInt64)

	if err != nil {
		fmt.Fprintln(os.Stderr, "create contract error:", err)
		return
	}

	params := rpc.RawParm{
		Data: data,
	}

	ctx := NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func createEvmTx(action proto.Message, execer, caller, addr, expire, rpcLaddr string, fee uint64) (string, error) {
	tx := &types.Transaction{Execer: []byte(execer), Payload: types.Encode(action), Fee: 0, To: addr}

	tx.Fee, _ = tx.GetRealFee(types.MinFee)
	if tx.Fee < int64(fee) {
		tx.Fee += int64(fee)
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()

	txHex := types.Encode(tx)
	rawTx := hex.EncodeToString(txHex)

	unsignedTx := &types.ReqSignRawTx{
		Addr:   caller,
		TxHex:  rawTx,
		Expire: expire,
	}

	var res string
	client, err := rpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", err
	}
	err = client.Call("Chain33.SignRawTx", unsignedTx, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", err
	}

	return res, nil
}

func createEvmTransferTx(cmd *cobra.Command, caller, execName, expire, rpcLaddr string, amountInt64 int64, isWithdraw bool) (string, error) {
	paraName, _ := cmd.Flags().GetString("paraName")
	var tx *types.Transaction
	transfer := &types.CoinsAction{}

	if isWithdraw {
		transfer.Value = &types.CoinsAction_Withdraw{Withdraw: &types.CoinsWithdraw{Amount: amountInt64, ExecName: execName, To: address.ExecAddress(execName)}}
		transfer.Ty = types.CoinsActionWithdraw
	} else {
		transfer.Value = &types.CoinsAction_TransferToExec{TransferToExec: &types.CoinsTransferToExec{Amount: amountInt64, ExecName: execName, To: address.ExecAddress(execName)}}
		transfer.Ty = types.CoinsActionTransferToExec
	}
	if paraName == "" {
		tx = &types.Transaction{Execer: []byte(types.ExecName(paraName + "coins")), Payload: types.Encode(transfer), To: address.ExecAddress(execName)}
	} else {
		tx = &types.Transaction{Execer: []byte(types.ExecName(paraName + "coins")), Payload: types.Encode(transfer), To: address.ExecAddress(types.ExecName(paraName + "coins"))}
	}

	var err error
	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		return "", err
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()

	txHex := types.Encode(tx)
	rawTx := hex.EncodeToString(txHex)

	unsignedTx := &types.ReqSignRawTx{
		Addr:   caller,
		TxHex:  rawTx,
		Expire: expire,
	}

	var res string
	client, err := rpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", err
	}
	err = client.Call("Chain33.SignRawTx", unsignedTx, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", err
	}

	return res, nil
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
	caller, _ := cmd.Flags().GetString("caller")
	expire, _ := cmd.Flags().GetString("expire")
	note, _ := cmd.Flags().GetString("note")
	amount, _ := cmd.Flags().GetFloat64("amount")
	fee, _ := cmd.Flags().GetFloat64("fee")
	name, _ := cmd.Flags().GetString("exec")
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")

	amountInt64 := uint64(amount*1e4) * 1e4
	feeInt64 := uint64(fee*1e4) * 1e4
	toAddr := address.ExecAddress(name)

	bCode, err := common.FromHex(code)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse evm code error", err)
		return
	}

	action := types.EVMContractAction{Amount: amountInt64, Code: bCode, GasLimit: 0, GasPrice: 0, Note: note}

	//name表示发给哪个执行器
	data, err := createEvmTx(&action, name, caller, toAddr, expire, rpcLaddr, feeInt64)

	if err != nil {
		fmt.Fprintln(os.Stderr, "call contract error", err)
		return
	}

	params := rpc.RawParm{
		Data: data,
	}

	ctx := NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func addCallContractFlags(cmd *cobra.Command) {
	addCommonFlags(cmd)
	cmd.Flags().StringP("exec", "e", "", "evm contract name, like user.evm.xxxxx")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().Float64P("amount", "a", 0, "the amount transfer to the contract (optional)")
}

func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("input", "i", "", "input contract binary code")
	cmd.MarkFlagRequired("input")

	cmd.Flags().StringP("caller", "c", "", "the caller address")
	cmd.MarkFlagRequired("caller")

	cmd.Flags().StringP("expire", "p", "120s", "transaction expire time (optional)")

	cmd.Flags().StringP("note", "n", "", "transaction note info (optional)")

	cmd.Flags().Float64P("fee", "f", 0, "contract gas fee (optional)")
}

func estimateContract(cmd *cobra.Command, args []string) {
	code, _ := cmd.Flags().GetString("input")
	name, _ := cmd.Flags().GetString("exec")
	caller, _ := cmd.Flags().GetString("caller")
	amount, _ := cmd.Flags().GetFloat64("amount")

	toAddr := address.ExecAddress("evm")
	if len(name) > 0 {
		toAddr = address.ExecAddress(name)
	}

	amountInt64 := uint64(amount*1e4) * 1e4
	bCode, err := common.FromHex(code)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse evm code error", err)
		return
	}

	var estGasReq = types.EstimateEVMGasReq{To: toAddr, Code: bCode, Caller: caller, Amount: amountInt64}
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

	cmd.Flags().StringP("exec", "e", "", "evm contract name (like user.evm.xxxxx)")

	cmd.Flags().StringP("caller", "c", "", "the caller address")

	cmd.Flags().Float64P("amount", "a", 0, "the amount transfer to the contract (optional)")
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
		Short: "Check if the address is a valid EVM contract",
		Run:   checkContractAddr,
	}
	addCheckContractAddrFlags(cmd)
	return cmd
}

func addCheckContractAddrFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("to", "t", "", "evm contract address (optional)")
	cmd.Flags().StringP("exec", "e", "", "evm contract name, like user.evm.xxxxx (optional)")
}

func checkContractAddr(cmd *cobra.Command, args []string) {
	to, _ := cmd.Flags().GetString("to")
	name, _ := cmd.Flags().GetString("exec")
	toAddr := to
	if len(toAddr) == 0 && len(name) > 0 {
		if strings.Contains(name, "evm") {
			toAddr = address.ExecAddress(name)
		}
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
	} else {
		fmt.Fprintln(os.Stderr, "error")
	}
}

// 查询或设置EVM调试开关
func EvmDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Query current debug status (default) or set debug flag ( -s flag)",
		Run:   evmDebug,
	}
	addEvmDebugFlags(cmd)
	return cmd
}

func addEvmDebugFlags(cmd *cobra.Command) {
	cmd.Flags().IntP("set", "s", 0, "set debug flag (>0 to true, <0 to false optional)")
}

func evmDebug(cmd *cobra.Command, args []string) {
	debug, _ := cmd.Flags().GetInt("set")

	var debugReq = types.EvmDebugReq{Optype: int32(debug)}
	var debugResp types.EvmDebugResp
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	query := sendQuery(rpcLaddr, "EvmDebug", debugReq, &debugResp)

	if query {
		proto.MarshalText(os.Stdout, &debugResp)
	} else {
		fmt.Fprintln(os.Stderr, "error")
	}
}

// 向EVM合约地址转账
func EvmTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Transfer to evm contract address",
		Run:   evmTransfer,
	}
	addEvmTransferFlags(cmd)
	return cmd
}

func addEvmTransferFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("to", "t", "", "evm contract address (like user.evm.xxx)")
	cmd.MarkFlagRequired("to")

	cmd.Flags().StringP("caller", "c", "", "the caller address")
	cmd.MarkFlagRequired("caller")

	cmd.Flags().Float64P("amount", "a", 0, "the amount transfer to the contract")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("expire", "p", "120s", "transaction expire time (optional)")
}

func evmTransfer(cmd *cobra.Command, args []string) {
	caller, _ := cmd.Flags().GetString("caller")
	amount, _ := cmd.Flags().GetFloat64("amount")
	to, _ := cmd.Flags().GetString("to")
	expire, _ := cmd.Flags().GetString("expire")
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")

	amountInt64 := int64(amount*1e4) * 1e4

	data, err := createEvmTransferTx(cmd, caller, to, expire, rpcLaddr, amountInt64, false)

	if err != nil {
		fmt.Fprintln(os.Stderr, "create contract transfer error:", err)
		return
	}

	params := rpc.RawParm{
		Data: data,
	}

	ctx := NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

// 向EVM合约地址转账
func EvmWithdrawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw",
		Short: "Withdraw from evm contract address to caller's balance",
		Run:   evmWithdraw,
	}
	addEvmWithdrawFlags(cmd)
	return cmd
}

func addEvmWithdrawFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("exec", "e", "", "evm contract address (like user.evm.xxx)")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().StringP("caller", "c", "", "the caller address")
	cmd.MarkFlagRequired("caller")

	cmd.Flags().Float64P("amount", "a", 0, "the amount transfer to the contract")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("expire", "p", "120s", "transaction expire time (optional)")
}

func evmWithdraw(cmd *cobra.Command, args []string) {
	caller, _ := cmd.Flags().GetString("caller")
	amount, _ := cmd.Flags().GetFloat64("amount")
	from, _ := cmd.Flags().GetString("exec")
	expire, _ := cmd.Flags().GetString("expire")
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")

	amountInt64 := int64(amount*1e4) * 1e4

	data, err := createEvmTransferTx(cmd, caller, from, expire, rpcLaddr, amountInt64, true)

	if err != nil {
		fmt.Fprintln(os.Stderr, "create contract transfer error:", err)
		return
	}

	params := rpc.RawParm{
		Data: data,
	}

	ctx := NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func sendQuery(rpcAddr, funcName string, request, result interface{}) bool {
	params := rpc.Query4Cli{
		Execer:   "evm",
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
