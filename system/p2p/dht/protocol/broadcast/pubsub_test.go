// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"context"
	"testing"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/require"
)

func testRecvMsg(p *pubSub, topic string, data []byte, buf *[]byte) (types.Message, error) {
	msg := p.newMsg(topic)
	err := p.decodeMsg(data, buf, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func newTestPubSub() *pubSub {
	p := &pubSub{&Protocol{}}
	p.P2PEnv = &protocol.P2PEnv{}
	p.ChainCfg = testnode.GetDefaultConfig()

	return p
}

func TestPubSub(t *testing.T) {

	ps := newTestPubSub()
	ctx, cancel := context.WithCancel(context.Background())
	ps.Ctx = ctx
	addr, priv := util.Genaddress()
	tx := util.CreateCoinsTx(ps.ChainCfg, priv, addr, 1)
	block := util.CreateCoinsBlock(ps.ChainCfg, priv, 10)
	txHash := ps.getMsgHash(psTxTopic, tx)
	blockHash := ps.getMsgHash(psBlockTopic, block)

	sendBuf := make([]byte, 0)

	txData := ps.encodeMsg(tx, &sendBuf)
	require.Equal(t, len(txData), len(sendBuf))

	blockData := ps.encodeMsg(block, &sendBuf)

	require.Equal(t, len(blockData), len(sendBuf))

	recvBuf := make([]byte, 0)
	msg, err := testRecvMsg(ps, psTxTopic, txData, &recvBuf)
	require.Nil(t, err)
	require.Equal(t, len(types.Encode(tx)), len(recvBuf))
	require.Equal(t, txHash, ps.getMsgHash(psTxTopic, msg))
	msg, err = testRecvMsg(ps, psBlockTopic, blockData, &recvBuf)
	require.Nil(t, err)
	require.Equal(t, blockHash, ps.getMsgHash(psBlockTopic, msg))
	cancel()
	time.Sleep(time.Second)
}
