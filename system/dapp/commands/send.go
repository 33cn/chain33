// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/33cn/chain33/common/address"
)

// OneStepSend one step send
func OneStepSend(args []string) {
	name := args[0]
	params := args[2:]
	if len(params) < 1 {
		loadHelp()
		return
	}

	if params[0] == "help" || params[0] == "--help" || params[0] == "-h" {
		loadHelp()
		return
	}
	hasKey := false
	var key string
	size := len(params)
	for i, v := range params {
		if v == "-k" {
			hasKey = true
			if i < size-1 {
				key = params[i+1]
				params = append(params[:i], params[i+2:]...)
			} else {
				fmt.Fprintln(os.Stderr, "no private key found")
				return
			}
		}
	}
	hasAddr := false
	var rpcAddr string
	size = len(params)
	for i, v := range params {
		if v == "--rpc_laddr" {
			hasAddr = true
			if i < size-1 {
				rpcAddr = params[i+1]
				params = append(params[:i], params[i+2:]...)
			}
		}
	}
	var isAddr bool
	err := address.CheckAddress(key)
	if err != nil {
		isAddr = false
	} else {
		isAddr = true
	}

	cmdCreate := exec.Command(name, params...)
	var outCreate bytes.Buffer
	var errCreate bytes.Buffer
	cmdCreate.Stdout = &outCreate
	cmdCreate.Stderr = &errCreate
	err = cmdCreate.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if errCreate.String() != "" {
		fmt.Println(errCreate.String())
		return
	}

	if !hasKey || key == "" {
		fmt.Fprintln(os.Stderr, "no private key found")
		return
	}
	bufCreate := outCreate.Bytes()
	addrOrKey := "-k"
	if isAddr {
		addrOrKey = "-a"
	}
	cParams := []string{"wallet", "sign", "-d", string(bufCreate[:len(bufCreate)-1]), addrOrKey, key}
	if hasAddr {
		cParams = append(cParams, "--rpc_laddr", rpcAddr)
	}
	cmdSign := exec.Command(name, cParams...)
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
	cParams = []string{"wallet", "send", "-d", string(bufSign[:len(bufSign)-1])}
	if hasAddr {
		cParams = append(cParams, "--rpc_laddr", rpcAddr)
	}
	cmdSend := exec.Command(name, cParams...)
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

func loadHelp() {
	fmt.Println("Use similarly as bty/token/trade/bind_miner raw transaction creation, in addition to the parameter of private key or from address input following \"-k\".")
	fmt.Println("e.g.: cli send bty transfer -a 1 -n note -t toAddr -k privKey/fromAddr")
}
