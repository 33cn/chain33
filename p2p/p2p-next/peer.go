package p2p_next

import (
	pb "github.com/33cn/chain33/types"
)

type Peer struct {
}

func (p *Peer) GetPeerInfo() (*pb.P2PPeerInfo, error) {
	return nil, nil
}
