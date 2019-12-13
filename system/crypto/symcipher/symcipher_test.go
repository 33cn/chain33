package symcipher

import (
	"fmt"
	"testing"
)

func Test_SymCipher(t *testing.T) {
	symKey, _ := GenerateSymKey()
	fmt.Printf("The generated key is:%s, len:%d\n", string(symKey), len(symKey))

	plainTxt := "hello chain33"
	cipherData, _ := EncryptSymmetric(symKey, []byte(plainTxt))
	fmt.Printf("The cipher info is:%s, len:%d\n", string(cipherData), len(cipherData))

	decryptRes, _ := DecryptSymmetric(symKey, cipherData)
	fmt.Printf("The decrypted resuls is:%s\n", string(decryptRes))

	if plainTxt != string(decryptRes) {
		fmt.Printf("The decrypted resuls is:%s, and the origin is:%s\n", string(decryptRes), plainTxt)
		t.Fail()
	}
}
