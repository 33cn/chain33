// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client

import (
	"testing"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {

	module := New()
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	q := queue.New("test")
	q.SetConfig(cfg)
	cli := q.Client()
	module.SetQueueClient(cli)
	defer q.Close()
	defer module.Close()

	for i := 0; i < 10; i++ {
		height := int64(i)
		err := cli.Send(cli.NewMessage("crypto", types.EventAddBlock, &types.Header{Height: height}), true)
		require.Nil(t, err)
	}
	module.Wait()
	sleep := 0
	for GetCryptoContext().CurrBlockHeight != 9 {
		if sleep >= 1000 {
			t.Errorf("expect=9, actual=%d", GetCryptoContext().CurrBlockHeight)
			return
		}
		time.Sleep(10 * time.Millisecond)
		sleep += 10
	}

}
