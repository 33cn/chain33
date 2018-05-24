package commands

import (
	"encoding/hex"
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/model"
	"gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet"
	"math/rand"
	"time"
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
		Transfer2ContractCmd(),
		Withdraw4ContractCmd(),
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

	action := model.ContractAction{Amount: 0, Code: common.FromHex(code), GasLimit: 0, GasPrice: 0, Note: note}

	data, err := createEvmTx(&action, key, "", expire, feeInt64)

	if err != nil {
		fmt.Printf("create contract error:%v", err)
		return
	}

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	params := rpc.RawParm{
		Data: data,
	}

	ctx := NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func createEvmTx(action *model.ContractAction, key, addr, expire string, fee uint64) (string, error) {
	tx := &types.Transaction{Execer: []byte("user.evm"), Payload: types.Encode(action), Fee: 0, To: ""}

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

	amountInt64 := uint64(amount*1e4) * 1e4
	feeInt64 := uint64(fee*1e4) * 1e4

	action := model.ContractAction{Amount: amountInt64, Code: common.FromHex(code), GasLimit: 0, GasPrice: 0, Note: note}

	data, err := createEvmTx(&action, key, to, expire, feeInt64)

	if err != nil {
		fmt.Printf("call contract error:%v", err)
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
	cmd.Flags().StringP("to", "t", "", "contract address")
	cmd.MarkFlagRequired("to")

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

func emptyCmd(cmd *cobra.Command, args []string) {
	fmt.Println("do nothing")
}
func addEmptyFlags(cmd *cobra.Command) {
	fmt.Println("do nothing")
}

// 估算合约消耗
func EstimateContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimate",
		Short: "Estimate the gas cost of calling or creating a contract",
		Run:   emptyCmd,
	}
	addEmptyFlags(cmd)
	return cmd
}

// 向EVM合约账户转账
func Transfer2ContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send to the EVM contract (my account)",
		Run:   emptyCmd,
	}
	addEmptyFlags(cmd)
	return cmd
}

// 从EVM合约账户取钱
func Withdraw4ContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw",
		Short: "Withdraw from the EVM contract (my account)",
		Run:   emptyCmd,
	}
	addEmptyFlags(cmd)
	return cmd
}

// 检查地址是否为EVM合约
func CheckContractAddrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check if the address is valid EVM contract",
		Run:   emptyCmd,
	}
	addEmptyFlags(cmd)
	return cmd
}
