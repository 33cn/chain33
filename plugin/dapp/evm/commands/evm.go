package commands

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"time"

	"strings"

	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto/sha3"
	common2 "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	evmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
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
		EvmToolsCmd(),
	)

	return cmd
}

// some tools for evm op
func EvmToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tool",
		Short: "Some tools for evm op",
	}
	cmd.AddCommand(EvmToolsAddressCmd())
	return cmd
}

// transfer address format between ethereum and chain33
func EvmToolsAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "address",
		Short: "Transfer address format between ethereum and local (you should input one address of them)",
		Run:   transferAddress,
	}
	addEvmAddressFlags(cmd)
	return cmd
}

func addEvmAddressFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("eth", "e", "", "ethereum address")

	cmd.Flags().StringP("local", "l", "", "evm contract address (like user.evm.xxx or plain address)")
}

func transferAddress(cmd *cobra.Command, args []string) {
	eth, _ := cmd.Flags().GetString("eth")
	local, _ := cmd.Flags().GetString("local")
	if len(eth) == 40 || len(eth) == 42 {
		data, err := common.FromHex(eth)
		if err != nil {
			fmt.Println(fmt.Errorf("ethereum address is invalid: %v", eth))
			return
		}
		fmt.Println(fmt.Sprintf("Ethereum Address: %v", eth))
		fmt.Println(fmt.Sprintf("Local Address: %v", common2.BytesToAddress(data).String()))
		return
	}
	if len(local) >= 34 {
		var addr common2.Address
		if strings.HasPrefix(local, evmtypes.EvmPrefix) {
			addr = common2.ExecAddress(local)
			fmt.Println(fmt.Sprintf("Local Contract Name: %v", local))
			fmt.Println(fmt.Sprintf("Local Address: %v", addr.String()))
		} else {
			addrP := common2.StringToAddress(local)
			if addrP == nil {
				fmt.Println(fmt.Errorf("Local address is invalid: %v", local))
				return
			}
			addr = *addrP
			fmt.Println(fmt.Sprintf("Local Address: %v", local))
		}
		fmt.Println(fmt.Sprintf("Ethereum Address: %v", ChecksumAddr(addr.Bytes())))

		return
	}
	fmt.Fprintln(os.Stderr, "address is invalid!")
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
	//balance(cmd, args)

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	execer, _ := cmd.Flags().GetString("exec")
	err := address.CheckAddress(addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, types.ErrInvalidAddress)
		return
	}
	if ok := types.IsAllowExecName([]byte(execer), []byte(execer)); !ok {
		fmt.Fprintln(os.Stderr, types.ErrExecNameNotAllow)
		return
	}

	var addrs []string
	addrs = append(addrs, addr)
	params := types.ReqBalance{
		Addresses: addrs,
		Execer:    execer,
		StateHash: "",
	}
	var res []*rpctypes.Account
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetBalance", params, &res)
	ctx.SetResultCb(parseGetBalanceRes)
	ctx.Run()
}

type AccountResult struct {
	Currency int32  `json:"currency,omitempty"`
	Balance  string `json:"balance,omitempty"`
	Frozen   string `json:"frozen,omitempty"`
	Addr     string `json:"addr,omitempty"`
}

