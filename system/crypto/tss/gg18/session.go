package gg18

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getamis/alice/crypto/tss/dkg"
	"github.com/getamis/alice/crypto/tss/ecdsa/gg18/reshare"
	"github.com/getamis/alice/crypto/tss/ecdsa/gg18/signer"
)

type sessionCore struct {
	mu      sync.RWMutex
	dkg     *dkg.DKG
	signer  *signer.Signer
	reshare *reshare.Reshare
}

var (
	sessionsMu sync.RWMutex
	sessions   = make(map[string]*sessionCore)
)


var sessionFallbackCounter uint64

// returns a new unique session identifier.
func newSessionID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	seq := atomic.AddUint64(&sessionFallbackCounter, 1)
	return fmt.Sprintf("%x-%x", time.Now().UnixNano(), seq)
}


func getSession(sessionID string) (*sessionCore, bool) {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	session, ok := sessions[sessionID]
	return session, ok
}

func newSession() (string, *sessionCore) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	sessionID := newSessionID()
	session := &sessionCore{}
	sessions[sessionID] = session
	return sessionID, session
}

func removeSession(sessionID string) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	delete(sessions, sessionID)
}