package tss

// MessageWrapper tss message wrapper
type MessageWrapper struct {
	PeerID   string
	Protocol string
	Msg      []byte
}

type DKGRequest struct {
	Rank      uint32
	Threshold uint32
	PeerIDs   []string
}

type DKGResult struct {
}
