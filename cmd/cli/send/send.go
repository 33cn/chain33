package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	argsWithoutProg := os.Args[1:]
	hasKey := false
	var key string
	size := len(argsWithoutProg)
	for i, v := range argsWithoutProg {
		if v == "-k" {
			hasKey = true
			if i < size-1 {
				key = argsWithoutProg[i+1]
				argsWithoutProg = append(argsWithoutProg[:i], argsWithoutProg[i+2:]...)
			} else {
				fmt.Fprintln(os.Stderr, "no private key found")
				return
			}
		}
	}
	if !hasKey || key == "" {
		fmt.Fprintln(os.Stderr, "no private key found")
		return
	}

	cmdCreate := exec.Command("cli", argsWithoutProg...)
	var outCreate bytes.Buffer
	var errCreate bytes.Buffer
	cmdCreate.Stdout = &outCreate
	cmdCreate.Stderr = &errCreate
	err := cmdCreate.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Println("unsignedTx", outCreate.String(), errCreate.String())

	bufCreate := outCreate.Bytes()
	cmdSign := exec.Command("cli", "wallet", "sign", "-d", string(bufCreate[:len(bufCreate)-1]), "-k", key)
	var outSign bytes.Buffer
	var errSign bytes.Buffer
	cmdSign.Stdout = &outSign
	cmdSign.Stderr = &errSign
	err = cmdSign.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Println("signedTx", outSign.String(), errSign.String())

	bufSign := outSign.Bytes()
	cmdSend := exec.Command("cli", "tx", "send", "-d", string(bufSign[1:len(bufSign)-2]))
	var outSend bytes.Buffer
	var errSend bytes.Buffer
	cmdSend.Stdout = &outSend
	cmdSend.Stderr = &errSend
	err = cmdSend.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	fmt.Println("sentTx", outSend.String(), errSend.String())
}
