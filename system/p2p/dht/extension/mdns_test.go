package extension

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_mdns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := getNetHosts(2, t)
	_, err := NewMDNS(ctx, hosts[0], "33test123")
	require.Nil(t, err)
	tmdns, err := NewMDNS(ctx, hosts[1], "33test123")
	require.Nil(t, err)

	select {
	case peerinfo := <-tmdns.PeerChan():
		require.Equal(t, hosts[0].ID(), peerinfo.ID)
	case <-time.After(time.Second * 10):
		t.Error("mdns discovery failed, timeout")
	}

}
