package commands

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"unsafe"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	"gitlab.33.cn/chain33/chain33/types"
)

func BTYCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bty",
		Short: "Construct BTY transactions",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		//TransferCmd(),
		//WithdrawFromExecCmd(),
		CreateRawTransferCmd(),
		CreateRawWithdrawCmd(),
		CreateRawSendToExecCmd(),
		CreateTxGroupCmd(),
		CreatePub2PrivTxCmd(),
		CreatePriv2PrivTxCmd(),
		CreatePriv2PubTxCmd(),
	)

	return cmd
}

// create raw transfer tx
func CreateRawTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Create a transfer transaction",
		Run:   createTransfer,
	}
	addCreateTransferFlags(cmd)
	return cmd
}

func addCreateTransferFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0, "transaction amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
}

func createTransfer(cmd *cobra.Command, args []string) {
	toAddr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	txHex, err := CreateRawTx(toAddr, amount, note, false, false, "", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// create raw withdraw tx
func CreateRawWithdrawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw",
		Short: "Create a withdraw transaction",
		Run:   createWithdraw,
	}
	addCreateWithdrawFlags(cmd)
	return cmd
}

func addCreateWithdrawFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("exec", "e", "", "execer withdrawn from")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().Float64P("amount", "a", 0, "withdraw amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
}

func createWithdraw(cmd *cobra.Command, args []string) {
	exec, _ := cmd.Flags().GetString("exec")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	execAddr, err := GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex, err := CreateRawTx(execAddr, amount, note, true, false, "", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// create send to exec
func CreateRawSendToExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send_exec",
		Short: "Create a send to executor transaction",
		Run:   sendToExec,
	}
	addCreateRawSendToExecFlags(cmd)
	return cmd
}

func addCreateRawSendToExecFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("exec", "e", "", "executor to be sent to")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().Float64P("amount", "a", 0, "send amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
}

func sendToExec(cmd *cobra.Command, args []string) {
	exec, _ := cmd.Flags().GetString("exec")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	execAddr, err := GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex, err := CreateRawTx(execAddr, amount, note, false, false, "", exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// send to address
func TransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Send coins to address",
		Run:   transfer,
	}
	addTransferFlags(cmd)
	return cmd
}

func addTransferFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("from", "f", "", "sender account address")
	cmd.MarkFlagRequired("from")

	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount, at most 4 decimal places")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
}

func transfer(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	fromAddr, _ := cmd.Flags().GetString("from")
	toAddr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	SendToAddress(rpcLaddr, fromAddr, toAddr, amountInt64, note, false, "", false)
}

// withdraw from executor
func WithdrawFromExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw",
		Short: "Withdraw coin from executor",
		Run:   withdraw,
	}
	addWithdrawFlags(cmd)
	return cmd
}

func addWithdrawFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "t", "", "withdraw account address")
	cmd.MarkFlagRequired("addr")

	cmd.Flags().StringP("exec", "e", "", "execer withdrawn from")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount, at most 4 decimal places")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
}

func withdraw(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	exec, _ := cmd.Flags().GetString("exec")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	execAddr, err := GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	SendToAddress(rpcLaddr, addr, execAddr, amountInt64, note, false, "", true)
}

// create tx group
func CreateTxGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "txgroup",
		Short: "Create a transaction group",
		Run:   createTxGroup,
	}
	addCreateTxGroupFlags(cmd)
	return cmd
}

func addCreateTxGroupFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("txs", "t", "", "transactions in hex, separated by space")
	cmd.MarkFlagRequired("txs")
}

func createTxGroup(cmd *cobra.Command, args []string) {
	txs, _ := cmd.Flags().GetString("txs")
	txsArr := strings.Split(txs, " ")
	var transactions []*types.Transaction
	for _, t := range txsArr {
		txByte, err := hex.DecodeString(t)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		var transaction types.Transaction
		types.Decode(txByte, &transaction)
		transactions = append(transactions, &transaction)
	}
	group, err := types.CreateTxGroup(transactions)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	err = group.Check(types.MinFee)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	newtx := group.Tx()
	grouptx := hex.EncodeToString(types.Encode(newtx))
	fmt.Println(grouptx)
}

