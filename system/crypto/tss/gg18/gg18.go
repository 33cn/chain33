package gg18

import (
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/33cn/chain33/types"
	"github.com/getamis/alice/crypto/tss/dkg"
	"github.com/getamis/alice/crypto/tss/ecdsa/gg18/reshare"
	"github.com/getamis/alice/crypto/tss/ecdsa/gg18/signer"
)

func init() {

	tss.RegisterMsgHandler(DkgProtocol, handleDkgMsg)
	tss.RegisterMsgHandler(SignProtocol, handleSignMsg)
	tss.RegisterMsgHandler(ReshareProtocol, handleReshareMsg)
}

const (
	// DkgProtocol dkg protocol
	DkgProtocol = "/gg18/dkg"
	// SignProtocol sign protocol
	SignProtocol = "/gg18/sign"
	// ReshareProtocol reshare protocol
	ReshareProtocol = "/gg18/reshare"
)

var (
	log = log15.New("module", "tss.gg18")
)

func handleDkgMsg(wMsg *tss.MessageWrapper) {

	if wMsg == nil {
		log.Error("handleDkgMsg", "err", "nil message wrapper")
		return
	}
	sessionID := wMsg.SessionID
	if sessionID == "" {
		log.Error("handleDkgMsg", "err", "empty session")
		return
	}
	session, ok := getSession(sessionID)
	if !ok {
		log.Error("handleDkgMsg", "session", sessionID, "err", "session not found")
		return
	}
	msg := &dkg.Message{}
	err := types.Decode(wMsg.Msg, msg)
	if err != nil {
		log.Error("handleDkgMsg", "session", sessionID, "decode msg err", err)
		return
	}

	session.mu.RLock()
	core := session.dkg
	session.mu.RUnlock()
	if core == nil {
		log.Error("handleDkgMsg", "session", sessionID, "err", "dkg core not ready")
		return
	}
	err = core.AddMessage(msg.GetId(), msg)
	if err != nil {
		log.Error("handleDkgMsg", "session", sessionID, "Cannot add message to core, err", err)
	}
}

func handleSignMsg(wMsg *tss.MessageWrapper) {

	if wMsg == nil {
		log.Error("handleSignMsg", "err", "nil message wrapper")
		return
	}
	sessionID := wMsg.SessionID
	if sessionID == "" {
		log.Error("handleSignMsg", "err", "empty session")
		return
	}
	session, ok := getSession(sessionID)
	if !ok {
		log.Error("handleSignMsg", "session", sessionID, "err", "session not found")
		return
	}
	msg := &signer.Message{}
	err := types.Decode(wMsg.Msg, msg)
	if err != nil {
		log.Error("handleSignMsg", "session", sessionID, "decode msg err", err)
		return
	}

	session.mu.RLock()
	core := session.signer
	session.mu.RUnlock()
	if core == nil {
		log.Error("handleSignMsg", "session", sessionID, "err", "signer core not ready")
		return
	}
	err = core.AddMessage(msg.GetId(), msg)
	if err != nil {
		log.Error("handleSignMsg", "session", sessionID, "Cannot add message to core, err", err)
	}
}

func handleReshareMsg(wMsg *tss.MessageWrapper) {

	if wMsg == nil {
		log.Error("handleReshareMsg", "err", "nil message wrapper")
		return
	}
	sessionID := wMsg.SessionID
	if sessionID == "" {
		log.Error("handleReshareMsg", "err", "empty session")
		return
	}
	session, ok := getSession(sessionID)
	if !ok {
		log.Error("handleReshareMsg", "session", sessionID, "err", "session not found")
		return
	}
	msg := &reshare.Message{}
	err := types.Decode(wMsg.Msg, msg)
	if err != nil {
		log.Error("handleReshareMsg", "session", sessionID, "decode msg err", err)
		return
	}

	session.mu.RLock()
	core := session.reshare
	session.mu.RUnlock()
	if core == nil {
		log.Error("handleReshareMsg", "session", sessionID, "err", "reshare core not ready")
		return
	}
	err = core.AddMessage(msg.GetId(), msg)
	if err != nil {
		log.Error("handleReshareMsg", "session", sessionID, "Cannot add message to core, err", err)
	}

}
