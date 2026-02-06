package tss

import (
	"math/big"
	"strings"

	"github.com/getamis/alice/crypto/birkhoffinterpolation"
	"github.com/getamis/alice/crypto/ecpointgrouplaw"
	"github.com/getamis/alice/crypto/elliptic"
	"github.com/getamis/alice/crypto/tss/dkg"
)

// MessageWrapper tss message wrapper
type MessageWrapper struct {
	PeerID    string
	Protocol  string
	SessionID string
	Msg       []byte
}

type DKGRequest struct {
	Rank      uint32
	Threshold uint32
	PeerIDs   []string
}

const sessionProtocolSeparator = "|"

// ComposeProtocol combines protocol and session for transport.
func ComposeProtocol(protocol, sessionID string) string {
	if sessionID == "" {
		return protocol
	}
	return protocol + sessionProtocolSeparator + sessionID
}

// SplitProtocol extracts protocol and session from transport form.
func SplitProtocol(protocol string) (string, string) {
	if protocol == "" {
		return "", ""
	}
	idx := strings.LastIndex(protocol, sessionProtocolSeparator)
	if idx <= 0 || idx >= len(protocol)-1 {
		return protocol, ""
	}
	return protocol[:idx], protocol[idx+1:]
}

// NewDKGResult new value
func NewDKGResult(res *dkg.Result) *DKGResult {
	d := &DKGResult{}
	d.PubX = res.PublicKey.GetX().Bytes()
	d.PubY = res.PublicKey.GetY().Bytes()
	d.Share = res.Share.Bytes()
	d.Bks = make(map[string]*BK, len(res.Bks))
	for id, bk := range res.Bks {
		d.Bks[id] = &BK{
			Rank: bk.GetRank(),
			X:    bk.GetX().Bytes(),
		}
	}
	return d
}

// ConvertDKGResult converts DKG result from proto type
func ConvertDKGResult(res *DKGResult) (*dkg.Result, error) {

	// Build public key.
	x := new(big.Int).SetBytes(res.PubX)
	y := new(big.Int).SetBytes(res.PubY)
	pubkey, err := ecpointgrouplaw.NewECPoint(elliptic.Secp256k1(), x, y)
	if err != nil {
		log.Error("Cannot get public key", "err", err)
		return nil, err
	}
	// Build share.
	share := new(big.Int).SetBytes(res.Share)
	dkgResult := &dkg.Result{
		PublicKey: pubkey,
		Share:     share,
		Bks:       make(map[string]*birkhoffinterpolation.BkParameter),
	}
	// Build bks.
	for peerID, bk := range res.Bks {
		x := new(big.Int).SetBytes(bk.X)
		dkgResult.Bks[peerID] = birkhoffinterpolation.NewBkParameter(x, bk.Rank)
	}

	return dkgResult, nil
}
