package client

// Regression tests: ReWriteRawTx must reject transaction groups whose Header
// decodes to fewer than two members instead of indexing group.Txs[0] or
// encoding a nil group.Tx().

import (
	"encoding/hex"
	"testing"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/types"
)

func TestReWriteRawTxRejectsTooFewGroupMembers(t *testing.T) {
	cases := []struct {
		name    string
		members int
		index   int32
	}{
		{"empty group index 0", 0, 0},
		{"empty group index 1", 0, 1},
		{"single member index 0", 1, 0},
		{"single member index 1", 1, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			txs := make([]*types.Transaction, 0, tc.members)
			for i := 0; i < tc.members; i++ {
				txs = append(txs, &types.Transaction{Execer: []byte("none"), Fee: 1})
			}
			group := &types.Transactions{Txs: txs}
			tx := &types.Transaction{Execer: []byte("none"), GroupCount: 2, Header: types.Encode(group)}
			raw := types.Encode(tx)

			api := &mocks.QueueProtocolAPI{}
			api.On("GetConfig").Return(types.NewChain33Config(types.GetDefaultCfgstring()))
			c := &ChannelClient{QueueProtocolAPI: api}
			param := &types.ReWriteRawTx{Tx: hex.EncodeToString(raw), Fee: 1, Index: tc.index}

			// A panic here fails the test, which is exactly the regression this
			// test guards against.
			_, err := c.ReWriteRawTx(param)
			if err != types.ErrTxGroupCountLessThanTwo {
				t.Fatalf("want types.ErrTxGroupCountLessThanTwo, got: %v", err)
			}
		})
	}
}
