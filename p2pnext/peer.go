package p2pnext

import (
	pb "github.com/33cn/chain33/types"
)

type Peer struct {
}

func (p *Peer) GetPeerInfo() (*pb.P2PPeerInfo, error) {
	return nil, nil
}
