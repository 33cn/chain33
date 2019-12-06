package p2p_next

const (
	MEMPOOL    = "mempool"
	BLOCKCHAIN = "blockchain"
)

const (
	PeerInfo       = "peerinfo"
	Header         = "header"
	Download       = "download"
	BroadCastTx    = "broadcastTx"
	BroadCastBlock = "broadcastblock"
	NetInfo        = "netinfo"
)

var ProcessName = []string{
	PeerInfo,
	Header,
	Download,
	BroadCastTx,
	BroadCastBlock,
	NetInfo,
}
