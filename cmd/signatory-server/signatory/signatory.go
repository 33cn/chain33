package signatory

import (
	"math/rand"
	"time"
	"encoding/hex"

	l "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
)

var log      = l.New("module", "signatory")

type Signatory struct {
	Privkey string
}

func (*Signatory) Echo(in *string, out *interface{}) error {
	if in == nil {
		return types.ErrInputPara
	}
	*out = *in
	return nil
}

type TokenFinish struct {
	OwnerAddr string `json:"owner_addr"`
	Symbol    string `json:"symbol"`
	//	Fee       int64  `json:"fee"`
}

func (signatory *Signatory) SignApprove(in *TokenFinish, out *interface{}) error {
	if in == nil {
		return types.ErrInputPara
	}
	if len(in.OwnerAddr) == 0 || len(in.Symbol) == 0 {
		return types.ErrInputPara
	}
	v := &types.TokenFinishCreate{Symbol: in.Symbol, Owner: in.OwnerAddr}
	finish := &types.TokenAction{
		Ty:    types.TokenActionFinishCreate,
		Value: &types.TokenAction_Tokenfinishcreate{v},
	}

	tx := &types.Transaction{
		Execer:  []byte("token"),
		Payload: types.Encode(finish),
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("token").String(),
	}

	var err error
	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		log.Error("SignApprove", "calc fee failed", err)
		return err
	}
	err = signTx(tx, signatory.Privkey)
	if err != nil {
		return err
	}
	txHex := types.Encode(tx)
	*out = hex.EncodeToString(txHex)
	return nil
}

func (signatory *Signatory) SignTranfer(in *string, out *interface{}) error {
	if in == nil {
		return types.ErrInputPara
	}
	if len(*in) == 0 {
		return types.ErrInputPara
	}

	amount := 1 * types.Coin
	v := &types.CoinsTransfer{
		Amount: amount,
		Note: "transfer 1 bty by signatory-server",
	}
	transfer := &types.CoinsAction{
		Ty: types.CoinsActionTransfer,
		Value: &types.CoinsAction_Transfer{v},
	}

	tx := &types.Transaction{
		Execer:  []byte("coins"),
		Payload: types.Encode(transfer),
		To:      *in,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
	}

	var err error
	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		log.Error("SignTranfer", "calc fee failed", err)
		return err
	}
	err = signTx(tx, signatory.Privkey)
	if err != nil {
		log.Error("SignTranfer", "signTx failed", err)
		return err
	}
	txHex := types.Encode(tx)
	*out = hex.EncodeToString(txHex)
	return nil

}

func signTx(tx *types.Transaction, hexPrivKey string) error {
	signType := types.SECP256K1
	c, err := crypto.New(types.GetSignatureTypeName(signType))

	bytes, err := common.FromHex(hexPrivKey)
	if err != nil {
		log.Error("signTx", "err", err)
		return err
	}

	privKey, err := c.PrivKeyFromBytes(bytes)
	if err != nil {
		log.Error("signTx", "err", err)
		return err
	}

	tx.Sign(int32(signType), privKey)
	return nil
}

