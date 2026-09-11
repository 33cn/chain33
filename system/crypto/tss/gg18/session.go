package gg18

import (
	"errors"
	"fmt"
	"sync"

	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/getamis/alice/types"
)

// errMissingPeerID 表示收到的 tss 消息没有传输层认证的对端身份（peer id），
// 无法归属到任何参与者，只能拒收。
var errMissingPeerID = errors.New("gg18: missing authenticated peer id")

// Backend 底层处理组件
type Backend interface {
	AddMessage(senderID string, msg types.Message) error
}

type sessionCore struct {
	backend Backend
}

// pendingMessage 是缓冲的一条消息及其来源 peer id。peer id 在消息到达时就固化下来，
// 因为这些消息会在本地 session 注册后才冲入 alice 协议核心，届时传输层身份已不可得。
type pendingMessage struct {
	peerID string
	msg    types.Message
}

var (
	sessionsMu                   sync.RWMutex
	sessions                     = make(map[string]*sessionCore)
	maxPendingSessions           = 100
	pendingMessages              = make(map[string][]pendingMessage, maxPendingSessions)
	maxPendingMessagesPerSession = 32
)

// addMessage 把收到的 tss 消息交给对应 session 的 alice 协议核心。
// peerID 是传输层已认证的发送方（libp2p peer id，即 tss.MessageWrapper.PeerID），
// 不能使用消息体里的 Id：alice 核心会拒绝"自称身份与归属身份不一致"的消息，
// 而 Id 由发送方自行填写，可被伪造成其他 peer。
func addMessage(protocol, sessionID, peerID string, msg types.Message) error {
	if peerID == "" {
		// 传输层没给出已认证身份时不能退化成信任消息体的 Id，只能拒绝。
		return errMissingPeerID
	}
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	session, ok := sessions[id]
	if ok {
		return session.backend.AddMessage(peerID, msg)
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
	pendingMessages[id] = append(pendingMessages[id], pendingMessage{peerID: peerID, msg: msg})
	return nil
}

func registerSession(protocol, sessionID string, backend Backend) error {
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	_, ok := sessions[id]
	if ok {
		return fmt.Errorf("session already registered")
	}
	sessions[id] = &sessionCore{
		backend: backend,
	}
	// flush buffer messages
	for _, pending := range pendingMessages[id] {
		err := backend.AddMessage(pending.peerID, pending.msg)
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
