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
	priv, err := c.GenKey()
	require.Nil(t, priv)
	require.Nil(t, err)
	priv2, err := c.PrivKeyFromBytes([]byte("test"))
	require.Nil(t, priv2)
	require.Nil(t, err)
	pub, err := c.PubKeyFromBytes([]byte("test"))
	require.Nil(t, pub)
	require.Nil(t, err)
	sig, err := c.SignatureFromBytes([]byte("test"))
	require.Nil(t, sig)
	require.Nil(t, err)
	require.Nil(t, c.Validate([]byte("test"), nil, nil))
	_, err = crypto.Load(Name, -1)
	require.Nil(t, err)
	_, err = crypto.Load(Name, 0)
	require.Equal(t, crypto.ErrDriverNotEnable, err)
}
