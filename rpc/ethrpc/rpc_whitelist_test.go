// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ethrpc

import (
	"testing"

	ctypes "github.com/33cn/chain33/types"
)

func newTestHTTPServer(whitelist string) *httpServer {
	cfgstring := `Title="local"
TestNet=true

[mempool]
minTxFeeRate=100000

[wallet]
minFee=100000

[rpc]
whitelist=` + whitelist
	return &httpServer{cfg: ctypes.NewChain33Config(cfgstring)}
}

func TestCheckIPWhitelist(t *testing.T) {
	cases := []struct {
		name      string
		whitelist string
		addr      string
		want      bool
	}{
		{"star allows all", "[\"*\"]", "172.18.0.1", true},
		{"0000 allows all", "[\"0.0.0.0\"]", "172.18.0.1", true},
		{"exact match", "[\"172.18.0.1\"]", "172.18.0.1", true},
		{"other denied", "[\"192.168.1.1\"]", "172.18.0.1", false},
		{"empty allows all", "[]", "172.18.0.1", true},
		{"loopback always allowed", "[]", "127.0.0.1", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newTestHTTPServer(c.whitelist)
			if got := h.checkIPWhitelist(c.addr); got != c.want {
				t.Errorf("checkIPWhitelist(%q) = %v, want %v", c.addr, got, c.want)
			}
		})
	}
}
