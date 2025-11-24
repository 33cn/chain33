package protocol

import (
	"fmt"
	"sync"

	"github.com/33cn/chain33/queue"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// RegisterStreamHandler registers stream handler
func RegisterStreamHandler(h host.Host, p protocol.ID, handler network.StreamHandler) {
	if handler == nil {
		panic(fmt.Sprintf("addEventHandler, handler is nil, protocol=%s", p))
	}
	f := func(s network.Stream) {
		if h.ConnManager() != nil {
			h.ConnManager().Protect(s.Conn().RemotePeer(), string(p))
			defer h.ConnManager().Unprotect(s.Conn().RemotePeer(), string(p))
		}
		handler(s)
	}
	h.SetStreamHandler(p, HandlerWithClose(f))
}

// Initializer is a initial function which any protocol should have.
type Initializer func(env *P2PEnv)

var (
	protocolInitializerArray []Initializer
)

// RegisterProtocolInitializer registers the initial function.
func RegisterProtocolInitializer(initializer Initializer) {
	protocolInitializerArray = append(protocolInitializerArray, initializer)
}

// InitAllProtocol initials all protocols.
func InitAllProtocol(env *P2PEnv) {
	for _, initializer := range protocolInitializerArray {
		initializer(env)
	}
}

// EventHandler chain33 internal event handler
type EventHandler struct {
	CallBack EventHandlerFunc
	Inline   bool
}

// EventHandlerFunc event handler call back
type EventHandlerFunc func(*queue.Message)

// EventOpt event options
type EventOpt func(*EventHandler) error

var (
	eventHandlers = make(map[int64]*EventHandler)
	mu            sync.RWMutex
)

// WithEventOptInline invoke event callback inline
func WithEventOptInline(handler *EventHandler) error {
	handler.Inline = true
	return nil
}

// RegisterEventHandler registers a handler with an event ID.
func RegisterEventHandler(eventID int64, cb EventHandlerFunc, opts ...EventOpt) {
	mu.Lock()
	defer mu.Unlock()
	if cb == nil {
		panic(fmt.Sprintf("addEventHandler, handler is nil, id=%d", eventID))
	}
	if _, dup := eventHandlers[eventID]; dup {
		panic(fmt.Sprintf("addEventHandler, duplicate handler, id=%d, len=%d", eventID, len(eventHandlers)))
	}

	handler := &EventHandler{
		CallBack: EventHandlerWithRecover(cb),
	}

	for _, opt := range opts {
		opt(handler)
	}

	eventHandlers[eventID] = handler
}

// GetEventHandler gets event handler by event ID.
func GetEventHandler(eventID int64) *EventHandler {
	mu.RLock()
	defer mu.RUnlock()
	return eventHandlers[eventID]
}

// ClearEventHandler clear event handler map, plugin存在多个p2p实例测试，会导致重复注册，需要清除
func ClearEventHandler() {
	mu.Lock()
	defer mu.Unlock()
	for k := range eventHandlers {
		delete(eventHandlers, k)
	}
}
