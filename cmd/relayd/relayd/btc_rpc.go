package relayd

import (
	"errors"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	btcrpcclient "github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcwallet/wtxmgr"
)

type BlockStamp struct {
	Height int32
	Hash   chainhash.Hash
}

// RPCClient represents a persistent client connection to a bitcoin RPC server
// for information regarding the current best block chain.
type RPCClient struct {
	*btcrpcclient.Client
	connConfig        *btcrpcclient.ConnConfig // Work around unexported field
	chainParams       *chaincfg.Params
	reconnectAttempts int

	enqueueNotification chan interface{}
	dequeueNotification chan interface{}
	currentBlock        chan *BlockStamp

	quit    chan struct{}
	wg      sync.WaitGroup
	started bool
	quitMtx sync.Mutex
}

// NewRPCClient creates a client connection to the server described by the
// connect string.  If disableTLS is false, the remote RPC certificate must be
// provided in the certs slice.  The connection is not established immediately,
// but must be done using the Start method.  If the remote server does not
// operate on the same bitcoin network as described by the passed chain
// parameters, the connection will be disconnected.
func NewRPCClient(chainParams *chaincfg.Params, connect, user, pass string, certs []byte,
	disableTLS bool, reconnectAttempts int) (*RPCClient, error) {

	if reconnectAttempts < 0 {
		return nil, errors.New("reconnectAttempts must be positive")
	}

	client := &RPCClient{
		connConfig: &btcrpcclient.ConnConfig{
			Host:                 connect,
			Endpoint:             "ws",
			User:                 user,
			Pass:                 pass,
			Certificates:         certs,
			DisableAutoReconnect: true,
			DisableConnectOnNew:  true,
			DisableTLS:           disableTLS,
		},
		chainParams:         chainParams,
		reconnectAttempts:   reconnectAttempts,
		enqueueNotification: make(chan interface{}),
		dequeueNotification: make(chan interface{}),
		currentBlock:        make(chan *BlockStamp),
		quit:                make(chan struct{}),
	}
	ntfnCallbacks := &btcrpcclient.NotificationHandlers{
		OnClientConnected:   client.onClientConnect,
		OnBlockConnected:    client.onBlockConnected,
		OnBlockDisconnected: client.onBlockDisconnected,
		OnRecvTx:            client.onRecvTx,
		OnRedeemingTx:       client.onRedeemingTx,
		OnRescanFinished:    client.onRescanFinished,
		OnRescanProgress:    client.onRescanProgress,
	}
	rpcClient, err := btcrpcclient.New(client.connConfig, ntfnCallbacks)
	if err != nil {
		return nil, err
	}
	client.Client = rpcClient
	return client, nil
}

// Start attempts to establish a client connection with the remote server.
// If successful, handler goroutines are started to process notifications
// sent by the server.  After a limited number of connection attempts, this
// function gives up, and therefore will not block forever waiting for the
// connection to be established to a server that may not exist.
func (c *RPCClient) Start() error {
	err := c.Connect(c.reconnectAttempts)
	if err != nil {
		return err
	}

	// Verify that the server is running on the expected network.
	net, err := c.GetCurrentNet()
	if err != nil {
		c.Disconnect()
		return err
	}
	if net != c.chainParams.Net {
		c.Disconnect()
		return errors.New("mismatched networks")
	}

	c.quitMtx.Lock()
	c.started = true
	c.quitMtx.Unlock()

	c.wg.Add(1)
	go c.handler()
	return nil
}

// Stop disconnects the client and signals the shutdown of all goroutines
// started by Start.
func (c *RPCClient) Stop() {
	c.quitMtx.Lock()
	select {
	case <-c.quit:
	default:
		close(c.quit)
		c.Client.Shutdown()

		if !c.started {
			close(c.dequeueNotification)
		}
	}
	c.quitMtx.Unlock()
}

// WaitForShutdown blocks until both the client has finished disconnecting
// and all handlers have exited.
func (c *RPCClient) WaitForShutdown() {
	c.Client.WaitForShutdown()
	c.wg.Wait()
}

