package chain

import (
	"github.com/piotrnar/gocoin/lib/btc"
	"math/big"
)

const (
	POWRetargetSpam = 14 * 24 * 60 * 60 // two weeks
	TargetSpacing   = 10 * 60
	targetInterval  = POWRetargetSpam / TargetSpacing
)

func (ch *Chain) GetNextWorkRequired(lst *BlockTreeNode, ts uint32) (res uint32) {
	// Genesis block
	if lst.Parent == nil {
		return ch.Consensus.MaxPOWBits
	}

	if ((lst.Height + 1) % targetInterval) != 0 {
		// Special difficulty rule for testnet:
		if ch.testnet() {
			// If the new block's timestamp is more than 2* 10 minutes
			// then allow mining of a min-difficulty block.
			if ts > lst.Timestamp()+TargetSpacing*2 {
				return ch.Consensus.MaxPOWBits
			} else {
				// Return the last non-special-min-difficulty-rules-block
				prv := lst
				for prv.Parent != nil && (prv.Height%targetInterval) != 0 && prv.Bits() == ch.Consensus.MaxPOWBits {
					prv = prv.Parent
				}
				return prv.Bits()
			}
		}
		return lst.Bits()
	}

	prv := lst
	for i := 0; i < targetInterval-1; i++ {
		prv = prv.Parent
	}

	actualTimespan := int64(lst.Timestamp() - prv.Timestamp())

	if actualTimespan < POWRetargetSpam/4 {
		actualTimespan = POWRetargetSpam / 4
	}
	if actualTimespan > POWRetargetSpam*4 {
		actualTimespan = POWRetargetSpam * 4
	}

	// Retarget
	bnewbn := btc.SetCompact(lst.Bits())
	bnewbn.Mul(bnewbn, big.NewInt(actualTimespan))
	bnewbn.Div(bnewbn, big.NewInt(POWRetargetSpam))

	if bnewbn.Cmp(ch.Consensus.MaxPOWValue) > 0 {
		bnewbn = ch.Consensus.MaxPOWValue
	}

	res = btc.GetCompact(bnewbn)

	return
}
