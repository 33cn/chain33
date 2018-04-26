package test

import (
	"encoding/hex"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"testing"
	"encoding/binary"
	"gitlab.33.cn/chain33/chain33/types"
	"math/rand"
	"time"
	"gitlab.33.cn/chain33/chain33/wallet"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm"
)

// 正常创建合约逻辑
func TestCreateContract1(t *testing.T) {
	deployCode, _ := hex.DecodeString("60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")
	execCode, _ := hex.DecodeString("6060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")

	privKey := getPrivKey()

	gas := uint64(210000)
	gasLimit := gas * evm.TX_GAS_TIMES_FEE
	tx := createTx(privKey, deployCode, gas, 0)
	mdb := buildStateDB(getAddr(privKey).String(), 100000000)
	ret, addr, leftGas, err, statedb := createContract(mdb, tx, 0)

	test := NewTester(t)
	test.assertNil(err)

	test.assertEqualsB(ret, execCode)
	test.assertBigger(int(gasLimit), int(leftGas))
	test.assertNotEqualsI(common.Address(addr), common.EmptyAddress())

	// 检查返回数据是否正确
	test.assertEqualsV(statedb.GetLastSnapshot(), 0)

	kvset,_ := statedb.GetChangedData(statedb.GetLastSnapshot())
	data := kv2map(kvset)
	//acc := statedb.GetAccount(addr)

	// 检查返回的数据是否正确，在合约创建过程中，改变的数据是固定的
	// 应该生成2个变更，分别是：数据（代码、代码哈希）、状态（存储、存储哈希、nonce、是否自杀）
	test.assertEqualsV(len(data), 2)

	// 分别检查具体内容
	//item := data[string(acc.GetCodeKey())]
	//test.assertNotNil(item)
	//test.assertEqualsB(item, execCode)
	//
	//item = data[string(acc.GetCodeHashKey())]
	//test.assertNotNil(item)
	//test.assertEqualsB(item, common.BytesToHash(crypto.Sha256(execCode)).Bytes())
	//
	//item = data[string(acc.GetNonceKey())]
	//test.assertNotNil(item)
	//test.assertEqualsV(int(acc.GetNonce()), int(core.Byte2Int(item)))
}

// 创建合约gas不足
func TestCreateContract2(t *testing.T) {
	deployCode, _ := hex.DecodeString("60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")
	//execCode, _ := hex.DecodeString("6060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")

	privKey := getPrivKey()
	// 以上合约代码部署逻辑需要消耗61个Gas，存储代码需要消耗10600个Gas
	gas := uint64(30)
	tx := createTx(privKey, deployCode, gas, 0)
	mdb := buildStateDB(getAddr(privKey).String(), 100000000)
	ret, _, leftGas, err, _ := createContract(mdb, tx, 0)

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
	gasLimit := gas * evm.TX_GAS_TIMES_FEE
	tx := createTx(privKey, deployCode, gas, 0)
	mdb := buildStateDB(getAddr(privKey).String(), 100000000)
	ret, _, leftGas, err, _ := createContract(mdb, tx, 0)

	test := NewTester(t)

	// 这是合约代码已经生成了，但是没有存储到StateDb
	test.assertEqualsB(ret, execCode)

	//合约gas不足
	test.assertEqualsE(err, vm.ErrCodeStoreOutOfGas)

	// 合约计算是否正确
	// Gas消耗了部署的61个，还应该剩下
	test.assertEqualsV(int(leftGas), int(gasLimit-usedGas))
}

// Gas充足，但是合约代码超大 （通过修改合约代码大小限制）
func TestCreateContract4(t *testing.T) {
	deployCode, _ := hex.DecodeString("60606040523415600e57600080fd5b603580601b6000396000f3006060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")
	execCode, _ := hex.DecodeString("6060604052600080fd00a165627a7a723058204bf1accefb2526a5077bcdfeaeb8020162814272245a9741cc2fddd89191af1c0029")

	privKey := getPrivKey()
	// 以上合约代码部署逻辑需要消耗61个Gas，存储代码需要消耗10600个Gas
	gas := uint64(210000)
	gasLimit := gas * evm.TX_GAS_TIMES_FEE
	tx := createTx(privKey, deployCode, gas, 0)
	mdb := buildStateDB(getAddr(privKey).String(), 100000000)
	ret, _, leftGas, err, _ := createContract(mdb, tx, 50)

	test := NewTester(t)

	// 合约代码正常返回
	test.assertNotNil(ret)
	test.assertEqualsB(ret, execCode)

	test.assertBigger(int(gasLimit), int(leftGas))

	// 返回指定错误
	test.assertNotNil(err)
	test.assertEqualsS(err.Error(), "evm: max code size exceeded")
}

// 下面测试合约调用时的合约代码
// 对应二进制：608060405234801561001057600080fd5b506298967f60008190555060df806100296000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a72305820b3ccec4d8cbe393844da31834b7464f23d3b81b24f36ce7e18bb09601f2eb8660029
// contract MyStore {
//     uint value;
//     constructor() public{
//         value=9999999;
//     }
//     function set(uint x) public {
//         value = x;
//     }
//
//     function get() public constant returns (uint){
//         return value;
//     }
// }

