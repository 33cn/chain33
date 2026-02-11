package gg18

import (
	"fmt"
	"sync"

	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/getamis/alice/types"
)

// Backend 底层处理组件
type Backend interface {
	AddMessage(senderId string, msg types.Message) error
}

type sessionCore struct {
	backend Backend
}

var (
	sessionsMu                   sync.RWMutex
	sessions                     = make(map[string]*sessionCore)
	maxPendingSessions           = 100
	pendingMessages              = make(map[string][]types.Message, maxPendingSessions)
	maxPendingMessagesPerSession = 32
)

func addMessage(protocol, sessionID string, msg types.Message) error {
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	session, ok := sessions[id]
	if ok {
		return session.backend.AddMessage(msg.GetId(), msg)
	}
	if len(pendingMessages[id]) >= maxPendingMessagesPerSession {
		return fmt.Errorf("addMessage max pending messages reached")
	}
	if len(pendingMessages) >= maxPendingSessions {
		log.Debug("addMessage max pending sessions reached, clear pending messages")
		for id := range pendingMessages {
			delete(pendingMessages, id)
		}
	}
	pendingMessages[id] = append(pendingMessages[id], msg)
	return nil
}

func registerSession(protocol, sessionID string, backend Backend) error {
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	_, ok := sessions[id]
	if ok {
		return fmt.Errorf("session already registered")
	}
	sessions[id] = &sessionCore{
		backend: backend,
	}
	sessionsMu.Unlock()
	// flush buffer messages
	for _, msg := range pendingMessages[id] {
		err := backend.AddMessage(msg.GetId(), msg)
		if err != nil {
			log.Error("registerSession", "session", sessionID, "Cannot add pending message to core, err", err)
			return err
		}
	}
	delete(pendingMessages, id)
	return nil
}

func removeSession(protocol, sessionID string) {
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	delete(sessions, id)
}
