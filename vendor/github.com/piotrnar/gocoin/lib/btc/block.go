package btc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sync"
)

type Block struct {
	Raw               []byte
	Hash              *Uint256
	Txs               []*Tx
	TxCount, TxOffset int  // Number of transactions and byte offset to the first one
	Trusted           bool // if the block is trusted, we do not check signatures and some other things...
	LastKnownHeight   uint32

	// These flags are set during chain.(Pre/Post)CheckBlock and used later (e.g. by script.VerifyTxScript):
	VerifyFlags    uint32
	Height         uint32
	SigopsCost     uint32
	MedianPastTime uint32

	OldData []byte // all the block's transactions stripped from witnesses
}

func NewBlock(data []byte) (bl *Block, er error) {
	if data == nil {
		er = errors.New("nil pointer")
		return
	}
	bl = new(Block)
	bl.Hash = NewSha2Hash(data[:80])
	er = bl.UpdateContent(data)
	return
}

func (bl *Block) UpdateContent(data []byte) error {
	if len(data) < 81 {
		return errors.New("Block too short")
	}
	bl.Raw = data
	bl.TxCount, bl.TxOffset = VLen(data[80:])
	if bl.TxOffset == 0 {
		return errors.New("Block's txn_count field corrupt - RPC_Result:bad-blk-length")
	}
	bl.TxOffset += 80
	return nil
}

func (bl *Block) Version() uint32 {
	return binary.LittleEndian.Uint32(bl.Raw[0:4])
}

func (bl *Block) ParentHash() []byte {
	return bl.Raw[4:36]
}

func (bl *Block) MerkleRoot() []byte {
	return bl.Raw[36:68]
}

func (bl *Block) BlockTime() uint32 {
	return binary.LittleEndian.Uint32(bl.Raw[68:72])
}

func (bl *Block) Bits() uint32 {
	return binary.LittleEndian.Uint32(bl.Raw[72:76])
}

// Parses block's transactions and adds them to the structure, calculating hashes BTW.
// It would be more elegant to use bytes.Reader here, but this solution is ~20% faster.
func (bl *Block) BuildTxList() (e error) {
	if bl.TxCount == 0 {
		bl.TxCount, bl.TxOffset = VLen(bl.Raw[80:])
		if bl.TxCount == 0 || bl.TxOffset == 0 {
			e = errors.New("Block's txn_count field corrupt - RPC_Result:bad-blk-length")
			return
		}
		bl.TxOffset += 80
	}
	bl.Txs = make([]*Tx, bl.TxCount)

	offs := bl.TxOffset

	var wg sync.WaitGroup
	var data2hash, witness2hash []byte

	old_format_block := new(bytes.Buffer)
	old_format_block.Write(bl.Raw[:80])
	WriteVlen(old_format_block, uint64(bl.TxCount))

	for i := 0; i < bl.TxCount; i++ {
		var n int
		bl.Txs[i], n = NewTx(bl.Raw[offs:])
		if bl.Txs[i] == nil || n == 0 {
			e = errors.New("NewTx failed")
			break
		}
		bl.Txs[i].Raw = bl.Raw[offs : offs+n]
		bl.Txs[i].Size = uint32(n)
		if i == 0 {
			bl.Txs[i].wTxID = new(Uint256) // all zeros
			for _, ou := range bl.Txs[0].TxOut {
				ou.WasCoinbase = true
			}
		}
		if bl.Txs[i].SegWit != nil {
			data2hash = bl.Txs[i].Serialize()
			bl.Txs[i].NoWitSize = uint32(len(data2hash))
			if i > 0 {
				witness2hash = bl.Txs[i].Raw
			}
		} else {
			data2hash = bl.Txs[i].Raw
			bl.Txs[i].NoWitSize = bl.Txs[i].NoWitSize
			witness2hash = nil
		}
		old_format_block.Write(data2hash)
		wg.Add(1)
		go func(tx *Tx, b, w []byte) {
			tx.Hash = NewSha2Hash(b) // Calculate tx hash in a background
			if w != nil {
				tx.wTxID = NewSha2Hash(w)
			}
			wg.Done()
		}(bl.Txs[i], data2hash, witness2hash)
		offs += n
	}

	wg.Wait()

	bl.OldData = old_format_block.Bytes()

	return
}

func GetBlockReward(height uint32) uint64 {
	return 50e8 >> (height / 210000)
}

func (bl *Block) MerkleRootMatch() bool {
	if bl.TxCount == 0 {
		return false
	}
	merkle, _ := GetMerkle(bl.Txs)
	return bytes.Equal(merkle, bl.MerkleRoot())
}
