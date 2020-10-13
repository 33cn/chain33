package p2pstore

func (p *Protocol) updateHealthyRoutingTable() {
	header, err := p.API.GetLastHeader()
	if err != nil {
		log.Error("updateHealthyRoutingTable", "GetLastHeader error", err)
		return
	}
	for _, pid := range p.RoutingTable.ListPeers() {
		if p.PeerInfoManager.PeerHeight(pid)+50 >= header.Height {
			_, _ = p.healthyRoutingTable.Update(pid)
		}
	}
}
