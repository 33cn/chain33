package manage

import "sync"

// manage peer info

type PeerInfoManager struct {
	store sync.Map
}




func NewPeerInfoManager() *PeerInfoManager {

	return &PeerInfoManager{}
}
