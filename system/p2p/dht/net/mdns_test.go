package net

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_mdns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := getNetHosts(ctx, 3, t)
	connect(t, hosts[0], hosts[1])
	_, err := initMDNS(ctx, hosts[0], "33test123")
	assert.Nil(t, err)
	_, err = initMDNS(ctx, hosts[1], "33test123")
	assert.Nil(t, err)
	tmdns, err := initMDNS(ctx, hosts[2], "33test123")
	assert.Nil(t, err)

	select {
	case peerinfo := <-tmdns.PeerChan():
		t.Log("findMdns", peerinfo.ID)
	case <-time.After(time.Second * 10):
		return
	}

}
