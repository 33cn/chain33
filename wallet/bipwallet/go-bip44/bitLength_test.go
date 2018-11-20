// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bip44_test

import (
	"testing"

	"github.com/33cn/chain33/wallet/bipwallet"
	. "github.com/33cn/chain33/wallet/bipwallet/go-bip44"
)

func TestBitLength(t *testing.T) {
	child, err := NewKeyFromMnemonic(
		"element fence situate special wrap snack method volcano busy ribbon neck sphere",
		bipwallet.TypeFactomFactoids,
		2147483648,
		0,
		19,
	)

	if err != nil {
		t.Errorf("%v", err)
	}
	if len(child.Key) != 32 {
		t.Errorf("len: %d, child.Key:%x\n", len(child.Key), child.Key)
		t.Errorf("%v", child.String())
	}

	child, err = NewKeyFromMnemonic(
		"element fence situate special wrap snack method volcano busy ribbon neck sphere",
		bipwallet.TypeFactomFactoids,
		2147483648,
		1,
		19,
	)

	if err != nil {
		t.Errorf("%v", err)
	}
	if len(child.Key) != 32 {
		t.Errorf("len: %d, child.Key:%x\n", len(child.Key), child.Key)
		t.Errorf("%v", child.String())
	}
}
