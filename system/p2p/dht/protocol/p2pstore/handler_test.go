package p2pstore

import (
	"math"
	"testing"
)

func TestValidateFetchShardPeersCount(t *testing.T) {
	cases := []struct {
		count int32
		ok    bool
	}{
		{0, true},
		{1, true},
		{maxFetchShardPeers, true},
		{-1, false},
		{-maxFetchShardPeers, false},
		{maxFetchShardPeers + 1, false},
		{math.MaxInt32, false},
	}
	for _, tc := range cases {
		err := validateFetchShardPeersCount(tc.count)
		if tc.ok && err != nil {
			t.Fatalf("count=%d: want nil, got %v", tc.count, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("count=%d: want error", tc.count)
		}
	}
}
