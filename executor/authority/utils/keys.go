// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"math/big"

	"github.com/33cn/chain33/types"

	cert_util "github.com/33cn/chain33/system/crypto/common"
	secp256r1_util "github.com/33cn/chain33/system/crypto/secp256r1"
	sm2_util "github.com/33cn/chain33/system/crypto/sm2"
	"github.com/pkg/errors"
	"github.com/tjfoc/gmsm/sm2"
)

// SKI 计算ski
func SKI(curve elliptic.Curve, x, y *big.Int) (ski []byte) {
	raw := elliptic.Marshal(curve, x, y)

	hash := sha256.New()
	hash.Write(raw)
	return hash.Sum(nil)
}

// GetPublicKeySKIFromCert 从cert字节中获取公钥ski
func GetPublicKeySKIFromCert(cert []byte, signType int) (string, error) {
	dcert, _ := pem.Decode(cert)
	if dcert == nil {
		return "", errors.Errorf("Unable to decode cert bytes [%v]", cert)
	}

	var ski []byte
	switch signType {
	case secp256r1_util.ID:
		x509Cert, err := x509.ParseCertificate(dcert.Bytes)
		if err != nil {
			return "", errors.Errorf("Unable to parse cert from decoded bytes: %s", err)
		}
		ecdsaPk := x509Cert.PublicKey.(*ecdsa.PublicKey)
		ski = SKI(ecdsaPk.Curve, ecdsaPk.X, ecdsaPk.Y)
	case sm2_util.ID:
		sm2Cert, err := sm2.ParseCertificate(dcert.Bytes)
		if err != nil {
			return "", errors.Errorf("Unable to parse cert from decoded bytes: %s", err)
		}
		sm2Pk := sm2Cert.PublicKey.(*ecdsa.PublicKey)
		ski = SKI(sm2Pk.Curve, sm2Pk.X, sm2Pk.Y)
	default:
		return "", errors.Errorf("unknow public key type")
	}

	return hex.EncodeToString(ski), nil
}

// EncodeCertToSignature 证书编码进签名
func EncodeCertToSignature(signByte []byte, cert []byte, uid []byte) []byte {
	var certSign cert_util.CertSignature
	certSign.Signature = append(certSign.Signature, signByte...)
	certSign.Cert = append(certSign.Cert, cert...)
	certSign.Uid = append(certSign.Uid, uid...)
	return types.Encode(&certSign)
}

// DecodeCertFromSignature 从签名中解码证书
func DecodeCertFromSignature(signByte []byte) (*cert_util.CertSignature, error) {
	var certSign cert_util.CertSignature
	err := types.Decode(signByte, &certSign)
	if err != nil {
		return nil, err
	}

	return &certSign, nil
}
