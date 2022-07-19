package script_test

import (
	"encoding/hex"
	"testing"

	"github.com/33cn/chain33/client/mocks"
	cryptocli "github.com/33cn/chain33/common/crypto/client"
	nty "github.com/33cn/chain33/system/dapp/none/types"
	"github.com/stretchr/testify/mock"

	"github.com/33cn/chain33/system/crypto/btcscript"

	"github.com/33cn/chain33/system/crypto/btcscript/script"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/require"
)

func Test_WalletRecoveryScript(t *testing.T) {

	_, controlKey := util.Genaddress()
	_, recoverKey1 := util.Genaddress()
	_, recoverKey2 := util.Genaddress()
	_, invalidKey := util.Genaddress()
	delayTime := int64(10) //10 s

	pkScript, err := script.NewWalletRecoveryScript(
		controlKey.PubKey().Bytes(),
		[][]byte{recoverKey1.PubKey().Bytes(), recoverKey2.PubKey().Bytes()},
		delayTime)

	require.Nil(t, err)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx := util.CreateNoneTx(cfg, nil)
	signMsg := types.Encode(tx)

	signAndCheck := func(isRetrieve bool, priv []byte, delay int64, expectResult bool) {
		t.Logf("privKey:%s", hex.EncodeToString(priv))
		sig, pubKey, err := script.GetWalletRecoverySignature(isRetrieve, signMsg, priv, pkScript, delay)
		require.Nil(t, err)
		tx.Signature = &types.Signature{
			Ty:        btcscript.ID,
			Pubkey:    pubKey,
			Signature: sig,
		}
		require.Equal(t, expectResult, tx.CheckSign(-1))
	}

	// withdraw wallet balance with control address
	signAndCheck(false, controlKey.Bytes(), 0, true)

	api := new(mocks.QueueProtocolAPI)
	cryptocli.SetQueueAPI(api)
	cryptocli.SetCurrentBlock(0, delayTime+1)
	runFn := func(args mock.Arguments) {
		execer := args.Get(0).(string)
		require.Equal(t, nty.NoneX, execer)
		funcName := args.Get(1).(string)
		require.Equal(t, nty.QueryGetDelayTxInfo, funcName)
		param := args.Get(2).(*types.ReqBytes)
		require.Equal(t, tx.Hash(), param.Data)
	}
	api.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(&nty.CommitDelayTxLog{DelayBeginTimestamp: 1}, nil).Run(runFn)

	// withdraw wallet balance with recover address
	signAndCheck(true, recoverKey1.Bytes(), delayTime, true)
	signAndCheck(true, recoverKey2.Bytes(), delayTime, true)
	signAndCheck(true, recoverKey1.Bytes(), delayTime-1, false)
	signAndCheck(false, recoverKey1.Bytes(), 0, false)
	signAndCheck(false, recoverKey2.Bytes(), 0, false)

	// invalid key
	signAndCheck(false, invalidKey.Bytes(), 0, false)
	signAndCheck(true, invalidKey.Bytes(), delayTime, false)
}

func Test_MultisigScript(t *testing.T) {
	_, priv1 := util.Genaddress()
	_, priv2 := util.Genaddress()

	_, err := script.NewMultiSigScript(nil, 0)
	require.Equal(t, script.ErrInvalidMultiSigRequiredNum, err)

	_, err = script.NewMultiSigScript([][]byte{priv1.PubKey().Bytes()}, 2)
	require.Equal(t, script.ErrInvalidMultiSigRequiredNum, err)

	_, err = script.NewMultiSigScript([][]byte{priv1.PubKey().Bytes()}, 1)
	require.Nil(t, err)

	_, err = script.NewMultiSigScript([][]byte{priv1.PubKey().Bytes(), priv2.PubKey().Bytes()}, 1)
	require.Nil(t, err)
}
