package protocol

import (
	"context"
	"strings"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

func NewHost(h host.Host, title string) host.Host {
	return &wrapperHost{
		Host:  h,
		title: title,
	}
}

type wrapperHost struct {
	host.Host
	title string
}

func (h *wrapperHost) SetStreamHandler(pid protocol.ID, handler network.StreamHandler) {
	// 兼容老版本
	h.Host.SetStreamHandler(pid, handler)
	// new
	if title != "" {
		prefix := "/" + h.title
		if strings.HasPrefix(string(pid), prefix) {
			pid = protocol.ID(prefix) + pid
		}
	}
	h.Host.SetStreamHandler(pid, handler)
}

func (h *wrapperHost) SetStreamHandlerMatch(pid protocol.ID, f func(string) bool, handler network.StreamHandler) {
	if title != "" {
		prefix := "/" + h.title
		if strings.HasPrefix(string(pid), prefix) {
			pid = protocol.ID(prefix) + pid
		}
	}
	h.Host.SetStreamHandlerMatch(pid, f, handler)
}
func (h *wrapperHost) RemoveStreamHandler(pid protocol.ID) {
	if title != "" {
		prefix := "/" + h.title
		if strings.HasPrefix(string(pid), prefix) {
			pid = protocol.ID(prefix) + pid
		}
	}
	h.Host.RemoveStreamHandler(pid)
}

func (h *wrapperHost) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error) {
	if title != "" {
		prefix := "/" + h.title
		for i, pid := range pids {
			if strings.HasPrefix(string(pid), prefix) {
				pids[i] = protocol.ID(prefix) + pid
			}
		}
	}
	return h.Host.NewStream(ctx, p, pids...)
}
