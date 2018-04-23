package evm

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
	"testing"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm"
	time2 "time"
	"github.com/pkg/errors"
	"encoding/hex"
	"encoding/binary"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/core"
)

//func initEVM() *EVM {
//
//}

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
	binary.LittleEndian.PutUint64(b,amount)
	tx := types.Transaction{Execer: []byte("evm"), Payload: append(b, code...), Fee: int64(fee)}
	tx.Sign(types.SECP256K1, privKey)
	return tx
}

type MemDB struct {
	data map[string][]byte
}

func (m *MemDB) Get(key []byte) ([]byte, error){
	v,ok := m.data[string(key)]
	if ok {
		return v,nil
	}
	return v, errors.New("no data")
}

func (m *MemDB) Set(key []byte, value []byte) (err error){
	if m.data==nil{
		m.data = make(map[string][]byte)
	}
	m.data[string(key)] = value
	return nil
}

func addAccount(mdb *db.GoMemDB, acc1 *types.Account) {
	acc:=account.NewCoinsAccount()
	set := acc.GetKVSet(acc1)
	for i := 0; i < len(set); i++ {
		mdb.Set(set[i].GetKey(), set[i].Value)
	}
}


func createContract(tx types.Transaction, maxCodeSize int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error, statedb *core.MemoryStateDB) {
	inst := NewFakeEVM()

	msg := inst.GetMessage(&tx)

	config := inst.GetChainConfig()
	statedb = inst.GetStateDB()

	// 替换statedb中的数据库，获取测试需要的数据
	mdb,_ := db.NewGoMemDB("test","",0)

	// 将调用者账户设置进去，并给予金额，方便发起合约调用
	ac := &types.Account{Addr:msg.From().Str(), Balance:100000000}
	addAccount(mdb, ac)

	statedb.StateDB=mdb

	vmcfg := inst.GetVMConfig()

	// 获取当前区块的高度和时间
	height := int64(10)
	time := time2.Now().UnixNano() / int64(time2.Millisecond)

	coinbase := common.EmptyAddress()
	difficulty := uint64(10)

	context := NewEVMContext(msg, height, time, coinbase, difficulty)

	// 创建EVM运行时对象
	runtime := vm.NewEVM(context, statedb, config, *vmcfg)
	if(maxCodeSize !=0){
		runtime.SetMaxCodeSize(maxCodeSize)
	}

	ret,addr,leftGas,err :=  runtime.Create(vm.AccountRef(msg.From()), msg.Data(), msg.GasLimit(), msg.Value())

	return ret,addr,leftGas,err,statedb
}

func kv2map(kvset []*types.KeyValue) map[string][]byte {
	data := make(map[string][]byte)
	for i:=0; i<len(kvset);i++  {
		data[string(kvset[i].Key)] = kvset[i].Value
	}
	return data
}
// 正常创建合约逻辑
func TestCreateContract1(t *testing.T) {
	deployCode, _ := hex.DecodeString("60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")
	execCode, _ := hex.DecodeString("6060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")

	privKey := getPrivKey()

	gas := uint64(210000)
	tx := createTx(privKey, deployCode, gas,0)
	ret,addr,leftGas,err,statedb := createContract(tx,0)

	if err != nil{
		t.Errorf("create contract return error:%s",err)
		t.Fail()
	}else{
		if hex.EncodeToString(ret) != hex.EncodeToString(execCode) {
			t.Error("create contract failed!")
			t.Fail()
		}
		if leftGas >= gas {
			t.Errorf("leftGas is invalid:%s (%s)", leftGas, gas)
			t.Fail()
		}
		if common.Address(addr) == common.EmptyAddress() {
			t.Errorf("created contract address(%s) is invalid", addr)
			t.Fail()
		}
		// 检查返回数据是否正确
		if statedb.GetLastSnapshot() != 0 {
			t.Errorf("last snapshot version is not zero :%s", statedb.GetLastSnapshot())
			t.Fail()
		}

		kvset := statedb.GetChangedStatedData(statedb.GetLastSnapshot())
		data := kv2map(kvset)
		acc := statedb.GetAccount(addr)

		// 检查返回的数据是否正确，在合约创建过程中，改变的数据是固定的
		// 应该生成5个变更数据，分别是：代码、代码哈希、存储、存储哈希、nonce
		if len(data) != 5 {
			t.Errorf("create contract error, changed data size is not 5 : %s", len(data))
			t.Fail()
		}

		// 分别检查具体内容
		item := data[string(acc.GetCodeKey())]
		if item == nil{
			t.Errorf("create contract, changed data not contains code key:%s", acc.GetCodeKey())
			t.Fail()
		}else if string(item) != string(execCode){
			// 代码内容不同时也报错
			t.Errorf("create contract, chaged data contains error code!")
			t.Fail()
		}

		item = data[string(acc.GetCodeHashKey())]
		if item == nil{
			t.Errorf("create contract, changed data not contains code hash key:%s", acc.GetCodeHashKey())
			t.Fail()
		}else if string(item) != common.BytesToHash(crypto.Sha256(execCode)).Str(){
			// 代码内容不同时也报错
			t.Errorf("create contract, chaged data contains error code hash!")
			t.Fail()
		}

		item = data[string(acc.GetNonceKey())]
		if item == nil{
			t.Errorf("create contract, changed data not contains nonce key:%s", acc.GetCodeHashKey())
			t.Fail()
		}else if acc.GetNonce() != core.Byte2Int(item){
			// 代码内容不同时也报错
			t.Errorf("create contract, chaged data contains error nonce!")
			t.Fail()
		}

	}
}

