// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/33cn/chain33/common/address"
)

func OneStepSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "send",
		Short:              "send tx in one step",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			oneStepSend(os.Args[0], args)
		},
	}
	//cmd.Flags().StringP("key", "k", "", "private key or from address for sign tx")
	return cmd
}

// one step send
func oneStepSend(cmd string, params []string) {
	if len(params) < 1 || params[0] == "help" || params[0] == "--help" || params[0] == "-h" {
		loadSendHelp()
		return
	}
	keyIndex := -1
	key := ""
	var rpcAddrParams, keyParams []string
	for i, v := range params {
		if i == len(params)-1 {
			break
		}
		if v == "-k" || v == "--key" {
			keyIndex = i
			key = params[i+1]
		} else if v == "--rpc_laddr" {
			rpcAddrParams = append(rpcAddrParams, "--rpc_laddr", params[i+1])
		}
	}
	if key == "" {
		loadSendHelp()
		fmt.Println("Error: required flag(s) \"key\" not set")
		return
	}
	params = append(params[:keyIndex], params[keyIndex+2:]...) //remove key param
	if address.CheckAddress(key) != nil {
		keyParams = append(keyParams, "-k", key)
	} else {
		keyParams = append(keyParams, "-a", key)
	}

	cmdCreate := exec.Command(cmd, params...)
	var outCreate bytes.Buffer
	var errCreate bytes.Buffer
	cmdCreate.Stdout = &outCreate
	cmdCreate.Stderr = &errCreate
	err := cmdCreate.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if errCreate.String() != "" {
		fmt.Println(errCreate.String())
		return
	}
	bufCreate := outCreate.Bytes()
	signParams := []string{"wallet", "sign", "-d", string(bufCreate[:len(bufCreate)-1])}
	cmdSign := exec.Command(cmd, append(append(signParams, keyParams...), rpcAddrParams...)...)
	var outSign bytes.Buffer
	var errSign bytes.Buffer
	cmdSign.Stdout = &outSign
	cmdSign.Stderr = &errSign
	err = cmdSign.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if errSign.String() != "" {
		fmt.Println(errSign.String())
		return
	}
	//fmt.Println("signedTx", outSign.String(), errSign.String())

	bufSign := outSign.Bytes()
	sendParams := []string{"wallet", "send", "-d", string(bufSign[:len(bufSign)-1])}
	cmdSend := exec.Command(cmd, append(sendParams, rpcAddrParams...)...)
	var outSend bytes.Buffer
	var errSend bytes.Buffer
	cmdSend.Stdout = &outSend
	cmdSend.Stderr = &errSend
	err = cmdSend.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if errSend.String() != "" {
		fmt.Println(errSend.String())
		return
	}
	bufSend := outSend.Bytes()
	fmt.Println(string(bufSend[:len(bufSend)-1]))
}

func loadSendHelp() {
	help := `[Integrate create/sign/send transaction operations in one command]
Usage:
  -cli send [flags]

Examples:
cli send coins transfer -a 1 -n note -t toAddr -k [privateKey | fromAddr]

equivalent to three steps: 
1. cli coins transfer -a 1 -n note -t toAddr   //create raw tx
2. cli wallet sign -d rawTx -k privateKey      //sign raw tx
3. cli wallet send -d signTx                   //send tx to block chain

Flags:
  -h, --help         help for send
  -k, --key string   private key or from address for sign tx, required

Global Flags:
      --paraName string    parachain
      --rpc_laddr string   http url (default "https://localhost:8801")`
	fmt.Println(help)
}
