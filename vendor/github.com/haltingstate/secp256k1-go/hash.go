package secp256k1

/*
import (
	//"crypto/sha256"
	//"encoding/hex"
	//"errors"
	"hash"

	"github.com/skycoin/skycoin/src/cipher/base58"
	"github.com/skycoin/skycoin/src/cipher/ripemd160"
)

var (
	//sha256Hash    hash.Hash = sha256.New()
	ripemd160Hash hash.Hash = ripemd160.New()
)

// Ripemd160

func HashRipemd160(data []byte) []byte {
	ripemd160Hash.Reset()
	ripemd160Hash.Write(data)
	sum := ripemd160Hash.Sum(nil)
	return sum
}

// SHA256

// Double SHA256
func DoubleSHA256(b []byte) []byte {
	//h := SumSHA256(b)
	//return AddSHA256(h, h)
	h1 := SumSHA256(b)
	h2 := SumSHA256(h1[:])
	return h2
}

//prints the bitcoin address for a seckey
func BitcoinAddressFromPubkey(pubkey []byte) string {
	b1 := SumSHA256(pubkey[:])
	b2 := _HashRipemd160(b1[:])
	b3 := append([]byte{byte(0)}, b2[:]...)
	b4 := _DoubleSHA256(b3)
	b5 := append(b3, b4[0:4]...)
	return string(base58.Hex2Base58(b5))
}

//exports seckey in wallet import format
//key must be compressed
func BitcoinWalletImportFormatFromSeckey(seckey []byte) string {
	b1 := append([]byte{byte(0x80)}, seckey[:]...)
	b2 := append(b1[:], []byte{0x01}...)
	b3 := _DoubleSHA256(b2) //checksum
	b4 := append(b2, b3[0:4]...)
	return string(base58.Hex2Base58(b4))
}
*/