func parseGetBalanceRes(arg interface{}) (interface{}, error) {
	res := *arg.(*[]*rpctypes.Account)
	balanceResult := strconv.FormatFloat(float64(res[0].Balance)/float64(types.Coin), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(res[0].Frozen)/float64(types.Coin), 'f', 4, 64)
	result := &AccountResult{
		Addr:     res[0].Addr,
		Currency: res[0].Currency,
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	return result, nil
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
	action := evmtypes.EVMContractAction{Amount: 0, Code: bCode, GasLimit: 0, GasPrice: 0, Note: note, Alias: alias}

	data, err := createEvmTx(&action, types.ExecName(paraName+"evm"), caller, address.ExecAddress(types.ExecName(paraName+"evm")), expire, rpcLaddr, feeInt64)

	if err != nil {
		fmt.Fprintln(os.Stderr, "create contract error:", err)
		return
	}

	params := rpctypes.RawParm{
		Data: data,
	}

	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func createEvmTx(action proto.Message, execer, caller, addr, expire, rpcLaddr string, fee uint64) (string, error) {
	tx := &types.Transaction{Execer: []byte(execer), Payload: types.Encode(action), Fee: 0, To: addr}

	tx.Fee, _ = tx.GetRealFee(types.GInt("MinFee"))
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
	client, err := jsonclient.NewJSONClient(rpcLaddr)
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
	transfer := &cty.CoinsAction{}

	if isWithdraw {
		transfer.Value = &cty.CoinsAction_Withdraw{Withdraw: &types.AssetsWithdraw{Amount: amountInt64, ExecName: execName, To: address.ExecAddress(execName)}}
		transfer.Ty = cty.CoinsActionWithdraw
	} else {
		transfer.Value = &cty.CoinsAction_TransferToExec{TransferToExec: &types.AssetsTransferToExec{Amount: amountInt64, ExecName: execName, To: address.ExecAddress(execName)}}
		transfer.Ty = cty.CoinsActionTransferToExec
	}
	if paraName == "" {
		tx = &types.Transaction{Execer: []byte(types.ExecName(paraName + "coins")), Payload: types.Encode(transfer), To: address.ExecAddress(execName)}
	} else {
		tx = &types.Transaction{Execer: []byte(types.ExecName(paraName + "coins")), Payload: types.Encode(transfer), To: address.ExecAddress(types.ExecName(paraName + "coins"))}
	}

	var err error
	tx.Fee, err = tx.GetRealFee(types.GInt("MinFee"))
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
	client, err := jsonclient.NewJSONClient(rpcLaddr)
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

	action := evmtypes.EVMContractAction{Amount: amountInt64, Code: bCode, GasLimit: 0, GasPrice: 0, Note: note}

	//name表示发给哪个执行器
	data, err := createEvmTx(&action, name, caller, toAddr, expire, rpcLaddr, feeInt64)

	if err != nil {
		fmt.Fprintln(os.Stderr, "call contract error", err)
		return
	}

	params := rpctypes.RawParm{
		Data: data,
	}

	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
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

	var estGasReq = evmtypes.EstimateEVMGasReq{To: toAddr, Code: bCode, Caller: caller, Amount: amountInt64}
	var estGasResp evmtypes.EstimateEVMGasResp
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
		if strings.Contains(name, evmtypes.EvmPrefix) {
			toAddr = address.ExecAddress(name)
		}
	}
	if len(toAddr) == 0 {
		fmt.Fprintln(os.Stderr, "one of the 'to (contract address)' and 'name (contract name)' must be set")
		cmd.Help()
		return
	}

	var checkAddrReq = evmtypes.CheckEVMAddrReq{Addr: toAddr}
	var checkAddrResp evmtypes.CheckEVMAddrResp
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	query := sendQuery(rpcLaddr, "CheckAddrExists", checkAddrReq, &checkAddrResp)

	if query && checkAddrResp.Contract {
		proto.MarshalText(os.Stdout, &checkAddrResp)
	} else {
		fmt.Fprintln(os.Stderr, "not evm contract addr!")
	}
}

// 查询或设置EVM调试开关
func EvmDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Query or set evm debug status",
	}
	cmd.AddCommand(
		EvmDebugQueryCmd(),
		EvmDebugSetCmd(),
		EvmDebugClearCmd())

	return cmd
}

func EvmDebugQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "query",
		Short: "Query evm debug status",
		Run:   evmDebugQuery,
	}
}
func EvmDebugSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set",
		Short: "Set evm debug to ON",
		Run:   evmDebugSet,
	}
}
func EvmDebugClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Set evm debug to OFF",
		Run:   evmDebugClear,
	}
}

func evmDebugQuery(cmd *cobra.Command, args []string) {
	evmDebugRpc(cmd, 0)
}

func evmDebugSet(cmd *cobra.Command, args []string) {
	evmDebugRpc(cmd, 1)
}

func evmDebugClear(cmd *cobra.Command, args []string) {
	evmDebugRpc(cmd, -1)
}
func evmDebugRpc(cmd *cobra.Command, flag int32) {
	var debugReq = evmtypes.EvmDebugReq{Optype: flag}
	var debugResp evmtypes.EvmDebugResp
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

	params := rpctypes.RawParm{
		Data: data,
	}

	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
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

	params := rpctypes.RawParm{
		Data: data,
	}

	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func sendQuery(rpcAddr, funcName string, request interface{}, result proto.Message) bool {
	params := types.Query4Cli{
		Execer:   "evm",
		FuncName: funcName,
		Payload:  request,
	}

	jsonrpc, err := jsonclient.NewJSONClient(rpcAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}

	err = jsonrpc.Call("Chain33.Query", params, result)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}
	return true
}

// 这里实现 EIP55中提及的以太坊地址表示方式（增加Checksum）
func ChecksumAddr(address []byte) string {
	unchecksummed := hex.EncodeToString(address[:])
	sha := sha3.NewKeccak256()
	sha.Write([]byte(unchecksummed))
	hash := sha.Sum(nil)

	result := []byte(unchecksummed)
	for i := 0; i < len(result); i++ {
		hashByte := hash[i/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if result[i] > '9' && hashByte > 7 {
			result[i] -= 32
		}
	}
	return "0x" + string(result)
}
