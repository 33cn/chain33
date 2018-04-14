package privacy

import (
	"testing"
	"unsafe"
	"bytes"
)

func TestPrivacyOnetimeKey(t *testing.T) {
	t.Logf("Begin to do TestPrivacyOnetimeKey\n")

	//receiver's key-pair
	viewprikey := PrivacyProc.GenPriKey()
	viewPublic := PrivacyProc.MakePublicKey((*[32]byte)(unsafe.Pointer(&viewprikey.Bytes()[0])))
	//viewpubkey := viewprikey.PubKey()
	spendprikey := PrivacyProc.GenPriKey()
	spendPublic := PrivacyProc.MakePublicKey((*[32]byte)(unsafe.Pointer(&spendprikey.Bytes()[0])))

	//viewPublic := (*[32]byte)(unsafe.Pointer(&viewpubkey.Bytes()[0]))
	//spendPublic := (*[32]byte)(unsafe.Pointer(&spendpubkey.Bytes()[0]))

    t.Logf("viewprikey:%X, viewpubkey:%X\n", viewprikey.Bytes(), viewPublic[:])
	t.Logf("spendprikey:%X, spendpubkey:%X\n", spendprikey.Bytes(), spendPublic[:])

	pubkeyOnetime, txPublicKey, err := PrivacyProc.GenerateOneTimeAddr(viewPublic, spendPublic)
	if err != nil {
		t.Errorf("Failed to GenerateOneTimeAddr")
		return
	}
	//addrOneTime := account.PubKeyToAddress(pubkeyOnetime[:]).String()
	t.Logf("The generated pubkeyOnetime: %X \n", pubkeyOnetime[:])
	t.Logf("The generated txPublicKey:   %X \n", txPublicKey[:])

	onetimePriKey, err := PrivacyProc.RecoverOnetimePriKey(txPublicKey[:], viewprikey, spendprikey)
	if err != nil {
		t.Errorf("Failed to RecoverOnetimePriKey")
		return
	}
	t.Logf("The recovered one time privicy key is:%X", onetimePriKey.Bytes())

	recoverPub := PrivacyProc.MakePublicKey((*[32]byte)(unsafe.Pointer(&onetimePriKey.Bytes()[0])))[:]
	originPub := pubkeyOnetime[:]
	t.Logf("****￥￥￥*****The recoverPub key is:%X", recoverPub)
	t.Logf("****￥￥￥*****The originPub  key is:%X", originPub)

	//recoverPub := onetimePriKey.PubKey()
	//originPub, err := ed25519Driver.PubKeyFromBytes(pubkeyOnetime[:])
	//if err != nil {
	//	t.Errorf("Failed to ed25519Driver.PubKeyFromBytes")
	//}
	if !bytes.Equal(recoverPub, originPub) {
	//if !recoverPub.Equals(originPub){
		t.Failed()
		t.Errorf("recoverPub is not equal to originPub")
		return

	} else {
		t.Logf("Yea!!! Succeed to do the TestPrivacyOnetimeKey.")
	}

	t.Logf("End to do TestPrivacyOnetimeKey\n")
}
