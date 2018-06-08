package commands

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/types"
)

func ExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Executor operation",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		GetExecAddrCmd(),
		UserDataCmd(),
	)

	return cmd
}

// get address of an execer
func GetExecAddrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addr",
		Short: "Get address of executor",
		Run:   getAddrByExec,
	}
	addGetAddrFlags(cmd)
	return cmd
}

func addGetAddrFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("exec", "e", "", `executor name ("none", "coins", "hashlock", "retrieve", "ticket", "token", "trade" and "privacy" supported)`)
	cmd.MarkFlagRequired("exec")
}

func getAddrByExec(cmd *cobra.Command, args []string) {
	execer, _ := cmd.Flags().GetString("exec")
	if ok, err := isAllowExecName(execer); !ok {
		fmt.Println(err.Error())
		return
	}
	addrResult := account.ExecAddress(execer)
	result := addrResult
	fmt.Println(result)
}

// create user data
func UserDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "userdata",
		Short: "Write data to user defined executor",
		Run:   addUserData,
	}
	addUserDataFlags(cmd)
	return cmd
}
func addUserDataFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("exec", "e", "", `executor name must contain prefix "user"`)
	cmd.MarkFlagRequired("exec")
	cmd.Flags().StringP("topic", "t", "", `add #topic# before data, if empty add nothing`)
	cmd.Flags().StringP("data", "d", "", `payload data that users want to write`)
	cmd.MarkFlagRequired("data")
}

func addUserData(cmd *cobra.Command, args []string) {
	execer, err := cmd.Flags().GetString("exec")
	if err != nil {
		fmt.Println(err)
		return
	}
	if !strings.HasPrefix(execer, "user.") {
		fmt.Println(`user defined executor should start with "user."`)
		return
	}
	if len(execer) > 50 {
		fmt.Println("executor name too long")
		return
	}
	addrResult := account.ExecAddress(execer)
	topic, _ := cmd.Flags().GetString("topic")
	data, _ := cmd.Flags().GetString("data")
	if topic != "" {
		data = "#" + topic + "#" + data
	}
	if data == "" {
		fmt.Println("write empty data")
		return
	}
	tx := &types.Transaction{
		Execer:  []byte(execer),
		Payload: []byte(data),
		To:      addrResult,
	}
	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	//tx.Sign(int32(wallet.SignType), privKey)
	txHex := types.Encode(tx)
	fmt.Println(hex.EncodeToString(txHex))
}
