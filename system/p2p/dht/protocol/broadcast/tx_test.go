// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package broadcast

import (
	"encoding/hex"
	"testing"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func Test_sendTx(t *testing.T) {

	proto := newTestProtocol()
	proto.p2pCfg.LightTxTTL = 1
	proto.p2pCfg.MaxTTL = 3
	route := &types.P2PRoute{}
	tx := &types.P2PTx{Tx: tx, Route: route}
	sPid := testPid.Pretty()
	_, ok := proto.handleSend(tx, sPid)
	assert.True(t, ok)
	_, ok = proto.handleSend(tx, sPid)
	assert.False(t, ok)
	route.TTL = 4
	_, ok = proto.handleSend(tx, sPid)
	assert.False(t, ok)
	route.TTL = 1
	tx.Tx = tx1
	_, ok = proto.handleSend(tx, sPid)
	assert.True(t, ok)
}

func Test_recvTx(t *testing.T) {

	q := queue.New("test")
	go q.Start()
	defer q.Close()

	proto := newTestProtocolWithQueue(q)
	tx := &types.P2PTx{Tx: tx}
	sPid := testPid.Pretty()
	sendData, _ := proto.handleSend(tx, sPid)

	newCli := q.Client()
	newCli.Sub("mempool")
	err := proto.handleReceive(sendData, sPid, testAddr, broadcastV1)
	assert.Nil(t, err)

	msg := <-newCli.Recv()
	assert.Equal(t, types.EventTx, int(msg.Ty))
	tran, ok := msg.Data.(*types.Transaction)
	assert.True(t, ok)
	assert.Equal(t, tx.Tx.Hash(), tran.Hash())

}

func Test_recvLtTx(t *testing.T) {

	proto := newTestProtocol()
	proto.p2pCfg.LightTxTTL = 0
	tx := &types.P2PTx{Tx: tx}
	sendData, _ := proto.handleSend(tx, testPidStr)
	err := proto.handleReceive(sendData, testPidStr, testAddr, broadcastV1)
	assert.Equal(t, errSendPeer, err)

	proto.txFilter.Add(hex.EncodeToString(tx.Tx.Hash()), true)
	err = proto.handleReceive(sendData, testPidStr, testAddr, broadcastV1)
	assert.Equal(t, nil, err)
}
