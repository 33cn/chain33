package relayd

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const SETUP = 1000

var (
	currentBlockHashKey chainhash.Hash
	executor            = []byte("relay")
	zeroBlockHeader     = []byte("")
)
