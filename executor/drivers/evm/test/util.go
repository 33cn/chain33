package test

import (
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/account"
	"encoding/binary"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/core"
	"time"
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
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b,amount)
	tx := types.Transaction{Execer: []byte("evm"), Payload: append(b, code...), Fee: int64(fee)}
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

func buildStateDB(addr string, balance int64) *db.GoMemDB {
	// 替换statedb中的数据库，获取测试需要的数据
	mdb,_ := db.NewGoMemDB("test","",0)

	// 将调用者账户设置进去，并给予金额，方便发起合约调用
	ac := &types.Account{Addr:addr, Balance:balance}
	addAccount(mdb, ac)

	return mdb
}

func createContract(mdb *db.GoMemDB, tx types.Transaction, maxCodeSize int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error, statedb *core.MemoryStateDB) {
	inst := evm.NewFakeEVM()

	msg := inst.GetMessage(&tx)

	config := inst.GetChainConfig()
	statedb = inst.GetStateDB()

	statedb.StateDB=mdb

	vmcfg := inst.GetVMConfig()

	// 获取当前区块的高度和时间
	height := int64(10)
	tm := time.Now().UnixNano() / int64(time.Millisecond)

	coinbase := common.EmptyAddress()
	difficulty := uint64(10)

	context := evm.NewEVMContext(msg, height, tm, coinbase, difficulty)

	// 创建EVM运行时对象
	runtime := vm.NewEVM(context, statedb, config, *vmcfg)
	if(maxCodeSize !=0){
		runtime.SetMaxCodeSize(maxCodeSize)
	}

	ret,addr,leftGas,err :=  runtime.Create(vm.AccountRef(msg.From()), msg.Data(), msg.GasLimit(), msg.Value())

	return ret,addr,leftGas,err,statedb
}

// 合约调用（从DB中加载之前创建的合约）
func callContract(mdb db.KV, tx types.Transaction, contractAdd common.Address, input []byte) (ret []byte, leftOverGas uint64, err error, statedb *core.MemoryStateDB) {

	inst := evm.NewFakeEVM()

	msg := inst.GetMessage(&tx)

	config := inst.GetChainConfig()
	statedb = inst.GetStateDB()

	// 替换statedb中的数据库，获取测试需要的数据
	statedb.StateDB=mdb

	vmcfg := inst.GetVMConfig()

	// 获取当前区块的高度和时间
	height := int64(10)
	tm := time.Now().UnixNano() / int64(time.Millisecond)

	coinbase := common.EmptyAddress()
	difficulty := uint64(10)

	context := evm.NewEVMContext(msg, height, tm, coinbase, difficulty)

	// 创建EVM运行时对象
	runtime := vm.NewEVM(context, statedb, config, *vmcfg)

	//ret,addr,leftGas,err :=  runtime.Create(vm.AccountRef(msg.From()), msg.Data(), msg.GasLimit(), msg.Value())

	ret,leftGas,err := runtime.Call(vm.AccountRef(msg.From()),contractAdd, msg.Data(), msg.GasLimit(), msg.Value())

	return ret,leftGas,err,statedb
}

func kv2map(kvset []*types.KeyValue) map[string][]byte {
	data := make(map[string][]byte)
	for i:=0; i<len(kvset);i++  {
		data[string(kvset[i].Key)] = kvset[i].Value
	}
	return data
}

