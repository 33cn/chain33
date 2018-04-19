package relayd

import (
	"sync"

	//"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcwallet/chain"
	"net/http"
)

type BtcPeer struct {
	ipPort string
}

type ChainPeer struct {
}

// confirmed checks whether a transaction at height txHeight has met minconf
// confirmations for a blockchain at height curHeight.
func confirmed(minconf, txHeight, curHeight int32) bool {
	return confirms(txHeight, curHeight) >= minconf
}

func confirms(txHeight, curHeight int32) int32 {
	switch {
	case txHeight == -1, txHeight > curHeight:
		return 0
	default:
		return curHeight - txHeight + 1
	}
}

type btcConnection struct {
}

type chain33Connection struct {
}

type Relayd struct {
	httpServer http.Server
	db         RelaydDB

	btcPeers   BtcPeer
	chainPeers ChainPeer
	mu         sync.Mutex
	bestBlock  btcjson.GetBestBlockResult

	btcClient     *chain.RPCClient
	btcClientLock sync.Mutex

	chainClient     Chain33Client
	chainClientLock sync.Mutex

	wg      sync.WaitGroup
	started bool
	quit    chan struct{}
	quitMu  sync.Mutex
}

func (r *Relayd) GetTransaction() {

}

func (r *Relayd) GetBestBlockHeader() {

}

func (r *Relayd) GetBestBlock() {

}

func (r *Relayd) ShuttingDown() bool {

	return true
}

func (r *Relayd) Ping() error {
	return nil
}
