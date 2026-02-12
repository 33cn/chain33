package gg18

import (
	"fmt"
	"math/big"

	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/getamis/alice/crypto/elliptic"
	"github.com/getamis/alice/crypto/homo/paillier"
	"github.com/getamis/alice/crypto/tss/dkg"
	"github.com/getamis/alice/crypto/tss/ecdsa/gg18/reshare"
	"github.com/getamis/alice/crypto/tss/ecdsa/gg18/signer"
)

// ProcessDKG process dkg，return DKGResult
// sessionID 各节点同一Dkg的sessionID相同
func ProcessDKG(peers []string, threshold, rank uint32, sessionID string) (*tss.DKGResult, error) {

	pm := tss.NewReadyPeerManager(peers, DkgProtocol, sessionID)
	listener := tss.NewListener(DkgProtocol)
	dkgCore, err := dkg.NewDKG(elliptic.Secp256k1(), pm, threshold, rank, listener)
	if err != nil {
		log.Error("ProcessDKG", "session", sessionID, "NewDKG err", err)
		return nil, err
	}
	err = registerSession(DkgProtocol, sessionID, dkgCore)
	if err != nil {
		log.Error("ProcessDKG", "session", sessionID, "register session failed, err", err)
		return nil, err
	}
	defer removeSession(DkgProtocol, sessionID)
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
// sessionID 各节点同一Sign的sessionID需相同
// 门限签名peers可以是部分节点，需保证调用时各节点peers参数相同
func ProcessSign(peers []string, msg []byte, result *tss.DKGResult, threshold uint32, sessionID string) (*signer.Result, error) {

	dkgResult, err := tss.ConvertDKGResult(peers, result)
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "ConvertDKGResult err", err)
		return nil, err
	}
	pm := tss.NewReadyPeerManager(peers, SignProtocol, sessionID)
	listener := tss.NewListener(SignProtocol)
	homo, err := paillier.NewPaillier(2048)
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "NewPaillier err", err)
		return nil, err
	}
	signerCore, err := signer.NewSigner(pm, dkgResult.PublicKey, homo, dkgResult.Share,
		dkgResult.Bks, msg, listener)
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "NewSigner err", err)
		return nil, err
	}
	err = registerSession(SignProtocol, sessionID, signerCore)
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "register session failed, err", err)
		return nil, err
	}
	defer removeSession(SignProtocol, sessionID)
	signerCore.Start()
	defer signerCore.Stop()

	if err := <-listener.Done(); err != nil {
		log.Error("ProcessSign", "session", sessionID, "listener done err", err)
		return nil, err
	}
	signResult, err := signerCore.GetResult()
	if err != nil {
		log.Error("ProcessSign", "session", sessionID, "GetResult err", err)
		return nil, err
	}
	return signResult, nil
}

// ProcessReshare reshare public key
// sessionID 各节点同一Reshare的sessionID需相同
func ProcessReshare(peers []string, result *tss.DKGResult, threshold uint32, sessionID string) (*reshare.Result, error) {
	dkgRes, err := tss.ConvertDKGResult(peers, result)
	if err != nil {
		log.Error("ProcessReshare", "session", sessionID, "ConvertDKGResult err", err)
		return nil, err
	}
	pm := tss.NewReadyPeerManager(peers, ReshareProtocol, sessionID)
	listener := tss.NewListener(ReshareProtocol)
	reshareCore, err := reshare.NewReshare(pm, threshold, dkgRes.PublicKey, dkgRes.Share,
		dkgRes.Bks, listener)

	if err != nil {
		log.Error("ProcessReshare", "session", sessionID, "NewReshare err", err)
		return nil, err
	}
	err = registerSession(ReshareProtocol, sessionID, reshareCore)
	if err != nil {
		log.Error("ProcessReshare", "session", sessionID, "register session failed, err", err)
		return nil, err
	}
	defer removeSession(ReshareProtocol, sessionID)
	reshareCore.Start()
	defer reshareCore.Stop()

	if err := <-listener.Done(); err != nil {
		log.Error("ProcessReshare", "session", sessionID, "listener done err", err)
		return nil, err
	}
	return reshareCore.GetResult()
}

// bigInt 转 ModNScalar
func bigIntToModNScalar(val *big.Int) (*btcec.ModNScalar, error) {
	var scalar btcec.ModNScalar

	// 转为 32 字节大端序
	b := val.Bytes()
	if len(b) > 32 {
		return nil, fmt.Errorf("value exceeds 32 bytes")
	}

	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)

	// SetByteSlice 返回 true 表示溢出
	if scalar.SetByteSlice(padded) {
		return nil, fmt.Errorf("modulus overflow")
	}

	return &scalar, nil
}

// AliceToBtcecSignature converts Alice signer result to btcec Signature.
func AliceToBtcecSignature(result *signer.Result) (*ecdsa.Signature, error) {
	rScalar, err := bigIntToModNScalar(result.R)
	if err != nil {
		return nil, fmt.Errorf("convert R failed: %w", err)
	}

	sScalar, err := bigIntToModNScalar(result.S)
	if err != nil {
		return nil, fmt.Errorf("convert S failed: %w", err)
	}

	return ecdsa.NewSignature(rScalar, sScalar), nil
}
