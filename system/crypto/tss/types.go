package tss

type Message struct {
	Name string
	Data interface{}
}

type DKGRequest struct {
	Rank      uint32
	Threshold uint32
	PeerIDs   []string
}

type DKGResult struct {
}
