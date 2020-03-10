// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package p2p

import (
	"net"
	"testing"

	"github.com/33cn/chain33/p2p/utils"

	"github.com/stretchr/testify/assert"
)

func TestNetAddress(t *testing.T) {
	tcpAddr := new(net.TCPAddr)
	tcpAddr.IP = net.ParseIP("localhost")
	tcpAddr.Port = 2223
	nad := NewNetAddress(tcpAddr)
	nad1 := nad.Copy()
	nad.Equals(nad1)
	nad2s, err := NewNetAddressStrings([]string{"localhost:3306"})
	if err != nil {
		return
	}
	nad.Less(nad2s[0])

}

func TestAddrRouteble(t *testing.T) {
	resp := P2pComm.AddrRouteble([]string{"114.55.101.159:13802"}, utils.CalcChannelVersion(119, VERSION))
	t.Log(resp)
}

func TestGetLocalAddr(t *testing.T) {
	t.Log(P2pComm.GetLocalAddr())
}

func TestP2pListen(t *testing.T) {
	var node Node
	node.listenPort = 3333
	listen1 := newListener("tcp", &node)
	assert.Equal(t, true, listen1 != nil)
	listen2 := newListener("tcp", &node)
	assert.Equal(t, true, listen2 != nil)

	listen1.Close()
	listen2.Close()
}
