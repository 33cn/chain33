// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

const (
	keyWalletPassKey = "WalletPassKey"
)

// CalcWalletPassKey 获取钱包密码的数据库字段Key值
func CalcWalletPassKey() []byte {
	return []byte(keyWalletPassKey)
}
