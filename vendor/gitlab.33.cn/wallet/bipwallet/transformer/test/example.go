//模块的样例程序，以DCR为例
package main

import (
	"fmt"

	"gitlab.33.cn/wallet/bipwallet/transformer"
	_ "gitlab.33.cn/wallet/bipwallet/transformer/decred"
)

var testPrivkey = []byte{
	0xba, 0x6a, 0xf2, 0xa1, 0xf0, 0x15, 0x0d, 0xb3, 0xa8, 0x74,
	0x5c, 0x2a, 0xdb, 0xdf, 0x98, 0xe9, 0x28, 0xc6, 0x26, 0x4d,
	0xd0, 0xaa, 0x86, 0x6e, 0x59, 0xb0, 0x6f, 0xa6, 0x89, 0x74,
	0x0f, 0x05}

func main() {
	//New对应币种的Transformer
	t, err := transformer.New("DCR")
	if err != nil {
		fmt.Errorf("new transformer error: %s\n", err)
		return
	}
	//私钥生成公钥
	pub, err := t.PrivKeyToPub(testPrivkey)
	if err != nil {
		fmt.Errorf("generate public key error: %s\n", err)
		return
	}
	fmt.Printf("public key is: %x\n", pub)
	//公钥生成地址
	addr, err := t.PubKeyToAddress(pub)
	if err != nil {
		fmt.Errorf("generate address error: %s\n", err)
		return
	}
	fmt.Printf("address is: %s\n", addr)
}
