package commands

import (
	"math/rand"
	"github.com/spf13/cobra"
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	ttypes "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
	"time"
	"strconv"
)

var (
	tendermintlog = log15.New("module", "tendermint")
	strChars      = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" // 62 characters
	genFile       = "genesis_file.json"
	pvFile        = "priv_validator_"
)

func KeyFileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keyfile",
		Short: "Initialize Tendermint Keyfile",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(
		CreateCmd(),
	)
	return cmd
}

func CreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Begin to create keyfile",
		Run:   createFiles,
	}
	addCreateCmdFlags(cmd)
	return cmd
}

func addCreateCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("num", "n", "", "Num of the keyfile to create")
	cmd.MarkFlagRequired("num")
}

func addCleanCmdFlags(cmd *cobra.Command) {
}

func RandStr(length int) string {
	chars := []byte{}
MAIN_LOOP:
	for {
		val := rand.Int63()
		for i := 0; i < 10; i++ {
			v := int(val & 0x3f) // rightmost 6 bits
			if v >= 62 { // only 62 characters in strChars
				val >>= 6
				continue
			} else {
				chars = append(chars, strChars[v])
				if len(chars) == length {
					break MAIN_LOOP
				}
				val >>= 6
			}
		}
	}

	return string(chars)
}

func initCryptoImpl() error {
	cr, err := crypto.New(types.GetSignatureTypeName(types.ED25519))
	if err != nil {
		tendermintlog.Error("New crypto impl failed", "err", err)
		return err
	}
	ttypes.ConsensusCrypto = cr
	return nil
}

func createFiles(cmd *cobra.Command, args []string) {
	// init crypto instance
	err := initCryptoImpl()
	if err != nil {
		return
	}

	// genesis file
	genDoc := ttypes.GenesisDoc{
		ChainID:     ttypes.Fmt("chain33-%v", RandStr(6)),
		GenesisTime: time.Now(),
	}

	num, _ := cmd.Flags().GetString("num")
	n, _ := strconv.Atoi(num)
	for i := 0; i < n; i ++ {
		// create private validator file
		pvFileName := pvFile + strconv.Itoa(i) + ".json"
		privValidator := ttypes.LoadOrGenPrivValidatorFS(pvFileName)
		if privValidator == nil {
			tendermintlog.Error("Create priv_validator file failed.")
			break
		}

		// create genesis validator by the pubkey of private validator
		gv := ttypes.GenesisValidator{
			PubKey: ttypes.KeyText{"ed25519", privValidator.GetPubKey().KeyString()},
			Power:  10,
		}
		genDoc.Validators = append(genDoc.Validators, gv)
	}

	if err := genDoc.SaveAs(genFile); err != nil {
		tendermintlog.Error("Generated genesis file failed.")
		return
	}
	tendermintlog.Info("Generated genesis file", "path", genFile)

	return
}
