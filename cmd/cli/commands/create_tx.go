package commands

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
)

func CreateTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a raw transaction",
		Run:   createTx,
	}
	addCreateTxFlags(cmd)
	return cmd
}

func addCreateTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("key", "k", "", "private key of sender")
	cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Int64P("amount", "a", 0, "transaction amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
	cmd.MarkFlagRequired("note")
}

func createTx(cmd *cobra.Command, args []string) {
	key, _ := cmd.Flags().GetString("key")
	toAddr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetInt64("amount")
	note, _ := cmd.Flags().GetString("note")

	transfer := &types.CoinsAction{}
	if amount > 0 {
		v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount, Note: note}}
		transfer.Value = v
		transfer.Ty = types.CoinsActionTransfer
	} else {
		v := &types.CoinsAction_Withdraw{&types.CoinsWithdraw{Amount: -amount, Note: note}}
		transfer.Value = v
		transfer.Ty = types.CoinsActionWithdraw
	}

	tx := &types.Transaction{
		Execer:  []byte("coins"),
		Payload: types.Encode(transfer),
		To:      toAddr,
	}

	realFee, err := tx.GetRealFee(types.MinBalanceTransfer)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	// TODO
	tx.Fee += realFee + types.MinBalanceTransfer
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()

	c, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	bytes, err := common.FromHex(key)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	privKey, err := c.PrivKeyFromBytes(bytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	tx.Sign(types.SECP256K1, privKey)

	txHex := types.Encode(tx)
	fmt.Println(hex.EncodeToString(txHex))
}
