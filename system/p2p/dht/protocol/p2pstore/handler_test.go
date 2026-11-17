package p2pstore

import (
	"math"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
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

func TestHandleStreamFetchShardPeersInvalidRequest(t *testing.T) {
	p := &Protocol{}
	cases := []struct {
		name string
		req  *types.P2PRequest
	}{
		{"empty request", &types.P2PRequest{}},
		{"nil reqPeers", &types.P2PRequest{Request: &types.P2PRequest_ReqPeers{}}},
		{"negative count", &types.P2PRequest{Request: &types.P2PRequest_ReqPeers{ReqPeers: &types.ReqPeers{Count: -1}}}},
		{"over max count", &types.P2PRequest{Request: &types.P2PRequest_ReqPeers{ReqPeers: &types.ReqPeers{Count: maxFetchShardPeers + 1}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, p.handleStreamFetchShardPeers(tc.req, &types.P2PResponse{}))
		})
	}
}
