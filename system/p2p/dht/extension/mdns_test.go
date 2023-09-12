package extension

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_mdns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := getNetHosts(2, t)
	tmdns, err := NewMDNS(ctx, hosts[0], "33test123")
	require.Nil(t, err)
	require.Nil(t, tmdns.Service.Start())
	//_, err = NewMDNS(ctx, hosts[1], "33test123")
	//require.Nil(t, err)
	tmdns, err = NewMDNS(ctx, hosts[1], "33test123")
	require.Nil(t, err)
	require.Nil(t, tmdns.Service.Start())

	select {
	case peerinfo := <-tmdns.PeerChan():
		require.Equal(t, hosts[0].ID(), peerinfo.ID)
	case <-time.After(time.Second * 10):
		t.Error("mdns discovery failed, timeout")
	}

}
