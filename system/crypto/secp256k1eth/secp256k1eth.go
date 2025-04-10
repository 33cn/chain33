// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secp256k1eth

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/crypto/common/authority/utils"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/crypto"
	ecommon "github.com/ethereum/go-ethereum/common"

	//"github.com/33cn/chain33/types"
	"math/big"

	"github.com/33cn/chain33/system/crypto/secp256k1eth/types"
	secp256k1 "github.com/btcsuite/btcd/btcec/v2"
	etypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const privKeyBytesLen = 32
const pubkeyBytesLen = 64 + 1

var log = log15.New("module", Name)
var chainID int64
var coinsPrecision int64

// PrivKeySecp256k1Eth PrivKey
type PrivKeySecp256k1Eth [32]byte

// Driver 驱动
type Driver struct{}

// SignatureFromBytes  对字节数组签名
func (d Driver) SignatureFromBytes(b []byte) (crypto.Signature, error) {
	return SignatureSecp256k1Eth(b), nil
}

// PrivKeyFromBytes 字节转为私钥
func (d Driver) PrivKeyFromBytes(b []byte) (crypto.PrivKey, error) {
	if len(b) != privKeyBytesLen {
		return nil, errors.New("invalid priv key byte")
	}

	privKeyBytes := new([privKeyBytesLen]byte)
	copy(privKeyBytes[:], b[:privKeyBytesLen])
	return PrivKeySecp256k1Eth(*privKeyBytes), nil
}

// PubKeyFromBytes must 65 bytes uncompress key
func (d Driver) PubKeyFromBytes(b []byte) (crypto.PubKey, error) {
	if len(b) != pubkeyBytesLen && len(b) != 33 {
		return nil, errors.New("invalid pub key byte,must be 65 bytes")
	}
	if len(b) == 33 {
		p, err := ethcrypto.DecompressPubkey(b)
		if err != nil {
			return nil, err
		}
		b = ethcrypto.FromECDSAPub(p)
	}
	var pubKeyBytes [pubkeyBytesLen]byte
	//pubKeyBytes := new([pubkeyBytesLen]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeySecp256k1Eth(pubKeyBytes), nil
}

// Validate check signature
func (d Driver) Validate(msg, pub, sig []byte) error {
	return crypto.BasicValidation(d, msg, pub, sig)
}

// GenKey 生成私钥
func (d Driver) GenKey() (crypto.PrivKey, error) {
	privKeyBytes := [32]byte{}
	copy(privKeyBytes[:], crypto.CRandBytes(32))
	//fmt.Println(fmt.Sprintf("GenKey:%x", privKeyBytes))
	priv, _ := secp256k1.PrivKeyFromBytes(privKeyBytes[:])
	copy(privKeyBytes[:], priv.Serialize())
	return PrivKeySecp256k1Eth(privKeyBytes), nil
}

// Bytes 字节格式
func (privKey PrivKeySecp256k1Eth) Bytes() []byte {
	s := make([]byte, 32)
	copy(s, privKey[:])
	return s
}

// Sign 签名 The produced signature is in the [R || S || V] format where V is 0 or 1.
func (privKey PrivKeySecp256k1Eth) Sign(msg []byte) crypto.Signature {

	priv, err := ethcrypto.ToECDSA(privKey[:])
	if err != nil {
		return nil
	}
	hash := ethcrypto.Keccak256(msg)
	sig, err := ethcrypto.Sign(hash, priv)
	if err != nil {
		panic("Error Sign calculates an ECDSA signature." + err.Error())
	}
	return SignatureSecp256k1Eth(sig)
}

// PubKey 私钥生成公钥 非压缩 65 bytes 0x04+pub.X+pub.Y
func (privKey PrivKeySecp256k1Eth) PubKey() crypto.PubKey {
	priv, err := ethcrypto.ToECDSA(privKey[:])
	if nil != err {
		return nil
	}
	//uncompressed pubkey
	var pubSecp256k1 PubKeySecp256k1Eth
	pub := ethcrypto.FromECDSAPub(&priv.PublicKey)
	copy(pubSecp256k1[:], pub)
	return pubSecp256k1
}

// Equals 私钥是否相等
func (privKey PrivKeySecp256k1Eth) Equals(other crypto.PrivKey) bool {
	if otherSecp, ok := other.(PrivKeySecp256k1Eth); ok {
		return bytes.Equal(privKey[:], otherSecp[:])
	}
	return false

}

func (privKey PrivKeySecp256k1Eth) String() string {
	return "PrivKeySecp256k1{*****}"
}

// SignatureSecp256k1Eth Signature
type SignatureSecp256k1Eth []byte

// SignatureS 签名
type SignatureS struct {
	crypto.Signature
}

// Bytes 字节格式
func (sig SignatureSecp256k1Eth) Bytes() []byte {
	s := make([]byte, len(sig))
	copy(s, sig[:])
	return s
}

