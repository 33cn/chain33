package chain

import (
	"errors"
	"fmt"
	"github.com/piotrnar/gocoin/lib/btc"
	"github.com/piotrnar/gocoin/lib/script"
	"github.com/piotrnar/gocoin/lib/utxo"
	"sync"
	"sync/atomic"
)

// TrustedTxChecker is meant to speed up verifying transactions that had
// been verified already by the client while being taken to its memory pool
var TrustedTxChecker func(*btc.Uint256) bool

func (ch *Chain) ProcessBlockTransactions(bl *btc.Block, height, lknown uint32) (changes *utxo.BlockChanges, e error) {
	changes = new(utxo.BlockChanges)
	changes.Height = height
	changes.LastKnownHeight = lknown
	changes.DeledTxs = make(map[[32]byte][]bool)
	e = ch.commitTxs(bl, changes)
	return
}

// This function either appends a new block at the end of the existing chain
// in which case it also applies all the transactions to the unspent database.
// If the block does is not the heighest, it is added to the chain, but maked
// as an orphan - its transaction will be verified only if the chain would swap
// to its branch later on.
func (ch *Chain) AcceptBlock(bl *btc.Block) (e error) {
	ch.BlockIndexAccess.Lock()
	cur := ch.AcceptHeader(bl)
	ch.BlockIndexAccess.Unlock()
	return ch.CommitBlock(bl, cur)
}

// Make sure to call this function with ch.BlockIndexAccess locked
func (ch *Chain) AcceptHeader(bl *btc.Block) (cur *BlockTreeNode) {
	prevblk, ok := ch.BlockIndex[btc.NewUint256(bl.ParentHash()).BIdx()]
	if !ok {
		panic("This should not happen")
	}

	// create new BlockTreeNode
	cur = new(BlockTreeNode)
	cur.BlockHash = bl.Hash
	cur.Parent = prevblk
	cur.Height = prevblk.Height + 1
	copy(cur.BlockHeader[:], bl.Raw[:80])

	// Add this block to the block index
	prevblk.addChild(cur)
	ch.BlockIndex[cur.BlockHash.BIdx()] = cur

	return
}

func (ch *Chain) CommitBlock(bl *btc.Block, cur *BlockTreeNode) (e error) {
	cur.BlockSize = uint32(len(bl.Raw))
	cur.TxCount = uint32(bl.TxCount)
	if ch.BlockTreeEnd == cur.Parent {
		// The head of out chain - apply the transactions
		var changes *utxo.BlockChanges
		changes, e = ch.ProcessBlockTransactions(bl, cur.Height, bl.LastKnownHeight)
		if e != nil {
			// ProcessBlockTransactions failed, so trash the block.
			//println("ProcessBlockTransactionsA", cur.BlockHash.String(), cur.Height, e.Error())
			ch.BlockIndexAccess.Lock()
			cur.Parent.delChild(cur)
			delete(ch.BlockIndex, cur.BlockHash.BIdx())
			ch.BlockIndexAccess.Unlock()
		} else {
			cur.SigopsCost = bl.SigopsCost
			// ProcessBlockTransactions succeeded, so save the block as "trusted".
			bl.Trusted = true
			ch.Blocks.BlockAdd(cur.Height, bl)
			// Apply the block's trabnsactions to the unspent database:
			ch.Unspent.CommitBlockTxs(changes, bl.Hash.Hash[:])
			ch.BlockTreeEnd = cur // Advance the head
			if ch.CB.BlockMinedCB != nil {
				ch.CB.BlockMinedCB(bl)
			}
		}
	} else {
		// The block's parent is not the current head of the chain...

		// Save the block, though do not mark it as "trusted" just yet
		ch.Blocks.BlockAdd(cur.Height, bl)

		// If it has a bigger height than the current head,
		// ... move the coin state into a new branch.
		if cur.Height > ch.BlockTreeEnd.Height {
			ch.MoveToBlock(cur)
			if ch.BlockTreeEnd != cur {
				e = errors.New("CommitBlock: MoveToBlock failed")
			}
		} else {
			println("Orphaned block", bl.Hash.String(), cur.Height)
		}
	}

	return
}

