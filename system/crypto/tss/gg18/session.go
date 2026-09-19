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

// registerSession 把 backend 绑定到 protocol|sessionID，并回灌"注册前到达"的缓存消息。
//
// 注册必须是原子的（全有或全无）：会话名是各节点约定的同一个常量（进程重启/重试都要复用
// 同一个名字），所以一轮失败后重试必然用**同一个** sessionID 再次注册。原先先写入 sessions
// 再回灌，回灌失败就直接返回 ⇒ 半个 session 留在了注册表里，后续所有重试都撞
// "session already registered"，节点一次超时之后再无恢复可能（与 cggmp 的同一缺陷，见
// cggmp/session.go 的说明）。这里在回灌失败时回滚。
//
// 缓存批次在回灌前就整体取出，无论成败都不再保留：核心拒收某条消息说明这批混了两轮
// （alice 的 echo 层按 (消息类型, 发送方) 记账，同一键的第二个不同消息体返回
// ErrDifferentHash），同一批里其余消息同样不可信；丢掉它，下一次尝试才是干净的一轮。
func registerSession(protocol, sessionID string, backend Backend) error {
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	_, ok := sessions[id]
	if ok {
		return fmt.Errorf("session already registered")
	}
	pending := pendingMessages[id]
	delete(pendingMessages, id)
	sessions[id] = &sessionCore{
		backend: backend,
	}
	// flush buffer messages
	for _, pendingMsg := range pending {
		err := backend.AddMessage(pendingMsg.peerID, pendingMsg.msg)
		if err != nil {
			log.Error("registerSession", "session", sessionID, "Cannot add pending message to core, err", err)
			delete(sessions, id)
			return err
		}
	}
	return nil
}

// removeSession 注销已结束（或已失败）的会话，并清掉它遗留的缓存消息：一个 sessionID 只代表
// 一轮，上一轮的残留绝不能被回灌进下一轮（届时它是同一 (消息类型, 发送方) 的第二个不同消息体，
// 会直接打断新一轮，见 registerSession）。
func removeSession(protocol, sessionID string) {
	id := tss.ComposeProtocol(protocol, sessionID)
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	delete(sessions, id)
	delete(pendingMessages, id)
}
