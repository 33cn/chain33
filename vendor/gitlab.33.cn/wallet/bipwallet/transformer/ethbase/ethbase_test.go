package ethbase

import (
	"encoding/hex"
	"testing"

	"gitlab.33.cn/wallet/bipwallet/transformer"
)

var testPrivkey = map[string]string{
	"ETH": "a9107ad67d50e3a574e9803bd5c946ecb4011388345a98243aa07c329bee383d",
	"ETC": "bf44112c411558a5d77107c9d3e82afb094dfdf07559526bd9087e6fcf0783c1",
}

var ansPubkey = map[string]string{
	"ETH": "0331c9177b96cf81b38393e747415a1f2484d78970f41ead62b8bc430903a01229",
	"ETC": "0251877cc8de7c3c12a2e4cdb246d44ca193651373f1f51e61fdc66c24a8d79a12",
}

var ansAddress = map[string]string{
	"ETH": "0xf9c6b65c1996C38632C16F7F15D34252B647dfb7",
	"ETC": "0x815a69C8A7b462B65EB772cfF5cA91440835e225",
}

//测试基于以太坊规则的币种
func TestBtcBaseTransformer(t *testing.T) {
	for name, _ := range testPrivkey {
		testPrivToPub(t, name)
		testPubToAddr(t, name)
	}
}

//测试私钥生成公钥
func testPrivToPub(t *testing.T, name string) {
	coinTrans, err := transformer.New(name)
	if err != nil {
		t.Errorf("new %s transformer error: %s", name, err)
	}
	privByte, err := hex.DecodeString(testPrivkey[name])
	if err != nil {
		t.Errorf("decode hex error: %s", err)
	}

	pubByte, err := coinTrans.PrivKeyToPub(privByte)
	if err != nil {
		t.Errorf("%s PrivKeyToPub error: %s", name, err)
	}
	if hex.EncodeToString(pubByte) == ansPubkey[name] {
		t.Logf("%s public key match", name)
	} else {
		t.Errorf("%s public key mismatch: want: %s have: %x", name, ansPubkey[name], pubByte)
	}
}

//测试公钥生成地址
func testPubToAddr(t *testing.T, name string) {
	coinTrans, err := transformer.New(name)
	if err != nil {
		t.Errorf("new %s transformer error: %s", name, err)
	}
	pubByte, err := hex.DecodeString(ansPubkey[name])
	if err != nil {
		t.Errorf("generate %s public key byte error: %s", name, err)
	}

	genAddr, err := coinTrans.PubKeyToAddress(pubByte)
	if err != nil {
		t.Errorf("%s PubKeyToAddress error: %s", name, err)
	}
	if genAddr == ansAddress[name] {
		t.Logf("%s address match", name)
	} else {
		t.Errorf("%s address mismatch: want: %s have %s", name, ansAddress[name], genAddr)
	}
}

//测试ETH地址生成
// func TestETHTransform(t *testing.T) {
// 	tmp, err := base58.Decode("L3U5kJqHAPXzRGeHvtV5HXTwyjmiAvYG8bk42zswDPz7XwUJ6XfP")
// 	fmt.Printf("%x", tmp)
// 	tmpPriv := hex.EncodeToString(tmp[1:33])
// 	pubkey, genAddr, err := ETHTransformer(tmpPriv)
// 	fmt.Printf("%s", pubkey)
// 	if err != nil {
// 		t.Errorf("ETH transform error: %s", err)
// 	}
// 	if pubkey == testPubHex["ETH"] {
// 		t.Logf("ETH public key transform correctly")
// 	} else {
// 		t.Errorf("ETH public key mismatch：want： %s have： %s", testPubHex["ETH"], pubkey)
// 	}
// 	if genAddr == testAddrHex["ETH"] {
// 		t.Logf("ETH address transform correctly")
// 	} else {
// 		t.Errorf("ETH Address mismatch：want： %s have： %s", testAddrHex["ETH"], genAddr)
// 	}
// }
