package main

import (
	"fmt"
	"gosm2"
)

func main() {
	priv, err := gosm2.NewKey()
	if err != nil {
		panic(err)
	}

	fmt.Println("private key: ", priv.EncodeString())
	fmt.Println("public key: ", priv.PublicKey.EncodeString())
}