// 创建合约gas不足
func TestCreateContract2(t *testing.T) {
	deployCode, _ := hex.DecodeString("60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")
	//execCode, _ := hex.DecodeString("6060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")

	privKey := getPrivKey()
	// 以上合约代码部署逻辑需要消耗61个Gas，存储代码需要消耗10600个Gas
	gas := uint64(60)
	tx := createTx(privKey, deployCode, gas,0)
	ret,_,leftGas,err,_ := createContract(tx,0)

	if ret != nil{
		//t.Fatal("create contract error!")
		t.Fail()
	}

	if err != vm.ErrOutOfGas{
		//t.Error("create contract return error:%s",err)
		t.Fail()
	}

	if int64(leftGas) != 0 {
		//t.Fatal("leftGas is invalid:%s (%s)", leftGas, gas)
		t.Fail()
	}
}

// 存储合约gas不足
func TestCreateContract3(t *testing.T) {
	deployCode, _ := hex.DecodeString("60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")
	execCode, _ := hex.DecodeString("6060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")

	privKey := getPrivKey()
	// 以上合约代码部署逻辑需要消耗61个Gas，存储代码需要消耗10600个Gas
	usedGas := uint64(61)
	gas := uint64(100)
	tx := createTx(privKey, deployCode, gas,0)
	ret,_,leftGas,err,_ := createContract(tx,0)

	// 这是合约代码已经生成了，但是没有存储到StateDb
	if hex.EncodeToString(ret) != hex.EncodeToString(execCode) {
		//t.Fatal("create contract failed!")
		t.Fail()
	}

	if err != vm.ErrCodeStoreOutOfGas{
		//t.Error("create contract return error:%s",err)
		t.Fail()
	}

	// Gas消耗了部署的61个，还应该剩下
	if int64(leftGas) != int64(gas-usedGas) {
		//t.Fatal("leftGas is invalid:%s (%s)", leftGas, gas)
		t.Fail()
	}
}

// Gas充足，但是合约代码超大 （通过修改合约代码大小限制）
func TestCreateContract4(t *testing.T) {
	deployCode, _ := hex.DecodeString("60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")
	//execCode, _ := hex.DecodeString("6060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")


	privKey := getPrivKey()
	// 以上合约代码部署逻辑需要消耗61个Gas，存储代码需要消耗10600个Gas
	gas := uint64(210000)
	tx := createTx(privKey, deployCode, gas,0)
	ret,_,_,err,_ := createContract(tx,50)

	// 合约代码正常返回
	if ret == nil{
		//t.Fatal("create contract error!")
		t.Fail()
	}

	// 返回指定错误
	if err==nil || err.Error() != errors.New("evm: max code size exceeded").Error(){
		//t.Error("create contract return error:%s",err)
		t.Fail()
	}
}



