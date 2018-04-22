package relayd

import (
	"net/http"
	"sync"
	//"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcwallet/chain"
	//con "gitlab.33.cn/chain33/chain33/cmd/relayd/connection"
)

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

type Relayd struct {
	config        *Config
	httpServer    http.Server
	db            RelaydDB
	mu            sync.Mutex
	bestBlock     btcjson.GetBestBlockResult
	btcClient     *chain.RPCClient
	btcClientLock sync.Mutex
	wg            sync.WaitGroup
	started       bool
	quit          chan struct{}
	quitMu        sync.Mutex
	cons33        *Connections33
}

func NewRelayd(config *Config) {

}

func (r *Relayd) ShuttingDown() bool {

	return true
}

// Ping
func (r *Relayd) Ping() error {
	return nil
}
