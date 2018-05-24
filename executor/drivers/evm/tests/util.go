package tests

import (
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	crypto2 "gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common/crypto"
	c "gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/state"
	"time"
	"encoding/hex"
	"gitlab.33.cn/chain33/chain33/wallet"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/runtime"
	"fmt"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/model"
)


func getPrivKey() crypto.PrivKey {
	c, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		return nil
	}
	key, err := c.GenKey()
	if err != nil {
		return nil
	}
	return key
}

func getAddr(privKey crypto.PrivKey) *account.Address {
	return account.PubKeyToAddress(privKey.PubKey().Bytes())
}

func createTx(privKey crypto.PrivKey, code []byte, fee uint64, amount uint64) types.Transaction {

	action := model.ContractAction{Amount:amount, Code:code}
	tx := types.Transaction{Execer: []byte("user.evm"), Payload: types.Encode(&action), Fee: int64(fee)}
	tx.Sign(types.SECP256K1, privKey)
	return tx
}

func addAccount(mdb *db.GoMemDB, acc1 *types.Account) {
	acc:=account.NewCoinsAccount()
	set := acc.GetKVSet(acc1)
	for i := 0; i < len(set); i++ {
		mdb.Set(set[i].GetKey(), set[i].Value)
	}
}

func addContractAccount(db *state.MemoryStateDB, mdb *db.GoMemDB, addr string, a AccountJson) {
	acc:=state.NewContractAccount(addr, db)
	code,err := hex.DecodeString(a.code)
	if err != nil {
		fmt.Println(err)
	}
	acc.SetCode(code)
	acc.SetNonce(uint64(a.nonce))
	for k,v := range a.storage {
		key,_ := hex.DecodeString(k)
		value,_ := hex.DecodeString(v)
		acc.SetState(common.BytesToHash(key), common.BytesToHash(value))
	}
	set := acc.GetDataKV()
	set = append(set, acc.GetStateKV()...)
	for i := 0; i < len(set); i++ {
		mdb.Set(set[i].GetKey(), set[i].Value)
	}
}

func buildStateDB(addr string, balance int64) *db.GoMemDB {
	// 替换statedb中的数据库，获取测试需要的数据
	mdb,_ := db.NewGoMemDB("test","",0)

	// 将调用者账户设置进去，并给予金额，方便发起合约调用
	ac := &types.Account{Addr:addr, Balance:balance}
	addAccount(mdb, ac)

	return mdb
}

func createContract(mdb *db.GoMemDB, tx types.Transaction, maxCodeSize int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error, statedb *state.MemoryStateDB) {
	inst := evm.NewEVMExecutor()

	msg,_ := inst.GetMessage(&tx)

	inst.SetEnv(10,0,"",uint64(10))
	statedb = inst.GetMStateDB()

	statedb.StateDB=mdb

	statedb.CoinsAccount = account.NewCoinsAccount()
	statedb.CoinsAccount.SetDB(statedb.StateDB)

	vmcfg := inst.GetVMConfig()

	context := inst.NewEVMContext(msg)

	// 创建EVM运行时对象
	env := runtime.NewEVM(context, statedb, *vmcfg)
	if(maxCodeSize !=0){
		env.SetMaxCodeSize(maxCodeSize)
	}

	addr := *crypto2.RandomContractAddress()
	ret,_,leftGas,err :=  env.Create(runtime.AccountRef(msg.From()), addr, msg.Data(), msg.GasLimit())

	return ret,addr,leftGas,err,statedb
}

// 合约调用（从DB中加载之前创建的合约）
func callContract(mdb db.KV, tx types.Transaction, contractAdd common.Address) (ret []byte, leftOverGas uint64, err error, statedb *state.MemoryStateDB) {

	inst := evm.NewEVMExecutor()

	msg,_ := inst.GetMessage(&tx)

	inst.SetEnv(10,0,common.EmptyAddress().NormalString(),uint64(10))

	statedb = inst.GetMStateDB()

	// 替换statedb中的数据库，获取测试需要的数据
	statedb.StateDB=mdb

	statedb.CoinsAccount = account.NewCoinsAccount()
	statedb.CoinsAccount.SetDB(statedb.StateDB)

	vmcfg := inst.GetVMConfig()


	context := inst.NewEVMContext(msg)

	// 创建EVM运行时对象
	env := runtime.NewEVM(context, statedb, *vmcfg)

	//ret,addr,leftGas,err :=  runtime.Create(vm.AccountRef(msg.From()), msg.Data(), msg.GasLimit(), msg.Value())

	ret,_,leftGas,err := env.Call(runtime.AccountRef(msg.From()),contractAdd, msg.Data(), msg.GasLimit(), msg.Value())

	return ret,leftGas,err,statedb
}

func kv2map(kvset []*types.KeyValue) map[string][]byte {
	data := make(map[string][]byte)
	for i:=0; i<len(kvset);i++  {
		data[string(kvset[i].Key)] = kvset[i].Value
	}
	return data
}

func procSignRawTx(wal *wallet.Wallet, unsigned *types.ReqSignRawTx, payload []byte) (ret string, err error) {
	var key crypto.PrivKey
	if unsigned.GetPrivkey() != "" {
		keyByte, err := c.FromHex(unsigned.GetPrivkey())
		if err != nil || len(keyByte) == 0 {
			return "", err
		}
		cr, err := crypto.New(types.GetSignatureTypeName(wallet.SignType))
		if err != nil {
			return "", err
		}
		key, err = cr.PrivKeyFromBytes(keyByte)
		if err != nil {
			return "", err
		}
	} else {
		return "", types.ErrNoPrivKeyOrAddr
	}
	var tx types.Transaction
	bytes, err := c.FromHex(unsigned.GetTxHex())
	if err != nil {
		return "", err
	}
	err = types.Decode(bytes, &tx)
	if err != nil {
		return "", err
	}
	expire, err := time.ParseDuration(unsigned.GetExpire())
	if err != nil {
		return "", err
	}
	tx.SetExpire(expire)
	// 因为反序列化存在问题，重新设置payload
	tx.Payload = payload

	tx.Sign(int32(wallet.SignType), key)

	txHex := types.Encode(&tx)
	signedTx := hex.EncodeToString(txHex)
	return signedTx, nil
}