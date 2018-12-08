// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

//ConfigSubModule 子模块的配置
type ConfigSubModule struct {
	Store     map[string][]byte
	Exec      map[string][]byte
	Consensus map[string][]byte
	Wallet    map[string][]byte
	Mempool   map[string][]byte
}