// IsZero 是否是0
func (sig SignatureSecp256k1Eth) IsZero() bool { return len(sig) == 0 }

func (sig SignatureSecp256k1Eth) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)

}

// Equals 相等
func (sig SignatureSecp256k1Eth) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(SignatureSecp256k1Eth); ok {
		return bytes.Equal(sig[:], otherEd[:])
	}
	return false

}

// PubKey

// PubKeySecp256k1Eth uncompressed pubkey (just the x-cord),
// prefixed with 0x04
type PubKeySecp256k1Eth [65]byte

// Bytes 字节格式
func (pubKey PubKeySecp256k1Eth) Bytes() []byte {
	s := make([]byte, 65)
	copy(s, pubKey[:])
	return s
}

// VerifyBytes 验证字节
func (pubKey PubKeySecp256k1Eth) VerifyBytes(msg []byte, sig crypto.Signature) bool {
	var hash []byte
	action, err := types.DecodeTxAction(msg)
	if err == nil && len(action.Note) == 0 || err != nil {
		//chain33格式交易，sha3哈希 eth 签名
		hash = common.Sha3(msg)
	} else { //解析ETH交易数据
		var etx = new(etypes.Transaction)
		err := etx.UnmarshalBinary(action.Note)
		if err != nil {
			log.Error("VerifyBytes", "PubKeySecp256k1Eth,UnmarshalBinary", err)
			return false
		}
		//chainID 可配置
		if etx.ChainId().Int64() != chainID {
			log.Error("VerifyBytes", "local ChainID", chainID, "etx.ChainID", etx.ChainId())
			return false
		}
		signer := etypes.NewLondonSigner(etx.ChainId())
		hash = signer.Hash(etx).Bytes() //metamask,eth 兼容交易，取出eth 交易格式下的哈希
		//check nonce 防止重放攻击
		if action.Nonce != int64(etx.Nonce()) {
			log.Error("VerifyBytes", "action.Nonce", action.Nonce, "tx.Nonce", etx.Nonce())
			return false
		}
		//checkout amount
		amount := etx.Value().Div(etx.Value(), new(big.Int).Div(big.NewInt(1e18), big.NewInt(coinsPrecision)))
		if amount.Uint64() != action.Amount { //防止自组装的chain33 Tx amount 被篡改
			log.Error("VerifyBytes", "amount:", amount, "action.Amount", action.Amount)
			return false
		}
		//check evm code
		if !bytes.Equal(etx.Data(), action.Code) { // 防止合约代码被篡改
			log.Error("VerifyBytes", "check action.Code not equal")
			return false
		}
		//check to address
		if etx.To() != nil && !bytes.Equal(ecommon.FromHex(action.To), etx.To().Bytes()) {
			log.Error("VerifyBytes", "ex.To", etx.To(), "action.To", action.To)
			return false
		}
	}

	sigBytes := sig.Bytes()
	recoverPub, err := ethcrypto.Ecrecover(hash, sigBytes)
	if err != nil {
		log.Error("VerifyBytes", "Ecrecover", err)
		return false
	}
	if !bytes.Equal(recoverPub, pubKey[:]) {
		log.Error("VerifyBytes", "pubkey not equal")
		return false
	}

	return ethcrypto.VerifySignature(pubKey[:], hash, sigBytes[:64])
}

func (pubKey PubKeySecp256k1Eth) String() string {
	return fmt.Sprintf("PubKeySecp256k1{%X}", pubKey[:])
}

// KeyString Must return the full bytes in hex.
// Used for map keying, etc.
func (pubKey PubKeySecp256k1Eth) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

// Equals 公钥相等
func (pubKey PubKeySecp256k1Eth) Equals(other crypto.PubKey) bool {
	if otherSecp, ok := other.(PubKeySecp256k1Eth); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	}
	return false

}

func initEvmIDFun(sub []byte) {
	var id struct {
		EvmChainID     int64 `json:"evmChainID,omitempty"`
		CoinsPrecision int64 `json:"coinsPrecision,omitempty"`
	}
	utils.MustDecode(sub, &id)
	chainID = id.EvmChainID
	coinsPrecision = id.CoinsPrecision
	if coinsPrecision == 0 { //默认配置是 1e8
		coinsPrecision = 1e8
	}
}

// GetEvmChainID return evm chainID
func GetEvmChainID() int64 {
	return chainID
}

// CaculCoinsEvmAccountKey 兼容eth 账户的nonce 的key
func CaculCoinsEvmAccountKey(addr string) []byte {
	//prefix := "mval-" + "evm" + "-noncestate:" + strings.ToLower(addr)
	prefix := "LODB-" + "evm" + "-noncestate:" + strings.ToLower(addr)
	return []byte(prefix)

}

// const
const (
	Name = "secp256k1eth"
	ID   = 260
)

func init() {
	crypto.Register(Name, &Driver{}, crypto.WithRegOptionTypeID(ID), crypto.WithRegOptionInitFunc(initEvmIDFun))
}
