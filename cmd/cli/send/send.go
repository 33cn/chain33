package main

import (
	"os/exec"
	"os"
	"fmt"
	"bytes"
)

func main() {
	argsWithoutProg := os.Args[1:]
	if argsWithoutProg[0] == "help" || argsWithoutProg[0] == "-h" {
		loadHelp()
		return
	}
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

	cmdCreate := exec.Command("cli", argsWithoutProg...)
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
	}
	//fmt.Println("unsignedTx", outCreate.String(), errCreate.String())

	if !hasKey || key == ""{
		fmt.Fprintln(os.Stderr, "no private key found")
		return
	}
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
	if errSign.String() != "" {
		fmt.Println(errSign.String())
	}
	//fmt.Println("signedTx", outSign.String(), errSign.String())

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
	if errSend.String() != "" {
		fmt.Println(errSend.String())
	}
	fmt.Println(outSend.String())
}

func loadHelp() {
	fmt.Println("Use similarly as bty/token/trade raw transaction creation, in addition to the parameter of private key input following \"-k\".")
}
