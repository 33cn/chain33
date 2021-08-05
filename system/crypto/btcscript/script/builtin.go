package script

import (
	"github.com/33cn/chain33/common/log"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
)

var (
	btcLog = log.New("module", "btc.script")
)

// NewMultiSigScript multi-sig pubKey script
func NewMultiSigScript(pubKeys [][]byte, required int) (script []byte, err error) {

	if required <= 0 || required > len(pubKeys) {
		return nil, ErrInvalidMultiSigRequiredNum
	}

	btcAddrs := make([]*btcutil.AddressPubKey, 0, 2)
	for _, pub := range pubKeys {
		addr, err := btcutil.NewAddressPubKey(pub, Chain33BtcParams)
		if err != nil {
			return nil, ErrInvalidBtcPubKey
		}
		btcAddrs = append(btcAddrs, addr)
	}
	return txscript.MultiSigScript(btcAddrs, required)
}

// NewWalletRecoveryScript wallet assets recovery pubKey script
// controlPubKey  secp256k1 pub key
// recoverPubKey  secp256k1 pub key
// relativeDelayTime  relative time of second or block height
// IF <A's Pubkey> CHECKSIG ELSE <sequence> CHECKSEQUENCEVERIFY DROP <B's Pubkey> CHECKSIGVERIFY ENDIF
func NewWalletRecoveryScript(controlPubKey, recoverPubKey []byte, relativeDelayTime int64) (script []byte, err error) {

	ctrAddr, err := btcutil.NewAddressPubKey(controlPubKey, Chain33BtcParams)
	if err != nil {
		return nil, ErrInvalidBtcPubKey
	}
	recovAddr, err := btcutil.NewAddressPubKey(recoverPubKey, Chain33BtcParams)
	if err != nil {
		return nil, ErrInvalidBtcPubKey
	}
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_IF).AddData(ctrAddr.ScriptAddress()).
		AddOp(txscript.OP_CHECKSIG).AddOp(txscript.OP_ELSE).
		AddInt64(relativeDelayTime).AddOp(txscript.OP_CHECKSEQUENCEVERIFY).
		AddOp(txscript.OP_DROP).AddData(recovAddr.ScriptAddress()).
		AddOp(txscript.OP_CHECKSIG).AddOp(txscript.OP_ENDIF)

	script, err = builder.Script()
	if err != nil {
		return nil, ErrBuildBtcScript
	}
	return script, nil
}

// GetWalletRecoverySignature get wallet asset recover signature
// isRetrieve set false when input control address private key, set true for wallet recovery
// signMsg	msg for sign
// privKey  private key of control address or recover address
// walletRecoverScript result of NewWalletRecoveryScript
// utxoSequence utxo sequence, set relative delay time for wallet recovery
func GetWalletRecoverySignature(isRetrieve bool, signMsg, privKey, walletRecoverScript []byte, utxoSequence int64) (sig []byte, pubKey []byte, err error) {

	btcTx := getBindBtcTx(signMsg)
	if !isRetrieve {
		utxoSequence = 0
	}
	setBtcTx(btcTx, 0, utxoSequence, nil)

	key, _ := NewBtcKeyFromBytes(privKey)

	txInSig, err := txscript.RawTxInSignature(btcTx, 0, walletRecoverScript, txscript.SigHashAll, key)
	if err != nil {
		btcLog.Error("GetWalletRecoverySignature", "sign btc tx in error", err)
		return nil, nil, ErrGetBtcTxInSig
	}
	builder := txscript.NewScriptBuilder()
	builder.AddData(txInSig)
	if isRetrieve {
		builder.AddOp(txscript.OP_FALSE)
	} else {
		builder.AddOp(txscript.OP_TRUE)
	}

	unlockScript, err := builder.Script()

	if err != nil {
		btcLog.Error("GetWalletRecoverySignature", "build script err", err)
		return nil, nil, ErrBuildBtcScript
	}

	sig, err = newBtcScriptSig(walletRecoverScript, unlockScript, 0, utxoSequence)
	if err != nil {
		btcLog.Error("GetWalletRecoverySignature", "new btc script sig err", err)
		return nil, nil, ErrNewBtcScriptSig
	}

	return sig, Script2PubKey(walletRecoverScript), nil
}
