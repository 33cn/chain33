package client

// Regression test: ReWriteRawTx must reject a transaction group whose header
// decodes to zero members instead of indexing group.Txs[0].

import (
	"encoding/hex"
	"testing"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/types"
)

func TestReWriteRawTxRejectsEmptyGroup(t *testing.T) {
	emptyGroup := types.Encode(&types.Transactions{})
	tx := &types.Transaction{Execer: []byte("none"), GroupCount: 2, Header: emptyGroup}
	raw := types.Encode(tx)

	api := &mocks.QueueProtocolAPI{}
	api.On("GetConfig").Return(types.NewChain33Config(types.GetDefaultCfgstring()))
	c := &ChannelClient{QueueProtocolAPI: api}
	param := &types.ReWriteRawTx{Tx: hex.EncodeToString(raw), Fee: 1, Index: 0}

	var returned error
	panicked := func() (p interface{}) {
		defer func() { p = recover() }()
		_, returned = c.ReWriteRawTx(param)
		return nil
	}()
	if panicked != nil {
		t.Fatalf("empty tx group must not panic: %v", panicked)
	}
	if returned != types.ErrIndex {
		t.Fatalf("want types.ErrIndex, got: %v", returned)
	}
}
