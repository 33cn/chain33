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
	dkgCore     *dkg.DKG
	signerCore  *signer.Signer
	reshareCore *reshare.Reshare

	log = log15.New("module", "tss.gg18")
)

func handleDkgMsg(msgData []byte) {

	msg := &dkg.Message{}
	err := types.Decode(msgData, msg)
	if err != nil {
		log.Error("handleDkgMsg", "decode msg err", err)
		return
	}

	err = dkgCore.AddMessage(msg.GetId(), msg)
	if err != nil {
		log.Error("handleDkgMsg", "Cannot add message to core, err", err)
	}
}

func handleSignMsg(msgData []byte) {

	msg := &signer.Message{}
	err := types.Decode(msgData, msg)
	if err != nil {
		log.Error("handleSignMsg", "decode msg err", err)
		return
	}

	err = signerCore.AddMessage(msg.GetId(), msg)
	if err != nil {
		log.Error("handleSignMsg", "Cannot add message to core, err", err)
	}
}

func handleReshareMsg(msgData []byte) {

	msg := &reshare.Message{}
	err := types.Decode(msgData, msg)
	if err != nil {
		log.Error("handleReshareMsg", "decode msg err", err)
		return
	}

	err = reshareCore.AddMessage(msg.GetId(), msg)
	if err != nil {
		log.Error("handleReshareMsg", "Cannot add message to core, err", err)
	}

}