// privacy transaction
// decomAmount2Nature 将amount切分为1,2,5的组合，这样在进行amount混淆的时候就能够方便获取相同额度的utxo
func decomAmount2Nature(amount int64, order int64) []int64 {
	res := make([]int64, 0)
	if order == 0 {
		return nil
	}
	mul := amount / order
	switch mul {
	case 3:
		res = append(res, order)
		res = append(res, 2*order)
	case 4:
		res = append(res, 2*order)
		res = append(res, 2*order)
	case 6:
		res = append(res, 5*order)
		res = append(res, order)
	case 7:
		res = append(res, 5*order)
		res = append(res, 2*order)
	case 8:
		res = append(res, 5*order)
		res = append(res, 2*order)
		res = append(res, 1*order)
	case 9:
		res = append(res, 5*order)
		res = append(res, 2*order)
		res = append(res, 2*order)
	default:
		res = append(res, mul*order)
		return res
	}
	return res
}

// 62387455827 -> 455827 + 7000000 + 80000000 + 300000000 + 2000000000 + 60000000000, where 455827 <= dust_threshold
//res:[455827, 7000000, 80000000, 300000000, 2000000000, 60000000000]
func decomposeAmount2digits(amount, dust_threshold int64) []int64 {
	res := make([]int64, 0)
	if 0 >= amount {
		return res
	}

	is_dust_handled := false
	var dust int64 = 0
	var order int64 = 1
	var chunk int64 = 0

	for 0 != amount {
		chunk = (amount % 10) * order
		amount /= 10
		order *= 10
		if dust+chunk < dust_threshold {
			dust += chunk //累加小数，直到超过dust_threshold为止
		} else {
			if !is_dust_handled && 0 != dust {
				//1st 正常情况下，先把dust保存下来
				res = append(res, dust)
				is_dust_handled = true
			}
			if 0 != chunk {
				//2nd 然后依次将大的整数额度进行保存
				goodAmount := decomAmount2Nature(chunk, order/10)
				res = append(res, goodAmount...)
			}
		}
	}

	//如果需要被拆分的额度 < dust_threshold，则直接将其进行保存
	if !is_dust_handled && 0 != dust {
		res = append(res, dust)
	}

	return res
}

//最后构造完成的utxo依次是2种类型，不构造交易费utxo，使其直接燃烧消失
//1.进行实际转账utxo
//2.进行找零转账utxo
func generateOuts(viewpubTo, spendpubto, viewpubChangeto, spendpubChangeto *[32]byte, transAmount, selectedAmount, fee int64) (*types.PrivacyOutput, error) {
	decomDigit := decomposeAmount2digits(transAmount, types.BTYDustThreshold)
	//计算找零
	changeAmount := selectedAmount - transAmount - fee
	var decomChange []int64
	if 0 < changeAmount {
		decomChange = decomposeAmount2digits(changeAmount, types.BTYDustThreshold)
	}

	pk := &privacy.PubKeyPrivacy{}
	sk := &privacy.PrivKeyPrivacy{}
	privacy.GenerateKeyPair(sk, pk)
	RtxPublicKey := pk.Bytes()

	sktx := (*[32]byte)(unsafe.Pointer(&sk[0]))
	var privacyOutput types.PrivacyOutput
	privacyOutput.RpubKeytx = RtxPublicKey
	privacyOutput.Keyoutput = make([]*types.KeyOutput, len(decomDigit)+len(decomChange))

	//添加本次转账的目的接收信息（UTXO），包括一次性公钥和额度
	for index, digit := range decomDigit {
		pubkeyOnetime, err := privacy.GenerateOneTimeAddr(viewpubTo, spendpubto, sktx, int64(index))
		if err != nil {
			return nil, err
		}
		keyOutput := &types.KeyOutput{
			Amount:        digit,
			Onetimepubkey: pubkeyOnetime[:],
		}
		privacyOutput.Keyoutput[index] = keyOutput
	}
	//添加本次转账选择的UTXO后的找零后的UTXO
	for index, digit := range decomChange {
		pubkeyOnetime, err := privacy.GenerateOneTimeAddr(viewpubChangeto, spendpubChangeto, sktx, int64(index+len(decomDigit)))
		if err != nil {
			return nil, err
		}
		keyOutput := &types.KeyOutput{
			Amount:        digit,
			Onetimepubkey: pubkeyOnetime[:],
		}
		privacyOutput.Keyoutput[index+len(decomDigit)] = keyOutput
	}
	//交易费不产生额外的utxo，方便执行器执行的时候直接燃烧殆尽
	if 0 != fee {
	}

	return &privacyOutput, nil
}

