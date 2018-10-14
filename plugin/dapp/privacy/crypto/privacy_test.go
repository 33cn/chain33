package privacy

/*
func TestPrivacyOnetimeKey(t *testing.T) {
	t.Logf("Begin to do TestPrivacyOnetimeKey\n")

	priKey := "BC7621CE84D3D67851326C360B518DB5"
	pribyte, _ := common.Hex2Bytes(priKey)
	priaddr32 := (*[32]byte)(unsafe.Pointer(&pribyte[0]))
	privacyInfo, err := NewPrivacyWithPrivKey(priaddr32)
	if err != nil {
		t.Errorf("Failed to NewPrivacyWithPrivKey", "err info is", err)
		return
	}

	t.Logf("viewprikey:%X, viewpubkey:%X\n", privacyInfo.ViewPrivKey, privacyInfo.ViewPubkey)
	t.Logf("spendprikey:%X, spendpubkey:%X\n", privacyInfo.SpendPrivKey, privacyInfo.SpendPubkey)

	viewPublic := (*[32]byte)(unsafe.Pointer(&privacyInfo.ViewPubkey[0]))
	spendPublic := (*[32]byte)(unsafe.Pointer(&privacyInfo.SpendPubkey[0]))
	pubkeyOnetime, txPublicKey, err := GenerateOneTimeAddr(viewPublic, spendPublic)
	if err != nil {
		t.Errorf("Failed to GenerateOneTimeAddr")
		return
	}
	t.Logf("The generated pubkeyOnetime: %X \n", pubkeyOnetime[:])
	t.Logf("The generated txPublicKey:   %X \n", txPublicKey[:])

	onetimePriKey, err := RecoverOnetimePriKey(txPublicKey[:], privacyInfo.ViewPrivKey, privacyInfo.SpendPrivKey)
	if err != nil {
		t.Errorf("Failed to RecoverOnetimePriKey")
		return
	}
	t.Logf("The recovered one time privicy key is:%X", onetimePriKey.Bytes())

	recoverPub := onetimePriKey.PubKey().Bytes()[:]
	originPub := pubkeyOnetime[:]
	t.Logf("****￥￥￥*****The recoverPub key is:%X", recoverPub)
	t.Logf("****￥￥￥*****The originPub  key is:%X", originPub)


	if !bytes.Equal(recoverPub, originPub) {
		t.Failed()
		t.Errorf("recoverPub is not equal to originPub")
		return

	} else {
		t.Logf("Yea!!! Succeed to do the TestPrivacyOnetimeKey.")
	}

	t.Logf("End to do TestPrivacyOnetimeKey\n")
}
*/

/*
// TODO: 需要增加隐私签名的UT
func TestPrivacySignWithFixInput(t *testing.T) {
	prislice, _ := common.Hex2Bytes("9E0ED368F3DDAA9F472FE7F319F866227A74A2EF16B43410CEB3CE7C1BAAEB09")
	var onetimePriKey PrivKeyPrivacy
	copy(onetimePriKey[:], prislice)

	recoverPub := onetimePriKey.PubKey().Bytes()[:]

	data := []byte("Yea!!! Succeed to do the TestPrivacyOnetimeKey")
	sig := onetimePriKey.Sign(data)
	sign := &types.Signature{
		Ty:        4,
		Pubkey:    recoverPub,
		Signature: sig.Bytes(),
	}

	c := &OneTimeEd25519{}

	pub, err := c.PubKeyFromBytes(sign.Pubkey)
	if err != nil {
		t.Failed()
		t.Errorf("Failed to PubKeyFromBytes")
		return
	}
	signbytes, err := c.SignatureFromBytes(sign.Signature)
	if err != nil {
		t.Failed()
		t.Errorf("Failed to SignatureFromBytes")
		return
	}

	if pub.VerifyBytes(data, signbytes) {
		t.Logf("Yea!!! Succeed to pass CheckSign.")

	} else {
		t.Failed()
		t.Errorf("Fail the CheckSign")
		return

	}

	t.Logf("End to do TestPrivacyOnetimeKey\n")
}
*/

//func TestPrivacySign(t *testing.T) {
//	t.Logf("Begin to do TestPrivacyOnetimeKey\n")
//
//	priKey := "BC7621CE84D3D67851326C360B518DB5"
//	pribyte, _ := common.Hex2Bytes(priKey)
//	priaddr32 := (*[32]byte)(unsafe.Pointer(&pribyte[0]))
//	privacyInfo, err := NewPrivacyWithPrivKey(priaddr32)
//	if err != nil {
//		t.Errorf("Failed to NewPrivacyWithPrivKey", "err info is", err)
//		return
//	}
//
//	t.Logf("viewprikey:%X, viewpubkey:%X\n", privacyInfo.ViewPrivKey, privacyInfo.ViewPubkey)
//	t.Logf("spendprikey:%X, spendpubkey:%X\n", privacyInfo.SpendPrivKey, privacyInfo.SpendPubkey)
//
//	viewPublic := (*[32]byte)(unsafe.Pointer(&privacyInfo.ViewPubkey[0]))
//	spendPublic := (*[32]byte)(unsafe.Pointer(&privacyInfo.SpendPubkey[0]))
//	pubkeyOnetime, txPublicKey, err := privacyInfo.GenerateOneTimeAddr(viewPublic, spendPublic)
//	if err != nil {
//		t.Errorf("Failed to GenerateOneTimeAddr")
//		return
//	}
//	t.Logf("The generated pubkeyOnetime: %X \n", pubkeyOnetime[:])
//	t.Logf("The generated txPublicKey:   %X \n", txPublicKey[:])
//
//	onetimePriKey, err := privacyInfo.RecoverOnetimePriKey(txPublicKey[:], privacyInfo.ViewPrivKey, privacyInfo.SpendPrivKey)
//	if err != nil {
//		t.Errorf("Failed to RecoverOnetimePriKey")
//		return
//	}
//	t.Logf("The recovered one time privicy key is:%X", onetimePriKey.Bytes())
//
//	recoverPub := onetimePriKey.PubKey().Bytes()[:]
//	originPub := pubkeyOnetime[:]
//	t.Logf("****￥￥￥*****The recoverPub key is:%X", recoverPub)
//	t.Logf("****￥￥￥*****The originPub  key is:%X", originPub)
//
//
//	if !bytes.Equal(recoverPub, originPub) {
//		t.Failed()
//		t.Errorf("recoverPub is not equal to originPub")
//		return
//
//	} else {
//		t.Logf("Yea!!! Succeed to do the TestPrivacyOnetimeKey.")
//	}
//    data := []byte("Yea!!! Succeed to do the TestPrivacyOnetimeKey")
//	sig := onetimePriKey.Sign(data)
//	sign := &types.Signature{
//		Ty: 4,
//		Pubkey: recoverPub,
//		Signature:sig.Bytes(),
//	}
//
//	c := &OneTimeEd25519{}
//
//	pub, err := c.PubKeyFromBytes(sign.Pubkey)
//	if err != nil {
//		t.Failed()
//		t.Errorf("Failed to PubKeyFromBytes")
//		return
//	}
//	signbytes, err := c.SignatureFromBytes(sign.Signature)
//	if err != nil {
//		t.Failed()
//		t.Errorf("Failed to SignatureFromBytes")
//		return
//	}
//
//	if pub.VerifyBytes(data, signbytes) {
//		t.Logf("Yea!!! Succeed to pass CheckSign.")
//
//	} else {
//		t.Failed()
//		t.Errorf("Fail the CheckSign")
//		return
//
//	}
//
//	t.Logf("End to do TestPrivacyOnetimeKey\n")
//}
