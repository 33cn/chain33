package types

import (
	"encoding/json"
	"testing"

	dtypes "github.com/33cn/chain33/system/dapp/coins/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPlainCoinsTransferInputIsEmpty(t *testing.T) {
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	to := "0x1ace55968b5d1f0857d2a21e76bb4039140c6905"
	action := &dtypes.CoinsAction{
		Value: &dtypes.CoinsAction_Transfer{Transfer: &ctypes.AssetsTransfer{Amount: 100, To: to}},
		Ty:    dtypes.CoinsActionTransfer,
	}
	tx := &ctypes.Transaction{
		Execer:  []byte("coins"),
		To:      to,
		Fee:     100000,
		Nonce:   1,
		Payload: ctypes.Encode(action),
		Signature: &ctypes.Signature{
			Ty:        1,
			Pubkey:    common.FromHex("0x02f5263862dae4e8516e08e5551f353be05f479c4d36ea754a7b2b359f81fbebb1"),
			Signature: common.FromHex("0x30450221008259d987850c34036a33ede7db25ea72dbe92edd3e9bfe6cbcabd0c43c427a9902206f410d921d6b0d09ce9a8cdf0db9f8eec64677c56404fcd2ccf0e2c5ef4dd06a"),
		},
	}

	parsed := paraseChain33Tx(tx, common.Hash{}, 1, 0, cfg)
	require.NotNil(t, parsed)
	require.Empty(t, []byte(parsed.Input))

	encoded, err := json.Marshal(parsed.Input)
	require.NoError(t, err)
	require.Equal(t, `"0x"`, string(encoded))

	details := &ctypes.TransactionDetails{Txs: []*ctypes.TransactionDetail{{
		Tx:      tx,
		Height:  1,
		Receipt: &ctypes.ReceiptData{Ty: ctypes.ExecOk},
	}}}
	txs, receipts, err := TxDetailsToEthReceipts(details, common.Hash{}, cfg)
	require.NoError(t, err)
	require.Len(t, txs, 1)
	require.Empty(t, []byte(txs[0].Input))
	require.Len(t, receipts, 1)
	require.Nil(t, receipts[0].ContractAddress)
}
