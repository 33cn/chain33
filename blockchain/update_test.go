// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"errors"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	client "github.com/33cn/chain33/queue"
	clientMocks "github.com/33cn/chain33/queue/mocks"
)

func TestUpgradePlugin(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())

	cli := new(clientMocks.Client)
	cli.On("Sub", "blockchain").Return(nil)
	cli.On("GetConfig").Return(cfg)
	cli.On("Recv").Return(nil)
	msg := &client.Message{Topic: "execs"}
	errSend := errors.New("Send error")
	errWait := errors.New("Wait error")

	cases := []struct {
		m1   *client.Message
		send error
		wait error
		resp *client.Message
	}{
		{msg, nil, nil, nil},
		{msg, nil, nil, &client.Message{Data: &types.LocalDBSet{}}},
		{msg, nil, nil, &client.Message{Data: &types.LocalDBSet{KV: []*types.KeyValue{}}}},
		{msg, errSend, nil, nil},
		{msg, nil, errWait, nil},
	}

	for _, c := range cases {
		func() {
			c1 := c
			defer func() {
				err := recover()
				var expat error
				if c1.send != nil {
					expat = c1.send
				} else if c1.wait != nil {
					expat = c1.wait
				}
				if err == nil {
					assert.Nil(t, expat)
				} else {
					e1 := err.(error)
					assert.Equal(t, expat.Error(), e1.Error())
				}
			}()

			cli.On("NewMessage", "execs", int64(types.EventUpgrade), nil).Return(msg).Once()
			cli.On("Send", msg, true).Return(c1.send).Once()
			cli.On("SendTimeout", mock.Anything, mock.Anything, mock.Anything).Return(c1.send).Once()
			if c1.send == nil {
				cli.On("Wait", msg).Return(c1.resp, c1.wait).Once()
			}
			chain := New(cfg)
			chain.client = cli
			chain.UpgradePlugin()
		}()
	}
}
