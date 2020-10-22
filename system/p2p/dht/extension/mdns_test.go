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

	hosts := getNetHosts(ctx, 3, t)
	connect(t, hosts[0], hosts[1])
	_, err := NewMDNS(ctx, hosts[0], "33test123")
	require.Nil(t, err)
	_, err = NewMDNS(ctx, hosts[1], "33test123")
	require.Nil(t, err)
	tmdns, err := NewMDNS(ctx, hosts[2], "33test123")
	require.Nil(t, err)

	select {
	case peerinfo := <-tmdns.PeerChan():
		t.Log("findMdns", peerinfo.ID)
	case <-time.After(time.Second * 10):
		return
	}

}
