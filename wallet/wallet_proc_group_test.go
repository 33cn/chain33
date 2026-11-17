package wallet

import (
	"encoding/hex"
	"testing"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/common/crypto"
	qmocks "github.com/33cn/chain33/queue/mocks"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProcSignRawTxRejectsTooFewGroupMembers(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	qm := &qmocks.Client{}
	qm.On("GetConfig").Return(cfg)
	api := &mocks.QueueProtocolAPI{}
	api.On("GetProperFee", mock.Anything).Return(&types.ReplyProperFee{ProperFee: 1}, nil)

	wallet := &Wallet{
		client:     qm,
		api:        api,
		SignType:   types.SECP256K1,
		lastHeader: &types.Header{},
	}
	cr, err := crypto.Load(types.GetSignName("", wallet.SignType), 0)
	require.NoError(t, err)
	key, err := cr.GenKey()
	require.NoError(t, err)
	privkey := hex.EncodeToString(key.Bytes())

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
				txs = append(txs, &types.Transaction{Execer: []byte("coins"), Fee: 1})
			}
			group := &types.Transactions{Txs: txs}
			tx := &types.Transaction{Execer: []byte("coins"), GroupCount: 2, Header: types.Encode(group)}
			unsigned := &types.ReqSignRawTx{
				TxHex:   hex.EncodeToString(types.Encode(tx)),
				Privkey: privkey,
				Index:   tc.index,
				Expire:  "0",
			}

			// A panic here fails the test, which is exactly the regression this
			// test guards against.
			_, err := wallet.ProcSignRawTx(unsigned)
			if err != types.ErrTxGroupCountLessThanTwo {
				t.Fatalf("want types.ErrTxGroupCountLessThanTwo, got: %v", err)
			}
		})
	}
}
