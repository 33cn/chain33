// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright (c) 2014 Casey Marshall. See LICENSE file for details.

package basen_test

import (
	"testing"

	"github.com/33cn/chain33/wallet/bipwallet/basen"
	"github.com/stretchr/testify/assert"
)

/*
func Test(t *testing.T) { gc.TestingT(t) }

type Suite struct{}

var _ = gc.Suite(&Suite{})

func (s *Suite) TestRoundTrip62(c *gc.C) {
	testCases := []struct {
		enc *basen.Encoding
		val []byte
		rep string
	}{
		{basen.Base62, []byte{1}, "1"},
		{basen.Base62, []byte{61}, "z"},
		{basen.Base62, []byte{62}, "10"},
		{basen.Base62, big.NewInt(int64(3844)).Bytes(), "100"},
		{basen.Base62, big.NewInt(int64(3843)).Bytes(), "zz"},

		{basen.Base58, big.NewInt(int64(10002343)).Bytes(), "Tgmc"},
		{basen.Base58, big.NewInt(int64(1000)).Bytes(), "if"},
		{basen.Base58, big.NewInt(int64(0)).Bytes(), ""},
	}

	for _, testCase := range testCases {
		rep := testCase.enc.EncodeToString(testCase.val)
		c.Check(rep, gc.Equals, testCase.rep)

		val, err := testCase.enc.DecodeString(testCase.rep)
		c.Assert(err, gc.IsNil)
		c.Check(val, gc.DeepEquals, testCase.val, gc.Commentf("%s", testCase.rep))
	}
}

func (s *Suite) TestRand256(c *gc.C) {
	for i := 0; i < 100; i++ {
		v := basen.Base62.MustRandom(32)
		// Should be 43 chars or less because math.log(2**256, 62) == 42.994887413002736
		c.Assert(len(v) < 44, gc.Equals, true)
	}
}

func (s *Suite) TestStringN(c *gc.C) {
	var val []byte
	var err error

	val, err = basen.Base58.DecodeStringN("", 4)
	c.Assert(err, gc.IsNil)
	c.Assert(val, gc.DeepEquals, []byte{0, 0, 0, 0})

	// ensure round-trip with padding is right
	val, err = basen.Base62.DecodeStringN("10", 4)
	c.Assert(err, gc.IsNil)
	c.Assert(val, gc.DeepEquals, []byte{0, 0, 0, 62})
	rep := basen.Base62.EncodeToString(val)
	c.Assert(rep, gc.Equals, "10")
}

func (s *Suite) TestNoMultiByte(c *gc.C) {
	c.Assert(func() { basen.NewEncoding("世界") }, gc.PanicMatches,
		"multi-byte characters not supported")
}
*/

func TestBase58(t *testing.T) {
	s := basen.Base58.MustRandom(10)
	assert.False(t, len(s) == 0)

	b, err := basen.Base58.DecodeString(s)
	assert.NoError(t, err)
	assert.True(t, len(b) == 10)

	b, err = basen.Base58.DecodeStringN(s, 12)
	assert.NoError(t, err)
	assert.True(t, len(b) == 12)

	assert.True(t, basen.Base58.Base() == 58)
}