// Notification types.  These are defined here and processed from from reading
// a notificationChan to avoid handling these notifications directly in
// btcrpcclient callbacks, which isn't very Go-like and doesn't allow
// blocking client calls.
type (
	// ClientConnected is a notification for when a client connection is
	// opened or reestablished to the chain server.
	ClientConnected struct{}

	// BlockConnected is a notification for a newly-attached block to the
	// best chain.
	BlockConnected wtxmgr.BlockMeta

	// BlockDisconnected is a notifcation that the block described by the
	// BlockStamp was reorganized out of the best chain.
	BlockDisconnected wtxmgr.BlockMeta

	// RelevantTx is a notification for a transaction which spends wallet
	// inputs or pays to a watched address.
	RelevantTx struct {
		TxRecord *wtxmgr.TxRecord
		Block    *wtxmgr.BlockMeta // nil if unmined
	}

	// RescanProgress is a notification describing the current status
	// of an in-progress rescan.
	RescanProgress struct {
		Hash   *chainhash.Hash
		Height int32
		Time   time.Time
	}

	// RescanFinished is a notification that a previous rescan request
	// has finished.
	RescanFinished struct {
		Hash   *chainhash.Hash
		Height int32
		Time   time.Time
	}
)

// Notifications returns a channel of parsed notifications sent by the remote
// bitcoin RPC server.  This channel must be continually read or the process
// may abort for running out memory, as unread notifications are queued for
// later reads.
func (c *RPCClient) Notifications() <-chan interface{} {
	return c.dequeueNotification
}

// BlockStamp returns the latest block notified by the client, or an error
// if the client has been shut down.
func (c *RPCClient) BlockStamp() (*BlockStamp, error) {
	select {
	case bs := <-c.currentBlock:
		return bs, nil
	case <-c.quit:
		return nil, errors.New("disconnected")
	}
}

// parseBlock parses a btcws definition of the block a tx is mined it to the
// Block structure of the wtxmgr package, and the block index.  This is done
// here since btcrpcclient doesn't parse this nicely for us.
func parseBlock(block *btcjson.BlockDetails) (*wtxmgr.BlockMeta, error) {
	if block == nil {
		return nil, nil
	}
	blkHash, err := chainhash.NewHashFromStr(block.Hash)
	if err != nil {
		return nil, err
	}
	blk := &wtxmgr.BlockMeta{
		Block: wtxmgr.Block{
			Height: block.Height,
			Hash:   *blkHash,
		},
		Time: time.Unix(block.Time, 0),
	}
	return blk, nil
}

func (c *RPCClient) onClientConnect() {
	select {
	case c.enqueueNotification <- ClientConnected{}:
	case <-c.quit:
	}
}

func (c *RPCClient) onBlockConnected(hash *chainhash.Hash, height int32, time time.Time) {
	select {
	case c.enqueueNotification <- BlockConnected{
		Block: wtxmgr.Block{
			Hash:   *hash,
			Height: height,
		},
		Time: time,
	}:
	case <-c.quit:
	}
}

func (c *RPCClient) onBlockDisconnected(hash *chainhash.Hash, height int32, time time.Time) {
	select {
	case c.enqueueNotification <- BlockDisconnected{
		Block: wtxmgr.Block{
			Hash:   *hash,
			Height: height,
		},
		Time: time,
	}:
	case <-c.quit:
	}
}

//OnRecvTx func(transaction *btcutil.Tx, details *btcjson.BlockDetails)
func (c *RPCClient) onRecvTx(tx *btcutil.Tx, block *btcjson.BlockDetails) {
	blk, err := parseBlock(block)
	if err != nil {
		// Log and drop improper notification.
		//log.Errorf("recvtx notification bad block: %v", err)
		return
	}

	rec, err := wtxmgr.NewTxRecordFromMsgTx(tx.MsgTx(), time.Now())
	if err != nil {
		//log.Errorf("Cannot create transaction record for relevant "+
		//	"tx: %v", err)
		return
	}
	select {
	case c.enqueueNotification <- RelevantTx{rec, blk}:
	case <-c.quit:
	}
}

