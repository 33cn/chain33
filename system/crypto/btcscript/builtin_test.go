package btcscript_test

import (
	"testing"

	"github.com/33cn/chain33/system/crypto/btcscript"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/require"
)

func Test_WalletRecoveryScript(t *testing.T) {

	_, controlKey := util.Genaddress()
	_, recoverKey := util.Genaddress()
	delayTime := int64(10) //10 block height

	pkScript, err := btcscript.NewWalletRecoveryScript(
		controlKey.PubKey().Bytes(), recoverKey.PubKey().Bytes(), delayTime)

	require.Nil(t, err)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx := util.CreateNoneTx(cfg, nil)
	signMsg := types.Encode(tx)

	// withdraw wallet balance with control address
	sig, pubKey, err := btcscript.GetWalletRecoverySignature(false, signMsg, controlKey.Bytes(),
		pkScript, delayTime)

	require.Nil(t, err)

	tx.Signature = &types.Signature{
		Ty:        btcscript.ID,
		Pubkey:    pubKey,
		Signature: sig,
	}
	require.True(t, tx.CheckSign(0))

	// withdraw wallet balance with recover address
	sig, pubKey, err = btcscript.GetWalletRecoverySignature(true, signMsg, recoverKey.Bytes(),
		pkScript, delayTime)

	require.Nil(t, err)

	tx.Signature = &types.Signature{
		Ty:        btcscript.ID,
		Pubkey:    pubKey,
		Signature: sig,
	}
	require.True(t, tx.CheckSign(10))

	// delay time not satisfied
	sig, pubKey, err = btcscript.GetWalletRecoverySignature(true, signMsg, recoverKey.Bytes(),
		pkScript, delayTime-1)

	require.Nil(t, err)
	tx.Signature = &types.Signature{
		Ty:        btcscript.ID,
		Pubkey:    pubKey,
		Signature: sig,
	}
	require.False(t, tx.CheckSign(10))

	// without delay time
	sig, pubKey, err = btcscript.GetWalletRecoverySignature(false, signMsg, recoverKey.Bytes(),
		pkScript, delayTime-1)

	require.Nil(t, err)
	tx.Signature = &types.Signature{
		Ty:        btcscript.ID,
		Pubkey:    pubKey,
		Signature: sig,
	}
	require.False(t, tx.CheckSign(0))
}

func Test_MultisigScript(t *testing.T) {
	_, priv1 := util.Genaddress()
	_, priv2 := util.Genaddress()

	_, err := btcscript.NewMultiSigScript(nil, 0)
	require.Equal(t, btcscript.ErrInvalidMultiSigRequiredNum, err)

	_, err = btcscript.NewMultiSigScript([][]byte{priv1.PubKey().Bytes()}, 2)
	require.Equal(t, btcscript.ErrInvalidMultiSigRequiredNum, err)

	_, err = btcscript.NewMultiSigScript([][]byte{priv1.PubKey().Bytes()}, 1)
	require.Nil(t, err)

	_, err = btcscript.NewMultiSigScript([][]byte{priv1.PubKey().Bytes(), priv2.PubKey().Bytes()}, 1)
	require.Nil(t, err)
}
