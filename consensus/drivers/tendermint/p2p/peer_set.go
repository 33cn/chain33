package p2p

import (
	"sync"
	"errors"
	log "github.com/inconshreveable/log15"
	"github.com/go-playground/locales/sw"
	"fmt"
)

var p2plog = log.New("module", "tendermint")

var (
	ErrSwitchDuplicatePeer = errors.New("Duplicate peer")
)

// IPeerSet has a (immutable) subset of the methods of PeerSet.
type IPeerSet interface {
	Has(key string) bool
	Get(key string) IPeer
	List() []IPeer
	Size() int
}

//-----------------------------------------------------------------------------

// PeerSet is a special structure for keeping a table of peers.
// Iteration over the peers is super fast and thread-safe.
type PeerSet struct {
	mtx    sync.Mutex
	lookup map[string]*peerSetItem
	list   []IPeer
}

type peerSetItem struct {
	peer  IPeer
	index int
}

// NewPeerSet creates a new peerSet with a list of initial capacity of 256 items.
func NewPeerSet() *PeerSet {
	return &PeerSet{
		lookup: make(map[string]*peerSetItem),
		list:   make([]IPeer, 0, 256),
	}
}

// Add adds the peer to the PeerSet.
// It returns ErrSwitchDuplicatePeer if the peer is already present.
func (ps *PeerSet) Add(peer IPeer) error {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	if ps.lookup[peer.Who("")] != nil {
		return ErrSwitchDuplicatePeer
	}

	index := len(ps.list)
	// Appending is safe even with other goroutines
	// iterating over the ps.list slice.
	ps.list = append(ps.list, peer)
	ps.lookup[peer.Who("")] = &peerSetItem{peer, index}
	return nil
}

// Has returns true iff the PeerSet contains
// the peer referred to by this peerKey.
func (ps *PeerSet) Has(peerKey string) bool {
	ps.mtx.Lock()
	_, ok := ps.lookup[peerKey]
	ps.mtx.Unlock()
	return ok
}

// Get looks up a peer by the provided peerKey.
func (ps *PeerSet) Get(peerKey string) IPeer {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	item, ok := ps.lookup[peerKey]
	if ok {
		return item.peer
	} else {
		return nil
	}
}

// Remove discards peer by its Key, if the peer was previously memoized.
func (ps *PeerSet) Remove(peer IPeer) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	item := ps.lookup[peer.Who("")]
	if item == nil {
		return
	}

	index := item.index
	// Create a new copy of the list but with one less item.
	// (we must copy because we'll be mutating the list).
	newList := make([]IPeer, len(ps.list)-1)
	copy(newList, ps.list)
	// If it's the last peer, that's an easy special case.
	if index == len(ps.list)-1 {
		ps.list = newList
		delete(ps.lookup, peer.Who(""))
		return
	}

	// Replace the popped item with the last item in the old list.
	lastPeer := ps.list[len(ps.list)-1]
	lastPeerKey := lastPeer.Who("")
	lastPeerItem := ps.lookup[lastPeerKey]
	newList[index] = lastPeer
	lastPeerItem.index = index
	ps.list = newList
	delete(ps.lookup, peer.Who(""))
}

// Size returns the number of unique items in the peerSet.
func (ps *PeerSet) Size() int {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return len(ps.list)
}

// List returns the threadsafe list of peers.
func (ps *PeerSet) List() []IPeer {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return ps.list
}

// Broadcast runs a go routine for each attempted send, which will block
// trying to send for defaultSendTimeoutSeconds. Returns a channel
// which receives success values for each attempted send (false if times out).
// NOTE: Broadcast uses goroutines, so order of broadcast may not be preserved.
// TODO: Something more intelligent.
func (ps *PeerSet) Broadcast(chID byte, msg interface{}) chan bool {
	successChan := make(chan bool, len(ps.List()))
	p2plog.Debug("Broadcast", "channel", chID, "msg", msg)
	for _, peer := range ps.List() {
		go func(peer Peer) {
			success := peer.WriteMsg(Msg{Service:"", Data:[]byte(fmt.Sprintf("%v:%v",chID, msg))})
			successChan <- success
		}(peer)
	}
	return successChan
}