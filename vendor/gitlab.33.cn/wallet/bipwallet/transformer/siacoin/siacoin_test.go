package siacoin

import (
	"encoding/hex"
	"testing"

	"gitlab.33.cn/wallet/bipwallet/transformer"
)

var testSeed = "corrode cabin meeting skew wickets token sequence lofty daytime having austere gels revamp magically obedient theatrics loaded upon erosion fizzle mural love muffin jeopardy autumn comb hatchet ouch"

var testPriv = "c981cc0e5b8da022cac784f4b119491e1a012526e18f0c218f7b91172fc086a54968be8c06c61c6e9c2787df13c5b6e7910ea97d682f1bf4a061308c7f42b641"

var testPub = "4968be8c06c61c6e9c2787df13c5b6e7910ea97d682f1bf4a061308c7f42b641"

var ansAddr = "ceb4110945f5044bfb370b308c871e4f1cda071ba05af6782da27b610ad329a87f0856896ff1"

//测试seed生成私钥及公钥地址转换
func TestScTransformer(t *testing.T) {
	//通过种子生成密钥
	sk, pk, err := SeedToSecretkey(testSeed, 1)
	if err != nil {
		t.Errorf("seed to secret key error: %s", err)
	}
	t.Logf("secret key: %s", sk)
	t.Logf("public key: %s", pk)

	//转成字节形式
	priv, err := hex.DecodeString(sk)
	if err != nil {
		t.Errorf("decode hex privkey error: %s", err)
	}
	pub, err := hex.DecodeString(pk)
	if err != nil {
		t.Errorf("decode hex pubkey error: %s", err)
	}

	//新建transformer
	scTrans, err := transformer.New("SC")
	if err != nil {
		t.Errorf("new sc transformer error: %s", err)
	}

	//测试私钥生成公钥
	genPubkey, err := scTrans.PrivKeyToPub(priv)
	if err != nil {
		t.Errorf("SC PrivKeyToPub error: %s", err)
	}
	if hex.EncodeToString(genPubkey) == pk {
		t.Logf("public key match: %x", genPubkey)
	} else {
		t.Errorf("public key mismatch：want： %s have： %x", pk, genPubkey)
	}

	//测试公钥生成地址
	genAddr, err := scTrans.PubKeyToAddress(pub)
	if err != nil {
		t.Errorf("SC PubKeyToAddress error: %s", err)
	}
	if genAddr == ansAddr {
		t.Logf("address match: %s", genAddr)
	} else {
		t.Errorf("address mismatch: want:  %s have:  %s", ansAddr, genAddr)
	}
}