func TestCreateTx(t *testing.T) {
	caller := "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
	to := "1HgXQ9b2Y7GSL2e7E1HPBNi35EYv9HZoaR"
	code := "6d4ce63c"
	deployCode, _ := hex.DecodeString(code)
	fee := 50000
	amount := 2000000000

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b,uint64(amount))
	tx := &types.Transaction{Execer: []byte("evm"), Payload: append(b, deployCode...), Fee: int64(fee), To:to}

	var err error
	tx.Fee, err = tx.GetRealFee(types.MinBalanceTransfer)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	tx.Fee += types.MinBalanceTransfer
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()
	//tx.Sign(int32(wallet.SignType), privKey)
	txHex := types.Encode(tx)
	rawTx := hex.EncodeToString(txHex)

	unsignedTx := &types.ReqSignRawTx{
		Addr:caller,
		PrivKey:"CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944",
		TxHex:rawTx,
		Expire:"2h",
	}

	wal := &wallet.Wallet{}
	signedTx, err := procSignRawTx(wal, unsignedTx, tx.Payload)
	if err != nil{
		t.Error(err)
		t.Fail()
	}else{
		t.Log(signedTx)
	}
}

func TestCallContract1(t *testing.T) {
	code := "608060405234801561001057600080fd5b506298967f60008190555060df806100296000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a72305820b3ccec4d8cbe393844da31834b7464f23d3b81b24f36ce7e18bb09601f2eb8660029"
	deployCode, _ := hex.DecodeString(code)
	execCode, _ := hex.DecodeString(code[82:])

	privKey := getPrivKey()

	gas := uint64(210000)
	usedGas := 64707
	gasLimit := gas * evm.TX_GAS_TIMES_FEE
	tx := createTx(privKey, deployCode, gas, 0)
	mdb := buildStateDB(getAddr(privKey).String(), 100000000)
	ret, addr, leftGas, err, statedb := createContract(mdb, tx, 0)

	test := NewTester(t)
	test.assertNil(err)

	test.assertEqualsB(ret, execCode)

	test.assertBigger(int(gasLimit), int(leftGas))
	test.assertEqualsV(usedGas, int(gasLimit-leftGas))
	test.assertNotEqualsI(common.Address(addr), common.EmptyAddress())

	// 检查返回数据是否正确
	test.assertEqualsV(statedb.GetLastSnapshot(), 0)

	// 将创建合约得出来的变更数据写入statedb，给调用合约时使用
	kvset, _ := statedb.GetChangedData(statedb.GetLastSnapshot())
	for i := 0; i < len(kvset); i++ {
		mdb.Set(kvset[i].Key, kvset[i].Value)
	}

	// 合约创建完成后，开始调用测试
	// 首先调用get方法，检查初始值设置是否正确
	gas = uint64(210000)
	gasLimit = gas * evm.TX_GAS_TIMES_FEE
	params := "6d4ce63c"
	callCode, _ := hex.DecodeString(params)
	tx = createTx(privKey, callCode, gas, 0)
	ret, leftGas, err, statedb = callContract(mdb, tx, addr)

	test.assertNil(err)
	test.assertBigger(int(gasLimit), int(leftGas))
	test.assertNotNil(ret)
	test.assertEqualsV(int(9999999), int(binary.BigEndian.Uint64(ret[24:])))

	// 调用合约的set(11)
	gas = uint64(210000)
	gasLimit = gas * evm.TX_GAS_TIMES_FEE
	params = "60fe47b1000000000000000000000000000000000000000000000000000000000000000b"
	callCode, _ = hex.DecodeString(params)
	tx = createTx(privKey, callCode, gas, 0)

	ret, leftGas, err, statedb = callContract(mdb, tx, addr)

	test.assertNil(err)
	test.assertBigger(int(gasLimit), int(leftGas))
	test.assertNilB(ret)

	// 再复制一次数据，确保set引起的变更被写入
	kvset, _ = statedb.GetChangedData(statedb.GetLastSnapshot())
	for i := 0; i < len(kvset); i++ {
		mdb.Set(kvset[i].Key, kvset[i].Value)
	}

	// 再调用get方法，检查设置的值是否生效
	gas = uint64(210000)
	gasLimit = gas * evm.TX_GAS_TIMES_FEE
	params = "6d4ce63c"
	callCode, _ = hex.DecodeString(params)
	tx = createTx(privKey, callCode, gas, 0)
	ret, leftGas, err, statedb = callContract(mdb, tx, addr)

	println(hex.EncodeToString(ret))
	test.assertNil(err)
	test.assertBigger(int(gasLimit), int(leftGas))
	test.assertNotNil(ret)
	test.assertEqualsV(int(11), int(binary.BigEndian.Uint64(ret[24:])))
}
