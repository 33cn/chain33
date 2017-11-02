// NCNL is No Copyright and No License
//
//

package gosm2

import (
	"encoding/base64"
	"encoding/hex"
	"log"
	"testing"
)

func newKey(t *testing.T) *PriveKey {
	priv, err := NewKey()
	if err != nil {
		t.Error(err)
	}
	return priv
}

func TestNewKeyFormSecret(t *testing.T) {
	priv, err := NewKeyFormSecret("abcdefghijklmn")
	if err != nil {
		t.Error(err)
	}
	log.Println("private:", priv.EncodeString())
	log.Println("public:", priv.PublicKey.EncodeString())
}

func TestPublicKeyEncode(t *testing.T) {
	priv := newKey(t)
	pub := priv.PublicKey
	log.Println(pub.HashString(), 3)

	s := pub.EncodeString()
	log.Println(s)

	err := pub.DecodeString(s)
	if err != nil {
		t.Error(err)
	}
	s2 := pub.EncodeString()
	log.Println(s2)
	if s != s2 {
		t.Error("s != pub.EncodeString()")
	}
}

func TestPrivateKeyEncode(t *testing.T) {
	priv := newKey(t)

	s := priv.EncodeString()
	log.Println(s)

	err := priv.DecodeString(s)
	if err != nil {
		t.Error(err)
	}
	s2 := priv.EncodeString()
	log.Println(s2)
	if s != s2 {
		t.Error("s != priv.EncodeString()")
	}
}

func TestEncrypt(t *testing.T) {
	priv := newKey(t)
	msg := []byte("I have an apple")
	cipertext, err := Encrypt(priv.PublicKey, msg)
	if err != nil {
		t.Error(err)
	}

	out, err := Decrypt(priv, cipertext)
	if err != nil {
		t.Error(err)
	}

	if string(out) != string(msg) {
		t.Errorf("out != msg, out:%s, msg:%s", string(out), string(msg))
	}

	// log.Println("...")
}

func TestSign2(t *testing.T) {
	pubStr := "0477dac51899bbdf4fdf3b7b9ba7f931f1e03cc2718894ae20cb823ecc9e5b5c2a170f4345b75c591b13c810d812bd7bd00d42f275cac6fb2ce9546eda3f2d25b3"
	var pubkey PublicKey
	err := pubkey.DecodeString(pubStr)
	if err != nil {
		t.Error(err)
	}

	log.Println("public:", pubkey.EncodeString())

	msg := []byte(`<?xml version="1.0" encoding="utf-8"?><T><D><M><k>Remitter Name:</k><v>Alan</v></M><M><k>Remitter Account:</k><v>1234567890123456</v></M><M><k>Amount:</k><v>123.23</v></M></D><E><M><k>Trade ID:</k><v>1234567890</v></M></E></T>`)
	// msg := []byte(`<?xml version=\"1.0\" encoding=\"utf-8\"?><T><D><M><k>Remitter Name:</k><v>Alan</v></M><M><k>Remitter Account:</k><v>1234567890123456</v></M><M><k>Amount:</k><v>123.23</v></M></D><E><M><k>Trade ID:</k><v>1234567890</v></M></E></T>`)
	hash, err := Hash(&pubkey, msg)
	if err != nil {
		t.Error(err)
	}
	log.Println("hash:", hex.EncodeToString(hash))

	sigStr := "Pt3FsCjEGW/Vu57PrQsMkzQhEmoYPDmvgdvg45G6qJhce8iWzkroGkS+yUwUk6cZ1QS85gdjgMmzb6Djqj8Q2w=="
	sig, err := base64.StdEncoding.DecodeString(sigStr)
	if err != nil {
		t.Error(err)
	}
	log.Println("sig1:", len(sig), hex.EncodeToString(sig))
	sig2, err := Sigbin2Der(sig)
	if err != nil {
		t.Error(err)
	}
	log.Println("sig2:", len(sig2), hex.EncodeToString(sig2))
	if err != nil {
		t.Error(err)
	}
	err = Verify(&pubkey, hash, sig2)
	if err != nil {
		t.Error(err)
	}
}

func TestSign(t *testing.T) {
	priv := newKey(t)
	msg := []byte(`<?xml version=\"1.0\" encoding=\"utf-8\"?><T><D><M><k>Remitter Name:</k><v>Alan</v></M><M><k>Remitter Account:</k><v>1234567890123456</v></M><M><k>Amount:</k><v>123.23</v></M></D><E><M><k>Trade ID:</k><v>1234567890</v></M></E></T>`)

	hash, err := Hash2(msg)
	if err != nil {
		t.Error(err)
	}

	log.Println("hash:", hex.EncodeToString(hash))

	sig, err := Sign(priv, hash)
	if err != nil {
		t.Error(err)
	}

	log.Println("sig:", len(sig), hex.EncodeToString(sig))

	err = Verify(priv.PublicKey, hash, sig)
	if err != nil {
		t.Error(err)
	}
}

func BenchmarkNewKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewKey()
	}
}

func BenchmarkEncrypt(b *testing.B) {
	priv, _ := NewKey()
	msg := []byte("I have an apple")
	for i := 0; i < b.N; i++ {
		Encrypt(priv.PublicKey, msg)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	priv, _ := NewKey()
	msg := []byte("I have an apple")
	cipertext, err := Encrypt(priv.PublicKey, msg)
	if err != nil {
		b.Error(err)
	}

	for i := 0; i < b.N; i++ {
		Decrypt(priv, cipertext)
	}
}

// func BenchmarkHash(b *testing.B) {
// 	priv, _, _ := NewKey("hahahaha")
// 	msg := []byte("I have an apple")
// 	for i := 0; i < b.N; i++ {
// 		Hash(priv, msg)
// 	}
// }

func BenchmarkSign(b *testing.B) {
	priv, _ := NewKey()
	msg := []byte("I have an apple")

	hash, err := Hash(priv.PublicKey, msg)
	if err != nil {
		b.Error(err)
	}

	for i := 0; i < b.N; i++ {
		Sign(priv, hash)
	}
}

func BenchmarkVerify(b *testing.B) {
	priv, _ := NewKey()
	msg := []byte("I have an apple")

	hash, err := Hash(priv.PublicKey, msg)
	if err != nil {
		b.Error(err)
	}

	sig, err := Sign(priv, hash)
	if err != nil {
		b.Error(err)
	}

	for i := 0; i < b.N; i++ {
		Verify(priv.PublicKey, hash[:], sig)
	}
}
