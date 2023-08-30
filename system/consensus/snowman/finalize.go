// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package snowman
package snowman

import (
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/consensus"
)

var (
	log = log15.New("module", "snowman")
)

func init() {

	consensus.RegFinalizer("snowman", &Transitive{})

}
