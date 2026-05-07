package types

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/33cn/chain33/common/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockSize(t *testing.T) {
	b := &Block{}
	assert.Equal(t, b.Size(), Size(b))
	b.Txs = append(b.Txs, &Transaction{Payload: []byte("test")})
	assert.Greater(t, b.Size(), 0)
}

func TestBlockGetHeader(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	b := &Block{
		Version:    1,
		ParentHash: []byte("parent"),
		TxHash:     []byte("txhash"),
		BlockTime:  12345,
		Height:     100,
		Difficulty: 5,
		StateHash:  []byte("state"),
	}
	b.Txs = append(b.Txs, &Transaction{Execer: []byte("coins")})
	header := b.GetHeader(cfg)
	assert.Equal(t, int64(1), header.Version)
	assert.Equal(t, "parent", string(header.ParentHash))
	assert.Equal(t, int64(100), header.Height)
	assert.Equal(t, int64(1), header.TxCount)
	assert.NotNil(t, header.Hash)
}

func TestBlockSetHeader(t *testing.T) {
	b := &Block{}
	header := &Header{
		Version:    2,
		ParentHash: []byte("newparent"),
		TxHash:     []byte("newtxhash"),
		BlockTime:  67890,
		Height:     200,
		Difficulty: 3,
		StateHash:  []byte("newstate"),
		Signature:  &Signature{Ty: 1},
	}
	b.SetHeader(header)
	assert.Equal(t, int64(2), b.Version)
	assert.Equal(t, "newparent", string(b.ParentHash))
	assert.Equal(t, int64(200), b.Height)
	assert.Equal(t, int32(1), b.Signature.Ty)
}

func TestBlock(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	b := &Block{}
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hex.EncodeToString(b.Hash(cfg)))
	assert.Equal(t, b.HashOld(), b.HashNew())
	assert.Equal(t, b.HashOld(), b.Hash(cfg))
	b.Height = 10
	b.Difficulty = 1
	assert.NotEqual(t, b.HashOld(), b.HashNew())
	assert.NotEqual(t, b.HashOld(), b.HashNew())
	assert.Equal(t, b.HashNew(), b.HashByForkHeight(10))
	assert.Equal(t, b.HashOld(), b.HashByForkHeight(11))
	assert.Equal(t, true, b.CheckSign(cfg))

	b.Txs = append(b.Txs, &Transaction{})
	assert.Equal(t, false, b.CheckSign(cfg))
	b.Txs = append(b.Txs, &Transaction{})
	b.Txs = append(b.Txs, &Transaction{})
	b.Txs = append(b.Txs, &Transaction{})
	b.Txs = append(b.Txs, &Transaction{})
	assert.Equal(t, false, b.CheckSign(cfg))
	assert.False(t, VerifySignature(cfg, b, b.Txs[:2]))
	assert.True(t, VerifySignature(cfg, b, b.Txs[:0]))
}

func TestFilterParaTxsByTitle(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	to := "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

	//构造一个主链交易
	maintx := &Transaction{Execer: []byte("coins"), Payload: []byte("none")}
	maintx.To = to
	maintx, err := FormatTx(cfg, "coins", maintx)
	require.NoError(t, err)

	//构造一个平行链交易
	execer := "user.p.hyb.none"
	paratx := &Transaction{Execer: []byte(execer), Payload: []byte("none")}
	paratx.To = address.ExecAddress(execer)
	paratx, err = FormatTx(cfg, execer, paratx)
	require.NoError(t, err)

	//构造一个平行链交易组
	execer1 := "user.p.hyb.coins"
	tx1 := &Transaction{Execer: []byte(execer1), Payload: []byte("none")}
	tx1.To = address.ExecAddress(execer1)
	tx1, err = FormatTx(cfg, execer1, tx1)
	require.NoError(t, err)

	execer2 := "user.p.hyb.token"
	tx2 := &Transaction{Execer: []byte(execer2), Payload: []byte("none")}
	tx2.To = address.ExecAddress(execer2)
	tx2, err = FormatTx(cfg, execer2, tx2)
	require.NoError(t, err)

	execer3 := "user.p.hyb.trade"
	tx3 := &Transaction{Execer: []byte(execer3), Payload: []byte("none")}
	tx3.To = address.ExecAddress(execer3)
	tx3, err = FormatTx(cfg, execer3, tx3)
	require.NoError(t, err)

	var txs Transactions
	txs.Txs = append(txs.Txs, tx1)
	txs.Txs = append(txs.Txs, tx2)
	txs.Txs = append(txs.Txs, tx3)
	feeRate := cfg.GetMinTxFeeRate()
	group, err := CreateTxGroup(txs.Txs, feeRate)
	require.NoError(t, err)

	//构造一个有平行链交易的区块
	block := &Block{}
	block.Version = 0
	block.Height = 0
	block.BlockTime = 1
	block.Difficulty = 1
	block.Txs = append(block.Txs, maintx)
	block.Txs = append(block.Txs, paratx)
	block.Txs = append(block.Txs, group.Txs...)

	blockdetal := &BlockDetail{}
	blockdetal.Block = block

	maintxreceipt := &ReceiptData{Ty: ExecOk}
	paratxreceipt := &ReceiptData{Ty: ExecPack}
	grouppara1receipt := &ReceiptData{Ty: ExecPack}
	grouppara2receipt := &ReceiptData{Ty: ExecPack}
	grouppara3receipt := &ReceiptData{Ty: ExecPack}

	blockdetal.Receipts = append(blockdetal.Receipts, maintxreceipt)
	blockdetal.Receipts = append(blockdetal.Receipts, paratxreceipt)
	blockdetal.Receipts = append(blockdetal.Receipts, grouppara1receipt)
	blockdetal.Receipts = append(blockdetal.Receipts, grouppara2receipt)
	blockdetal.Receipts = append(blockdetal.Receipts, grouppara3receipt)

	txDetail := blockdetal.FilterParaTxsByTitle(cfg, "user.p.hyb.")
	for _, tx := range txDetail.TxDetails {
		if tx != nil {
			execer := string(tx.Tx.Execer)
			if !strings.HasPrefix(execer, "user.p.hyb.") && tx.Tx.GetGroupCount() != 0 {
				assert.Equal(t, tx.Receipt.Ty, int32(ExecOk))
			} else {
				assert.Equal(t, tx.Receipt.Ty, int32(ExecPack))
			}
		}
	}
}
