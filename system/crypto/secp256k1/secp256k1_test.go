package secp256k1

import (
	"fmt"
	"os"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
)

var driver Driver

func Test_Cipher(t *testing.T) {
	fmt.Printf("Begin to test cipher by secp256k1\n")
	privateKey, _ := driver.GenKey()
	privateKeyBytes := privateKey.Bytes()
	fmt.Fprintf(os.Stdout, "The new generated private key is:%s, len:%d\n", common.ToHex(privateKeyBytes), len(privateKeyBytes))
	publicKey := privateKey.PubKey()
	publicKeyBytes := publicKey.Bytes()
	fmt.Fprintf(os.Stdout, "The new generated pubkic key is:%s, len:%d\n", common.ToHex(publicKeyBytes), len(publicKeyBytes))

	plainTxt := "hello chain33"
	cipherTxt, _ := publicKey.Encrypt([]byte(plainTxt))
	fmt.Fprintf(os.Stdout, "The ciphered text:%s\n", string(cipherTxt[:len(cipherTxt)-1]))

	decryptedTxt, _ := privateKey.Decrypt(cipherTxt)
	fmt.Fprintf(os.Stdout, "The decrypted text:%s\n", string(decryptedTxt))

	if string(decryptedTxt) != plainTxt {
		t.Fail()
	}
	t.Logf("result:%s", "Success")
}

func Test_EncryptBySecp256k1(t *testing.T) {
	fmt.Printf("Begin to Test_EncryptBySecp256k1\n")
	priKeyStr := "c34b5d9d44ac7b754806f761d3d4d2c4fe5214f6b074c19f069c4f5c2a29c8cc"
	priKeyBYte, _ := common.FromHex(priKeyStr)
	fmt.Fprintf(os.Stdout, "The imported private key with len:%d\n", len(priKeyBYte))

	privateKey, err := driver.PrivKeyFromBytes(priKeyBYte)
	if nil != err {
		fmt.Fprintf(os.Stdout, "PrivKeyFromBytes error due to:%s\n", err.Error())
		t.Fail()
		return
	}
	publicKey := privateKey.PubKey()
	publicKeyBytes := publicKey.Bytes()
	fmt.Fprintf(os.Stdout, "The new generated pubkic key is:%s, len:%d\n", common.ToHex(publicKeyBytes), len(publicKeyBytes))
	addr := address.PubKeyToAddr(publicKeyBytes)
	fmt.Fprintf(os.Stdout, "The address is:%s\n", addr)

	addActual := "1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK"
	if addActual != addr {
		fmt.Fprintf(os.Stdout, "PrivKeyFromBytes error due to:%s\n", err.Error())
		t.Fail()
		return
	}

	plainTxt := "hello chain33"
	cipherTxt, _ := publicKey.Encrypt([]byte(plainTxt))
	fmt.Fprintf(os.Stdout, "The ciphered text:%s\n", string(cipherTxt[:len(cipherTxt)-1]))

	decryptedTxt, _ := privateKey.Decrypt(cipherTxt)
	fmt.Fprintf(os.Stdout, "The decrypted text:%s\n", string(decryptedTxt))

	if string(decryptedTxt) != plainTxt {
		t.Fail()
	}
}
