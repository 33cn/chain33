package evm

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
	"testing"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm"
	"math/big"
	time2 "time"
	"github.com/pkg/errors"
	"encoding/hex"
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

func createTx(privKey crypto.PrivKey, code []byte, fee int64) types.Transaction {
	tx := types.Transaction{Execer: []byte("evm"), Payload: code, Fee: fee}
	tx.Sign(types.SECP256K1, privKey)
	return tx
}

type MemDB struct {
	data map[string][]byte
}

func (m MemDB) Get(key []byte) ([]byte, error){
	v,ok := m.data[string(key)]
	if ok {
		return v,nil
	}
	return v, errors.New("no data")
}

func (m MemDB) Set(key []byte, value []byte) (err error){
	m.data[string(key)] = value
	return nil
}

func createContract(deployCode []byte, tx types.Transaction, maxCodeSize int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error, mdb MemDB) {
	inst := NewFakeEVM()

	msg := inst.GetMessage(&tx)

	config := inst.GetChainConfig()
	statedb := inst.GetStateDB()

	// 替换statedb中的数据库，获取测试需要的数据
	mdb = MemDB{}
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

	ret,addr,leftGas,err :=  runtime.Create(vm.AccountRef(msg.From()), tx.Payload, context.GasLimit, big.NewInt(0))

	return ret,addr,leftGas,err,mdb
}

// 正常创建合约逻辑
func TestCreateContract1(t *testing.T) {
	deployCode, _ := hex.DecodeString("60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")
	execCode, _ := hex.DecodeString("6060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")

	privKey := getPrivKey()

	gas := int64(210000)
	tx := createTx(privKey, deployCode, gas)
	ret,addr,leftGas,err,_ := createContract(deployCode, tx,0)

	if err != nil{
		//t.Error("create contract return error:%s",err)
		t.Fail()
	}else{
		if hex.EncodeToString(ret) != hex.EncodeToString(execCode) {
			//t.Fatal("create contract failed!")
			t.Fail()
		}
		if int64(leftGas) >= gas {
			//t.Fatal("leftGas is invalid:%s (%s)", leftGas, gas)
			t.Fail()
		}
		if common.Address(addr) == common.EmptyAddress() {
			//t.Fatal("created contract address(%s) is invalid", addr)
			t.Fail()
		}
		//for k,v := range statedb.data {
		//	//t.Logf("%s = %s", hex.EncodeToString([]byte(k)), hex.EncodeToString(v))
		//}
	}
}

// 创建合约gas不足
func TestCreateContract2(t *testing.T) {
	deployCode, _ := hex.DecodeString("60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")
	//execCode, _ := hex.DecodeString("6060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")

	privKey := getPrivKey()
	// 以上合约代码部署逻辑需要消耗61个Gas，存储代码需要消耗10600个Gas
	gas := int64(60)
	tx := createTx(privKey, deployCode, gas)
	ret,_,leftGas,err,_ := createContract(deployCode, tx,0)

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
	usedGas := int64(61)
	gas := int64(100)
	tx := createTx(privKey, deployCode, gas)
	ret,_,leftGas,err,_ := createContract(deployCode, tx,0)

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
	gas := int64(210000)
	tx := createTx(privKey, deployCode, gas)
	ret,_,_,err,_ := createContract(deployCode, tx,50)

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

