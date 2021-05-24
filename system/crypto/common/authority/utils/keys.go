// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utils

import (
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	cert_util "github.com/33cn/chain33/system/crypto/common"
	"github.com/gogo/protobuf/proto"
)

// EncodeCertToSignature 证书编码进签名
func EncodeCertToSignature(signByte []byte, cert []byte, uid []byte) []byte {
	var certSign cert_util.CertSignature
	certSign.Signature = append(certSign.Signature, signByte...)
	certSign.Cert = append(certSign.Cert, cert...)
	certSign.Uid = append(certSign.Uid, uid...)
	b, err := proto.Marshal(&certSign)
	if err != nil {
		panic(err)
	}
	return b
}

// DecodeCertFromSignature 从签名中解码证书
func DecodeCertFromSignature(signByte []byte) (*cert_util.CertSignature, error) {
	var certSign cert_util.CertSignature
	err := proto.Unmarshal(signByte, &certSign)
	if err != nil {
		return nil, err
	}

	return &certSign, nil
}

// MustDecode json解码
func MustDecode(data []byte, v interface{}) {
	if data == nil {
		return
	}
	err := json.Unmarshal(data, v)
	if err != nil {
		panic(err)
	}
}

// ReadFile 读取文件
func ReadFile(file string) ([]byte, error) {
	fileCont, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Could not read file %s, err %s", file, err)
	}

	return fileCont, nil
}

// ReadPemFile 读取pem文件
func ReadPemFile(file string) ([]byte, error) {
	bytes, err := ReadFile(file)
	if err != nil {
		return nil, err
	}

	b, _ := pem.Decode(bytes)
	if b == nil {
		return nil, fmt.Errorf("No pem content for file %s", file)
	}

	return bytes, nil
}
