package protocol

import (
	"context"
	"strings"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// NewHost ...
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

// SetStreamHandler wraps host.SetStreamHandler
func (h *wrapperHost) SetStreamHandler(pid protocol.ID, handler network.StreamHandler) {
	// 兼容老版本
	h.Host.SetStreamHandler(pid, handler)
	// new
	if h.title != "" {
		prefix := "/" + h.title
		if strings.HasPrefix(string(pid), prefix) {
			pid = protocol.ID(prefix) + pid
		}
	}
	h.Host.SetStreamHandler(pid, handler)
}

// SetStreamHandlerMatch wraps host.SetStreamHandlerMatch
func (h *wrapperHost) SetStreamHandlerMatch(pid protocol.ID, f func(string) bool, handler network.StreamHandler) {
	if h.title != "" {
		prefix := "/" + h.title
		if strings.HasPrefix(string(pid), prefix) {
			pid = protocol.ID(prefix) + pid
		}
	}
	h.Host.SetStreamHandlerMatch(pid, f, handler)
}

// RemoveStreamHandler wraps host.RemoveStreamHandler
func (h *wrapperHost) RemoveStreamHandler(pid protocol.ID) {
	if h.title != "" {
		prefix := "/" + h.title
		if strings.HasPrefix(string(pid), prefix) {
			pid = protocol.ID(prefix) + pid
		}
	}
	h.Host.RemoveStreamHandler(pid)
}

// NewStream wraps host.NewStream
func (h *wrapperHost) NewStream(ctx context.Context, p peer.ID, pids ...protocol.ID) (network.Stream, error) {
	if h.title != "" {
		prefix := "/" + h.title
		for i, pid := range pids {
			if strings.HasPrefix(string(pid), prefix) {
				pids[i] = protocol.ID(prefix) + pid
			}
		}
	}
	return h.Host.NewStream(ctx, p, pids...)
}
