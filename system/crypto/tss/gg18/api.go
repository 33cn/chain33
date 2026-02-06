package gg18

import (
	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/getamis/alice/crypto/elliptic"
	"github.com/getamis/alice/crypto/homo/paillier"
	"github.com/getamis/alice/crypto/tss/dkg"
	"github.com/getamis/alice/crypto/tss/ecdsa/gg18/reshare"
	"github.com/getamis/alice/crypto/tss/ecdsa/gg18/signer"
)

// ProcessDKG process dkg，return DKGResult
func ProcessDKG(peers []string, threshold, rank uint32) (*tss.DKGResult, error) {
	sessionID, session := newSession()
	defer func() {
		session.dkg = nil
		removeSession(sessionID)
	}()
	pm := tss.NewPeerManager(peers, DkgProtocol, sessionID)
	listener := tss.NewListener(DkgProtocol)
	dkgCore, err := dkg.NewDKG(elliptic.Secp256k1(), pm, threshold, rank, listener)
	if err != nil {
		log.Error("ProcessDKG", "session", sessionID, "NewDKG err", err)
		return nil, err
	}
	session.dkg = dkgCore
	pm.EnsureAllConnected()
	dkgCore.Start()
	defer dkgCore.Stop()

	if err := <-listener.Done(); err != nil {
		log.Error("ProcessDKG", "session", sessionID, "listener done err", err)
		return nil, err
	}
	result, err := dkgCore.GetResult()
	if err != nil {
		log.Error("ProcessDKG", "session", sessionID, "GetResult err", err)
		return nil, err
	}
	return tss.NewDKGResult(result), nil
}

// ProcessSign sign message
func ProcessSign(peers []string, msg []byte, result *tss.DKGResult) (*signer.Result, error) {
	sessionID, session := newSession()
	defer func() {
		session.signer = nil
		removeSession(sessionID)
	}()
	pm := tss.NewPeerManager(peers, SignProtocol, sessionID)
	listener := tss.NewListener(SignProtocol)
	homo, err := paillier.NewPaillier(2048)
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "NewPaillier err", err)
		return nil, err
	}
	dkgResult, err := tss.ConvertDKGResult(result)
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "ConvertDKGResult err", err)
		return nil, err
	}
	signerCore, err := signer.NewSigner(pm, dkgResult.PublicKey, homo, dkgResult.Share,
		dkgResult.Bks, msg, listener)
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "NewSigner err", err)
		return nil, err
	}
	session.signer = signerCore
	pm.EnsureAllConnected()
	signerCore.Start()
	defer signerCore.Stop()

	if err := <-listener.Done(); err != nil {
		log.Error("ProcessSign", "session", sessionID, "listener done err", err)
		return nil, err
	}
	return signerCore.GetResult()
}

// ProcessReshare reshare public key
func ProcessReshare(peers []string, result *tss.DKGResult, threshold uint32) (*reshare.Result, error) {
	sessionID, session := newSession()
	defer func() {
		session.reshare = nil
		removeSession(sessionID)
	}()
	dkgRes, err := tss.ConvertDKGResult(result)
	if err != nil {
		log.Error("ProcessReshare", "session", sessionID, "ConvertDKGResult err", err)
		return nil, err
	}
	pm := tss.NewPeerManager(peers, ReshareProtocol, sessionID)
	listener := tss.NewListener(ReshareProtocol)

	reshareCore, err := reshare.NewReshare(pm, threshold, dkgRes.PublicKey, dkgRes.Share,
		dkgRes.Bks, listener)

	if err != nil {
		log.Error("ProcessReshare", "session", sessionID, "NewReshare err", err)
		return nil, err
	}
	session.reshare = reshareCore
	pm.EnsureAllConnected()
	reshareCore.Start()
	defer reshareCore.Stop()

	if err := <-listener.Done(); err != nil {
		log.Error("ProcessReshare", "session", sessionID, "listener done err", err)
		return nil, err
	}
	return reshareCore.GetResult()
}
