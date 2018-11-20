// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import "fmt"

const (
	keyAccount            = "Account"
	keyAddr               = "Addr"
	keyLabel              = "Label"
	keyTx                 = "Tx"
	keyEncryptionFlag     = "Encryption"
	keyEncryptionCompFlag = "EncryptionFlag" // 中间有一段时间运行了一个错误的密码版本，导致有部分用户信息发生错误，需要兼容下
	keyPasswordHash       = "PasswordHash"
	keyWalletSeed         = "walletseed"
)

// CalcAccountKey 用于所有Account账户的输出list，需要安装时间排序
func CalcAccountKey(timestamp string, addr string) []byte {
	return []byte(fmt.Sprintf("%s:%s:%s", keyAccount, timestamp, addr))
}

// CalcAddrKey 通过addr地址查询Account账户信息
func CalcAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("%s:%s", keyAddr, addr))
}

// CalcLabelKey 通过label查询Account账户信息
func CalcLabelKey(label string) []byte {
	return []byte(fmt.Sprintf("%s:%s", keyLabel, label))
}

// CalcTxKey 通过height*100000+index 查询Tx交易信息
//key:Tx:height*100000+index
func CalcTxKey(key string) []byte {
	return []byte(fmt.Sprintf("%s:%s", keyTx, key))
}

// CalcEncryptionFlag 加密标志Key
func CalcEncryptionFlag() []byte {
	return []byte(keyEncryptionFlag)
}

// CalckeyEncryptionCompFlag 加密比较标志Key
func CalckeyEncryptionCompFlag() []byte {
	return []byte(keyEncryptionCompFlag)
}

// CalcPasswordHash 密码hash的Key
func CalcPasswordHash() []byte {
	return []byte(keyPasswordHash)
}

// CalcWalletSeed 钱包Seed的Key
func CalcWalletSeed() []byte {
	return []byte(keyWalletSeed)
}
