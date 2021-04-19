// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package none

import (
	"testing"

	"github.com/33cn/chain33/common/crypto"
	"github.com/stretchr/testify/require"
)

func Test_None(t *testing.T) {

	c := &Driver{}
	pub, err := c.PubKeyFromBytes([]byte("test"))
	require.Nil(t, pub)
	require.Nil(t, err)
	sig, err := c.SignatureFromBytes([]byte("test"))
	require.Nil(t, sig)
	require.Nil(t, err)
	require.Nil(t, c.Validate([]byte("test"), nil, nil))
	_, err = crypto.New(Name)
	require.Nil(t, err)
	_, err = crypto.New(Name, crypto.WithNewOptionEnableCheck(0))
	require.NotNil(t, err)
}
