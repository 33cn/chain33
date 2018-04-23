package test

import (
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"testing"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm"
	"encoding/hex"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/core"
)

// 正常创建合约逻辑
func TestCreateContract1(t *testing.T) {
	deployCode, _ := hex.DecodeString("60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")
	execCode, _ := hex.DecodeString("6060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")

	privKey := getPrivKey()

	gas := uint64(210000)
	tx := createTx(privKey, deployCode, gas,0)
	ret,addr,leftGas,err,statedb := createContract(tx,0)

	test := NewTester(t)
	test.assertNil(err)

	test.assertEqualsB(ret,execCode)
	test.assertBigger(int(gas), int(leftGas))
	test.assertNotEqualsI(common.Address(addr) , common.EmptyAddress())

	// 检查返回数据是否正确
	test.assertEqualsV(statedb.GetLastSnapshot() , 0)

	kvset := statedb.GetChangedStatedData(statedb.GetLastSnapshot())
	data := kv2map(kvset)
	acc := statedb.GetAccount(addr)

	// 检查返回的数据是否正确，在合约创建过程中，改变的数据是固定的
	// 应该生成5个变更数据，分别是：代码、代码哈希、存储、存储哈希、nonce
	test.assertEqualsV(len(data), 5)

	// 分别检查具体内容
	item := data[string(acc.GetCodeKey())]
	test.assertNotNil(item)
	test.assertEqualsB(item, execCode)

	item = data[string(acc.GetCodeHashKey())]
	test.assertNotNil(item)
	test.assertEqualsB(item, common.BytesToHash(crypto.Sha256(execCode)).Bytes())

	item = data[string(acc.GetNonceKey())]
	test.assertNotNil(item)
	test.assertEqualsV(int(acc.GetNonce()), int(core.Byte2Int(item)))
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

	test := NewTester(t)

	// 创建时gas不足，应该返回空
	test.assertNilB(ret)

	test.assertEqualsE(err, vm.ErrOutOfGas)

	// gas不足时，应该是被扣光了
	test.assertEqualsV(int(leftGas), 0)

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

	test := NewTester(t)

	// 这是合约代码已经生成了，但是没有存储到StateDb
	test.assertEqualsB(ret,execCode)

	//合约gas不足
	test.assertEqualsE(err,vm.ErrCodeStoreOutOfGas)

	// 合约计算是否正确
	// Gas消耗了部署的61个，还应该剩下
	test.assertEqualsV(int(leftGas), int(gas-usedGas))
}

// Gas充足，但是合约代码超大 （通过修改合约代码大小限制）
func TestCreateContract4(t *testing.T) {
	deployCode, _ := hex.DecodeString("60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")
	execCode, _ := hex.DecodeString("6060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")


	privKey := getPrivKey()
	// 以上合约代码部署逻辑需要消耗61个Gas，存储代码需要消耗10600个Gas
	gas := uint64(210000)
	tx := createTx(privKey, deployCode, gas,0)
	ret,_,_,err,_ := createContract(tx,50)

	test := NewTester(t)

	// 合约代码正常返回
	test.assertNotNil(ret)
	test.assertEqualsB(ret,execCode)

	// 返回指定错误
	test.assertNotNil(err)
	test.assertEqualsS(err.Error(),"evm: max code size exceeded")

}



