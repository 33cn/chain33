package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"code.aliyun.com/chain33/chain33/common/crypto"
	_ "code.aliyun.com/chain33/chain33/common/crypto/sm2"
)

var pubKeyStr = flag.String("p", "", "pubkey(hex)")
var msgFile = flag.String("m", "", "msg file to read")
var signStr = flag.String("s", "", "signature(hex) of msg")

/*
pubkey 的 16 进制字符串:
77dac51899bbdf4fdf3b7b9ba7f931f1e03cc2718894ae20cb823ecc9e5b5c2a170f4345b75c591b13c810d812bd7bd00d42f275cac6fb2ce9546eda3f2d25b3

msg 文件原始内容(共226字节):
<?xml version="1.0" encoding="utf-8"?><T><D><M><k>Remitter Name:</k><v>Alan</v></M><M><k>Remitter Account:</k><v>1234567890123456</v></M><M><k>Amount:</k><v>123.23</v></M></D><E><M><k>Trade ID:</k><v>1234567890</v></M></E></T>

sm2 sign 的 16 进制字符串：
3eddc5b028c4196fd5bb9ecfad0b0c933421126a183c39af81dbe0e391baa8985c7bc896ce4ae81a44bec94c1493a719d504bce6076380c9b36fa0e3aa3f10db
*/
func main() {
	if len(os.Args) <= 6 {
		fmt.Printf("Usage: %s -p pub -s sign -m msg_file\n", os.Args[0])
		os.Exit(0)
	}

	flag.Parse()

	pubBytes, err := hex.DecodeString(*pubKeyStr)
	if err != nil {
		panic(err)
	}

	signBytes, err := hex.DecodeString(*signStr)
	if err != nil {
		panic(err)
	}

	msgBytes, err := ioutil.ReadFile(*msgFile)
	if err != nil {
		panic(err)
	}

	c, err := crypto.New("sm2")
	if err != nil {
		panic(err)
	}

	pub, err := c.PubKeyFromBytes(pubBytes)
	if err != nil {
		panic(err)
	}

	sign, err := c.SignatureFromBytes(signBytes)
	if err != nil {
		panic(err)
	}

	ok := pub.VerifyBytes(msgBytes, sign)
	fmt.Println("signature is OK?", ok)
}
