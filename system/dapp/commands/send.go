// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/33cn/chain33/common/address"
)

// OneStepSendCmd send cmd
func OneStepSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "send",
		Short:              "send tx in one step",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			oneStepSend(cmd, os.Args[0], args)
		},
	}
	cmd.Flags().StringP("key", "k", "", "private key or from address for sign tx")
	cmd.Flags().Int32P("signAddrType", "", -1, "sign address type ID, btc(0), btcMultiSign(1), eth(2)")
	return cmd
}

// one step send
func oneStepSend(cmd *cobra.Command, cmdName string, params []string) {
	if len(params) < 1 || params[0] == "help" || params[0] == "--help" || params[0] == "-h" {
		loadSendHelp()
		return
	}

	var createParams, keyParams []string
	//取出send命令的key参数, 保留原始构建的参数列表
	nextIgnore := false
	for i, v := range params {
		if strings.HasPrefix(v, "-k=") || strings.HasPrefix(v, "--key=") ||
			strings.HasPrefix(v, "--signAddrType=") {
			keyParams = append(keyParams, v)
			continue
		} else if (v == "-k" || v == "--key" || v == "--signAddrType") && i < len(params)-1 {
			keyParams = append(keyParams, v, params[i+1])
			nextIgnore = true
			continue
		}

		if nextIgnore {
			nextIgnore = false
			continue
		}
		createParams = append(createParams, v)
	}
	//创建交易命令
	cmdCreate := exec.Command(cmdName, createParams...)
	createRes, err := execCmd(cmdCreate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	//cli不传任何参数不会报错, 输出帮助信息
	if strings.Contains(createRes, "\n") {
		fmt.Println(createRes)
		return
	}

	//调用send命令parse函数解析key参数
	err = cmd.Flags().Parse(keyParams)
	key, _ := cmd.Flags().GetString("key")
	if len(key) <= 0 || err != nil {
		loadSendHelp()
		fmt.Fprintln(os.Stderr, "Error: required flag(s) \"key\" not proper set")
		return
	}
	addrTy, _ := cmd.Flags().GetInt32("signAddrType")

	//构造签名命令的key参数
	if address.CheckAddress(key, -1) == nil {
		keyParams = append([]string{}, "-a", key)
	} else {
		keyParams = append([]string{}, "-k", key, "-p", fmt.Sprintf("%d", addrTy))
	}

	//采用内部的构造交易命令,解析rpc_laddr地址参数
	createCmd, createFlags, _ := cmd.Root().Traverse(createParams)
	_ = createCmd.ParseFlags(createFlags)
	rpcAddr, _ := createCmd.Flags().GetString("rpc_laddr")
	//交易签名命令调用
	signParams := []string{"wallet", "sign", "-d", createRes, "--rpc_laddr", rpcAddr}
	cmdSign := exec.Command(cmdName, append(signParams, keyParams...)...)
	signRes, err := execCmd(cmdSign)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	//交易发送命令调用
	sendParams := []string{"wallet", "send", "-d", signRes, "--rpc_laddr", rpcAddr}
	cmdSend := exec.Command(cmdName, sendParams...)
	sendRes, err := execCmd(cmdSend)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(sendRes)
}

func execCmd(c *exec.Cmd) (string, error) {
	var outBuf, errBuf bytes.Buffer
	c.Stderr = &errBuf
	c.Stdout = &outBuf

	if err := c.Run(); err != nil {
		return "", errors.New(err.Error() + "\n" + errBuf.String())
	}
	if len(errBuf.String()) > 0 {
		return "", errors.New(errBuf.String())
	}
	outBytes := outBuf.Bytes()
	return string(outBytes[:len(outBytes)-1]), nil
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
  -h, --help                 help for send
  -k, --key string           address or private key for sign tx, required
      --signAddrType int32   sign address type ID, btc(0), btcMultiSign(1), eth(2)`
	fmt.Println(help)
}
