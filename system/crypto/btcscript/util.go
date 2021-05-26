package btcscript

import (
	"errors"

	"github.com/33cn/chain33/common"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

// GetBtcLockScript 根据地址类型，生成锁定脚本
func GetBtcLockScript(scriptType int32, pk, script []byte) (btcutil.Address, []byte, error) {

	var btcAddr btcutil.Address
	var err error
	if scriptType == TyPay2PubKey {
		btcAddr, err = btcutil.NewAddressPubKeyHash(btcutil.Hash160(pk), btcParams)
	} else {
		btcAddr, err = btcutil.NewAddressScriptHash(script, btcParams)
	}

	if err != nil {
		return nil, nil, errors.New("GetBtcLockScript new btc addr err:" + err.Error())
	}

	lockScript, err := txscript.PayToAddrScript(btcAddr)
	if err != nil {
		return nil, nil, errors.New("GetBtcLockScript get pay to addr script err:" + err.Error())
	}
	return btcAddr, lockScript, nil
}

// GetBtcUnlockScript 生成比特币解锁脚本
func GetBtcUnlockScript(msg, pkScript, prevScript []byte, hashType txscript.SigHashType,
	params *chaincfg.Params, kdb txscript.KeyDB, sdb txscript.ScriptDB) ([]byte, error) {

	tx := getBindBtcTx(msg)
	sigScript, err := txscript.SignTxOutput(params, tx, 0,
		pkScript, hashType, kdb, sdb, prevScript)
	if err != nil {
		return nil, errors.New("GetBtcUnlockScript sign btc output err:" + err.Error())
	}
	return sigScript, nil
}

// CheckBtcScript check btc script signature
func CheckBtcScript(msg, lockScript, unlockScript []byte, flags txscript.ScriptFlags) error {

	tx := getBindBtcTx(msg)
	tx.TxIn[0].SignatureScript = unlockScript
	vm, err := txscript.NewEngine(lockScript, tx, 0, flags, nil, nil, 0)
	if err != nil {
		return errors.New("CheckBtcScript, new script engine err:" + err.Error())
	}

	err = vm.Execute()
	if err != nil {
		return errors.New("CheckBtcScript, execute engine err:" + err.Error())
	}
	return nil
}

// 比特币脚本签名依赖原生交易结构，这里构造一个带一个输入的伪交易
// HACK: 通过构造临时比特币交易，将第一个输入的chainHash设为签名数据的哈希，完成绑定关系
func getBindBtcTx(msg []byte) *wire.MsgTx {

	tx := &wire.MsgTx{TxIn: []*wire.TxIn{{}}}
	_ = tx.TxIn[0].PreviousOutPoint.Hash.SetBytes(common.Sha256(msg)[:chainhash.HashSize])
	return tx
}

type addressToKey struct {
	key        *btcec.PrivateKey
	compressed bool
}

func mkGetKey(keys map[string]addressToKey) txscript.KeyDB {
	if keys == nil {
		return txscript.KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey,
			bool, error) {
			return nil, false, errors.New("nope")
		})
	}
	return txscript.KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey,
		bool, error) {
		a2k, ok := keys[addr.EncodeAddress()]
		if !ok {
			return nil, false, errors.New("nope")
		}
		return a2k.key, a2k.compressed, nil
	})
}

func mkGetScript(scripts map[string][]byte) txscript.ScriptDB {
	if scripts == nil {
		return txscript.ScriptClosure(func(addr btcutil.Address) ([]byte, error) {
			return nil, errors.New("nope")
		})
	}
	return txscript.ScriptClosure(func(addr btcutil.Address) ([]byte, error) {
		script, ok := scripts[addr.EncodeAddress()]
		if !ok {
			return nil, errors.New("nope")
		}
		return script, nil
	})
}
