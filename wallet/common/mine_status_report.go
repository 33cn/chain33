// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

// MineStatusReport 挖矿操作状态
type MineStatusReport interface {
	IsAutoMining() bool
	// 返回挖矿买票锁的状态
	IsTicketLocked() bool
}