func (c *RPCClient) onRedeemingTx(tx *btcutil.Tx, block *btcjson.BlockDetails) {
	// Handled exactly like recvtx notifications.
	c.onRecvTx(tx, block)
}

func (c *RPCClient) onRescanProgress(hash *chainhash.Hash, height int32, blkTime time.Time) {
	select {
	case c.enqueueNotification <- &RescanProgress{hash, height, blkTime}:
	case <-c.quit:
	}
}

func (c *RPCClient) onRescanFinished(hash *chainhash.Hash, height int32, blkTime time.Time) {
	select {
	case c.enqueueNotification <- &RescanFinished{hash, height, blkTime}:
	case <-c.quit:
	}

}

// handler maintains a queue of notifications and the current state (best
// block) of the chain.
func (c *RPCClient) handler() {
	hash, height, err := c.GetBestBlock()
	if err != nil {
		//log.Errorf("Failed to receive best block from chain server: %v", err)
		c.Stop()
		c.wg.Done()
		return
	}

	bs := &BlockStamp{Hash: *hash, Height: height}

	// TODO: Rather than leaving this as an unbounded queue for all types of
	// notifications, try dropping ones where a later enqueued notification
	// can fully invalidate one waiting to be processed.  For example,
	// blockconnected notifications for greater block heights can remove the
	// need to process earlier blockconnected notifications still waiting
	// here.

	var notifications []interface{}
	enqueue := c.enqueueNotification
	var dequeue chan interface{}
	var next interface{}
	pingChan := time.After(time.Minute)
out:
	for {
		select {
		case n, ok := <-enqueue:
			if !ok {
				// If no notifications are queued for handling,
				// the queue is finished.
				if len(notifications) == 0 {
					break out
				}
				// nil channel so no more reads can occur.
				enqueue = nil
				continue
			}
			if len(notifications) == 0 {
				next = n
				dequeue = c.dequeueNotification
			}
			notifications = append(notifications, n)
			pingChan = time.After(time.Minute)

		case dequeue <- next:
			if n, ok := next.(BlockConnected); ok {
				bs = &BlockStamp{
					Height: n.Height,
					Hash:   n.Hash,
				}
			}

			notifications[0] = nil
			notifications = notifications[1:]
			if len(notifications) != 0 {
				next = notifications[0]
			} else {
				// If no more notifications can be enqueued, the
				// queue is finished.
				if enqueue == nil {
					break out
				}
				dequeue = nil
			}

		case <-pingChan:
			// No notifications were received in the last 60s.
			// Ensure the connection is still active by making a new
			// request to the server.
			// TODO: A minute timeout is used to prevent the handler
			// loop from blocking here forever, but this is much larger
			// than it needs to be due to btcd processing websocket
			// requests synchronously (see
			// https://github.com/btcsuite/btcd/issues/504).  Decrease
			// this to something saner like 3s when the above issue is
			// fixed.
			type sessionResult struct {
				err error
			}
			sessionResponse := make(chan sessionResult, 1)
			go func() {
				_, err := c.Session()
				sessionResponse <- sessionResult{err}
			}()

			select {
			case resp := <-sessionResponse:
				if resp.err != nil {
					//log.Errorf("Failed to receive session "+"result: %v", resp.err)
					c.Stop()
					break out
				}
				pingChan = time.After(time.Minute)

			case <-time.After(time.Minute):
				//log.Errorf("Timeout waiting for session RPC")
				c.Stop()
				break out
			}

		case c.currentBlock <- bs:

		case <-c.quit:
			break out
		}
	}

	c.Stop()
	close(c.dequeueNotification)
	c.wg.Done()
}

// POSTClient creates the equivalent HTTP POST btcrpcclient.Client.
func (c *RPCClient) POSTClient() (*btcrpcclient.Client, error) {
	configCopy := *c.connConfig
	configCopy.HTTPPostMode = true
	return btcrpcclient.New(&configCopy, nil)
}
