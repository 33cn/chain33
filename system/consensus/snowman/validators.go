package snowman

import (
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/validators"
)

type vdrSet struct {
	validators.Set
}

func newVdrSet() *vdrSet {

	s := &vdrSet{}
	s.Set = validators.NewSet()
	return s
}

// Sample returns a collection of validatorIDs, potentially with duplicates.
// If sampling the requested size isn't possible, an error will be returned.
func (*vdrSet) Sample(size int) ([]ids.NodeID, error) {

	return nil, nil
}
