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
	if wMsg.Protocol != DkgProtocol {
		log.Error("handleDkgMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "invalid protocol", wMsg.Protocol)
		return
	}
	msg := &dkg.Message{}
	err := types.Decode(wMsg.Msg, msg)
	if err != nil {
		log.Error("handleDkgMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "decode msg err", err)
		return
	}
	err = addMessage(DkgProtocol, wMsg.SessionID, msg)
	if err != nil {
		log.Error("handleDkgMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "Cannot add message to core, err", err)
	}

}

func handleSignMsg(wMsg *tss.MessageWrapper) {

	if wMsg.Protocol != SignProtocol {
		log.Error("handleSignMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "invalid protocol", wMsg.Protocol)
		return
	}
	msg := &signer.Message{}
	err := types.Decode(wMsg.Msg, msg)
	if err != nil {
		log.Error("handleSignMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "decode msg err", err)
		return
	}
	err = addMessage(SignProtocol, wMsg.SessionID, msg)
	if err != nil {
		log.Error("handleSignMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "Cannot add message to core, err", err)
	}
}

func handleReshareMsg(wMsg *tss.MessageWrapper) {
	if wMsg.Protocol != ReshareProtocol {
		log.Error("handleReshareMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "invalid protocol", wMsg.Protocol)
		return
	}
	msg := &reshare.Message{}
	err := types.Decode(wMsg.Msg, msg)
	if err != nil {
		log.Error("handleReshareMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "decode msg err", err)
		return
	}
	err = addMessage(ReshareProtocol, wMsg.SessionID, msg)
	if err != nil {
		log.Error("handleReshareMsg", "peerID", wMsg.PeerID, "session", wMsg.SessionID, "Cannot add message to core, err", err)
	}
}