func parseViewSpendPubKeyPair(in string) (viewPubKey, spendPubKey []byte, err error) {
	src, err := common.FromHex(in)
	if err != nil {
		return nil, nil, err
	}
	if 64 != len(src) {
		return nil, nil, types.ErrPubKeyLen
	}
	viewPubKey = src[:32]
	spendPubKey = src[32:]
	return
}

func CreatePub2PrivTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pub2priv",
		Short: "Create a public to privacy transaction",
		Run:   createPub2PrivTx,
	}
	createPub2PrivTxFlags(cmd)
	return cmd
}

func createPub2PrivTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("pubkeypair", "p", "", "public key pair")
	cmd.MarkFlagRequired("pubkeypair")
	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount, at most 4 decimal places")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("tokenname", "t", "BTY", "token name")
	cmd.Flags().StringP("note", "n", "", "note for transaction")
}

func createPub2PrivTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	pubkeypair, _ := cmd.Flags().GetString("pubkeypair")
	amount := GetAmountValue(cmd, "amount")
	tokenname, _ := cmd.Flags().GetString("tokenname")
	note, _ := cmd.Flags().GetString("note")

	params := types.ReqCreateTransaction{
		Tokenname:  tokenname,
		Type:       1,
		Amount:     amount,
		Note:       note,
		Pubkeypair: pubkeypair,
	}
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateTrasaction", params, nil)
	ctx.RunWithoutMarshal()
}

func CreatePriv2PrivTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "priv2priv",
		Short: "Create a privacy to privacy transaction",
		Run:   createPriv2PrivTx,
	}
	createPriv2PrivTxFlags(cmd)
	return cmd
}

func createPriv2PrivTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("pubkeypair", "p", "", "public key pair")
	cmd.MarkFlagRequired("pubkeypair")
	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount, at most 4 decimal places")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().StringP("sender", "s", "", "account address")
	cmd.MarkFlagRequired("sender")

	cmd.Flags().StringP("tokenname", "t", "BTY", "token name")
	cmd.Flags().StringP("note", "n", "", "note for transaction")
}

func createPriv2PrivTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	pubkeypair, _ := cmd.Flags().GetString("pubkeypair")
	amount := GetAmountValue(cmd, "amount")
	tokenname, _ := cmd.Flags().GetString("tokenname")
	note, _ := cmd.Flags().GetString("note")
	sender, _ := cmd.Flags().GetString("sender")

	params := types.ReqCreateTransaction{
		Tokenname:  tokenname,
		Type:       2,
		Amount:     amount,
		Note:       note,
		Pubkeypair: pubkeypair,
		From:       sender,
		Mixcount:   16,
	}
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateTrasaction", params, nil)
	ctx.RunWithoutMarshal()
}

func CreatePriv2PubTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "priv2pub",
		Short: "Create a privacy to public transaction",
		Run:   createPriv2PubTx,
	}
	createPriv2PubTxFlags(cmd)
	return cmd
}

func createPriv2PubTxFlags(cmd *cobra.Command) {
	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount, at most 4 decimal places")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().StringP("from", "f", "", "from account address")
	cmd.MarkFlagRequired("from")
	cmd.Flags().StringP("to", "o", "", "to account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().StringP("tokenname", "t", "BTY", "token name")
	cmd.Flags().StringP("note", "n", "", "note for transaction")
}

func createPriv2PubTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	amount := GetAmountValue(cmd, "amount")
	tokenname, _ := cmd.Flags().GetString("tokenname")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	note, _ := cmd.Flags().GetString("note")

	params := types.ReqCreateTransaction{
		Tokenname: tokenname,
		Type:      3,
		Amount:    amount,
		Note:      note,
		From:      from,
		To:        to,
		Mixcount:  16,
	}
	ctx := NewRpcCtx(rpcLaddr, "Chain33.CreateTrasaction", params, nil)
	ctx.RunWithoutMarshal()
}
