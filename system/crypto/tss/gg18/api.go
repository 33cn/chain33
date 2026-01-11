package gg18

import (
	"sync"

	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/getamis/alice/crypto/elliptic"
	"github.com/getamis/alice/crypto/homo/paillier"
	"github.com/getamis/alice/crypto/tss/dkg"
	"github.com/getamis/alice/crypto/tss/ecdsa/gg18/reshare"
	"github.com/getamis/alice/crypto/tss/ecdsa/gg18/signer"
)

var (
	lock sync.RWMutex
)

// ProcessDKG run dkg
func ProcessDKG(peers []string, threshold, rank uint32) (*dkg.Result, error) {
	lock.Lock()
	defer lock.Unlock()
	var err error
	pm := tss.NewPeerManager(peers, DkgProtocol)
	listener := tss.NewListener(DkgProtocol)
	dkgCore, err = dkg.NewDKG(elliptic.Secp256k1(), pm, threshold, rank, listener)
	if err != nil {
		log.Error("ProcessDKG", "NewDKG err", err)
	}
	pm.EnsureAllConnected()
	dkgCore.Start()
	defer dkgCore.Stop()

	if err := <-listener.Done(); err != nil {
		log.Error("ProcessDKG", "listener done err", err)
		return nil, err
	}
	return dkgCore.GetResult()
}

// ProcessSign sign message
func ProcessSign(peers []string, msg []byte, dkgResult *dkg.Result) (*signer.Result, error) {
	lock.Lock()
	defer lock.Unlock()

	var err error
	pm := tss.NewPeerManager(peers, SignProtocol)
	listener := tss.NewListener(SignProtocol)
	homo, err := paillier.NewPaillier(2048)
	if err != nil {
		log.Error("ProcessSign", "NewPaillier err", err)
		return nil, err
	}
	signerCore, err = signer.NewSigner(pm, dkgResult.PublicKey, homo, dkgResult.Share,
		dkgResult.Bks, msg, listener)
	if err != nil {
		log.Error("ProcessSign", "NewSigner err", err)
		return nil, err
	}

	pm.EnsureAllConnected()
	signerCore.Start()
	defer signerCore.Stop()

	if err := <-listener.Done(); err != nil {
		log.Error("ProcessSign", "listener done err", err)
		return nil, err
	}
	return signerCore.GetResult()
}

// ProcessReshare reshare public key
func ProcessReshare(peers []string, dkgRes *dkg.Result, threshold uint32) (*reshare.Result, error) {
	lock.Lock()
	defer lock.Unlock()

	var err error
	pm := tss.NewPeerManager(peers, ReshareProtocol)
	listener := tss.NewListener(ReshareProtocol)

	reshareCore, err = reshare.NewReshare(pm, threshold, dkgRes.PublicKey, dkgRes.Share,
		dkgRes.Bks, listener)

	if err != nil {
		log.Error("ProcessReshare", "NewReshare err", err)
		return nil, err
	}

	pm.EnsureAllConnected()
	reshareCore.Start()
	defer reshareCore.Stop()

	if err := <-listener.Done(); err != nil {
		log.Error("ProcessReshare", "listener done err", err)
		return nil, err
	}
	return reshareCore.GetResult()
}