// This isusually the most time consuming process when applying a new block
func (ch *Chain) commitTxs(bl *btc.Block, changes *utxo.BlockChanges) (e error) {
	sumblockin := btc.GetBlockReward(changes.Height)
	var txoutsum, txinsum, sumblockout uint64

	if changes.Height+ch.Unspent.UnwindBufLen >= changes.LastKnownHeight {
		changes.UndoData = make(map[[32]byte]*utxo.UtxoRec)
	}

	blUnsp := make(map[[32]byte][]*btc.TxOut, 4*len(bl.Txs))

	var wg sync.WaitGroup
	var ver_err_cnt uint32

	for i := range bl.Txs {
		txoutsum, txinsum = 0, 0

		bl.SigopsCost += uint32(btc.WITNESS_SCALE_FACTOR * bl.Txs[i].GetLegacySigOpCount())

		// Check each tx for a valid input, except from the first one
		if i > 0 {
			tx_trusted := bl.Trusted
			if !tx_trusted && TrustedTxChecker != nil && TrustedTxChecker(bl.Txs[i].Hash) {
				tx_trusted = true
			}

			for j := 0; j < len(bl.Txs[i].TxIn); j++ {
				inp := &bl.Txs[i].TxIn[j].Input
				spendrec, waspent := changes.DeledTxs[inp.Hash]
				if waspent && spendrec[inp.Vout] {
					println("txin", inp.String(), "already spent in this block")
					e = errors.New("Input spent more then once in same block")
					return
				}
				tout := ch.PickUnspent(inp)
				if tout == nil {
					t, ok := blUnsp[inp.Hash]
					if !ok {
						e = errors.New("Unknown input TxID: " + btc.NewUint256(inp.Hash[:]).String())
						return
					}

					if inp.Vout >= uint32(len(t)) {
						println("Vout too big", len(t), inp.String())
						e = errors.New("Vout too big")
						return
					}

					if t[inp.Vout] == nil {
						println("Vout already spent", inp.String())
						e = errors.New("Vout already spent")
						return
					}

					if t[inp.Vout].WasCoinbase {
						e = errors.New("Cannot spend block's own coinbase in TxID: " + btc.NewUint256(inp.Hash[:]).String())
						return
					}

					tout = t[inp.Vout]
					t[inp.Vout] = nil // and now mark it as spent:
				} else {
					if tout.WasCoinbase && changes.Height-tout.BlockHeight < COINBASE_MATURITY {
						e = errors.New("Trying to spend prematured coinbase: " + btc.NewUint256(inp.Hash[:]).String())
						return
					}
					// it is confirmed already so delete it later
					if !waspent {
						spendrec = make([]bool, tout.VoutCount)
						changes.DeledTxs[inp.Hash] = spendrec
					}
					spendrec[inp.Vout] = true

					if changes.UndoData != nil {
						var urec *utxo.UtxoRec
						urec = changes.UndoData[inp.Hash]
						if urec == nil {
							urec = new(utxo.UtxoRec)
							urec.TxID = inp.Hash
							urec.Coinbase = tout.WasCoinbase
							urec.InBlock = tout.BlockHeight
							urec.Outs = make([]*utxo.UtxoTxOut, tout.VoutCount)
							changes.UndoData[inp.Hash] = urec
						}
						tmp := new(utxo.UtxoTxOut)
						tmp.Value = tout.Value
						tmp.PKScr = make([]byte, len(tout.Pk_script))
						copy(tmp.PKScr, tout.Pk_script)
						urec.Outs[inp.Vout] = tmp
					}
				}

				if !tx_trusted { // run VerifyTxScript() in a parallel task
					wg.Add(1)
					go func(prv []byte, amount uint64, i int, tx *btc.Tx) {
						if !script.VerifyTxScript(prv, amount, i, tx, bl.VerifyFlags) {
							atomic.AddUint32(&ver_err_cnt, 1)
						}
						wg.Done()
					}(tout.Pk_script, tout.Value, j, bl.Txs[i])
				}

				if btc.IsP2SH(tout.Pk_script) {
					bl.SigopsCost += uint32(btc.WITNESS_SCALE_FACTOR * btc.GetP2SHSigOpCount(bl.Txs[i].TxIn[j].ScriptSig))
				}

				bl.SigopsCost += uint32(bl.Txs[i].CountWitnessSigOps(j, tout.Pk_script))

				txinsum += tout.Value
			}

			if !tx_trusted {
				wg.Wait()
				if ver_err_cnt > 0 {
					println("VerifyScript failed", ver_err_cnt, "time (s)")
					return errors.New(fmt.Sprint("VerifyScripts failed ", ver_err_cnt, "time (s)"))
				}
			}
		} else {
			// For coinbase tx we need to check (like satoshi) whether the script size is between 2 and 100 bytes
			// (Previously we made sure in CheckBlock() that this was a coinbase type tx)
			if len(bl.Txs[0].TxIn[0].ScriptSig) < 2 || len(bl.Txs[0].TxIn[0].ScriptSig) > 100 {
				return errors.New(fmt.Sprint("Coinbase script has a wrong length ", len(bl.Txs[0].TxIn[0].ScriptSig)))
			}
		}
		sumblockin += txinsum

		for j := range bl.Txs[i].TxOut {
			txoutsum += bl.Txs[i].TxOut[j].Value
		}
		sumblockout += txoutsum

		if e != nil {
			return // If any input fails, do not continue
		}
		if i > 0 {
			bl.Txs[i].Fee = txinsum - txoutsum
			if txoutsum > txinsum {
				return errors.New(fmt.Sprintf("More spent (%.8f) than at the input (%.8f) in TX %s",
					float64(txoutsum)/1e8, float64(txinsum)/1e8, bl.Txs[i].Hash.String()))
			}
		}

		// Add each tx outs from the currently executed TX to the temporary pool
		outs := make([]*btc.TxOut, len(bl.Txs[i].TxOut))
		copy(outs, bl.Txs[i].TxOut)
		blUnsp[bl.Txs[i].Hash.Hash] = outs
	}

	if sumblockin < sumblockout {
		return errors.New(fmt.Sprintf("Out:%d > In:%d", sumblockout, sumblockin))
	}

	if bl.SigopsCost > btc.MAX_BLOCK_SIGOPS_COST {
		return errors.New("commitTxs(): too many sigops - RPC_Result:bad-blk-sigops")
	}

	var rec *utxo.UtxoRec
	for k, v := range blUnsp {
		for i := range v {
			if v[i] != nil {
				if rec == nil {
					rec = new(utxo.UtxoRec)
					rec.TxID = k
					rec.Coinbase = v[i].WasCoinbase
					rec.InBlock = changes.Height
					rec.Outs = make([]*utxo.UtxoTxOut, len(v))
				}
				rec.Outs[i] = &utxo.UtxoTxOut{Value: v[i].Value, PKScr: v[i].Pk_script}
			}
		}
		if rec != nil {
			changes.AddList = append(changes.AddList, rec)
			rec = nil
		}
	}

	return nil
}

// Check transactions for consistency and finality. Return true if OK
func CheckTransactions(txs []*btc.Tx, height, btime uint32) bool {
	var wg sync.WaitGroup
	var ver_err_cnt uint32
	for i := range txs {
		wg.Add(1)
		go func(tx *btc.Tx) {
			if tx.CheckTransaction() != nil || !tx.IsFinal(height, btime) {
				atomic.AddUint32(&ver_err_cnt, 1)
			}
			wg.Done()
		}(txs[i])
	}
	wg.Wait()
	return ver_err_cnt == 0
}
