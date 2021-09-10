// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package protocol

import (
	"context"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/stretchr/testify/require"
)

func Test_ReadWriteStreamCompatibility(t *testing.T) {

	host1, err := libp2p.New(context.Background())
	require.Nil(t, err)
	host2, err := libp2p.New(context.Background())
	require.Nil(t, err)
	defer host1.Close()
	defer host2.Close()

	id := protocol.ID("teststreamreadwrite")
	_, priv := util.Genaddress()
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	testBlock := util.CreateNoneBlock(cfg, priv, 100)
	handler := func(s core.Stream) {
		block := &types.Block{}
		// 1.4.3的pb结构不能使用老的接口解析, 即对应的1.3.*的pb包
		//err := ReadStreamLegacy(block, s)
		err := ReadStream(block, s)
		require.Nil(t, err)
		require.Equal(t, types.Encode(testBlock), types.Encode(block))
		err = WriteStreamLegacy(block, s)
		require.Nil(t, err)
	}

	host1.SetStreamHandler(id, handler)
	err = host2.Connect(context.Background(), peer.AddrInfo{Addrs: host1.Addrs(), ID: host1.ID()})
	require.Nil(t, err)
	stream, err := host2.NewStream(context.Background(), host1.ID(), id)
	require.Nil(t, err)
	err = WriteStream(testBlock, stream)
	require.Nil(t, err)

	block := &types.Block{}
	err = ReadStream(block, stream)
	require.Nil(t, err)
	require.Equal(t, types.Encode(testBlock), types.Encode(block))

}
