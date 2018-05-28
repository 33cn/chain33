package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"gitlab.33.cn/chain33/chain33/account"
)

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
	var isAddr bool
	err := account.CheckAddress(key)
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
	//fmt.Println("unsignedTx", outCreate.String(), errCreate.String())

	if !hasKey || key == "" {
		fmt.Fprintln(os.Stderr, "no private key found")
		return
	}
	bufCreate := outCreate.Bytes()

	addrOrKey := "-k"
	if isAddr {
		addrOrKey = "-a"
	}
	cmdSign := exec.Command(name, "wallet", "sign", "-d", string(bufCreate[:len(bufCreate)-1]), addrOrKey, key)
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
	cmdSend := exec.Command(name, "wallet", "send", "-d", string(bufSign[:len(bufSign)-1]))
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
