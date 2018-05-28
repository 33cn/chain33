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
	keyIndex := -1
	hasRpc := false
	var key string
	var rpcAddr string
	size := len(params)

	// 先查找需要的参数并记录索引
	for i, v := range params {
		if v == "-k" {
			hasKey = true
			if i < size-1 {
				key = params[i+1]
				keyIndex = i
			} else {
				fmt.Fprintln(os.Stderr, "no private key found")
				return
			}
		} else if v == "--rpc_laddr" {
			hasRpc = true
			if i < size-1 {
				rpcAddr = params[i+1]
			}
		}
	}
	// 再修改参数数组，将不需要的参数过滤掉
	params = append(params[:keyIndex], params[keyIndex+2:]...)

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
	cParams := []string{"wallet", "sign", "-d", string(bufCreate[:len(bufCreate)-1]), addrOrKey, key}
	if hasRpc {
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
	if hasRpc {
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
