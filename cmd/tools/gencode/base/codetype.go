// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.


package base




type DappCodeFile struct {
	BaseCodeFile
}


func (DappCodeFile)GetCodeType() string {
	return "dapp"
}
