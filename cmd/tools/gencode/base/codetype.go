// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

// DappCodeFile dapp code source
type DappCodeFile struct {
	CodeFile
}

// GetCodeType get code type
func (DappCodeFile) GetCodeType() string {
	return "dapp"
}
