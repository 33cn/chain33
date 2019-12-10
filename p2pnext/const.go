package p2pnext

const (
	MEMPOOL    = "mempool"
	BLOCKCHAIN = "blockchain"
)

const (
	PeerInfo  = "peerinfo"
	Header    = "header"
	Download  = "download"
	BroadCast = "broadcast"
	NetInfo   = "netinfo"
)

var ProcessName = []string{
	PeerInfo,
	Header,
	Download,
	NetInfo,
	BroadCast,
}
