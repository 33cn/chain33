package tests

import (
	"encoding/hex"
	"fmt"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/model"
	"testing"
	"time"
)

// 正常创建合约逻辑
func TestCreateContract1(t *testing.T) {
	deployCode, _ := hex.DecodeString("608060405260358060116000396000f3006080604052600080fd00a165627a7a723058203f5c7a16b3fd4fb82c8b466dd5a3f43773e41cc9c0acb98f83640880a39a68080029")
	execCode, _ := hex.DecodeString("6080604052600080fd00a165627a7a723058203f5c7a16b3fd4fb82c8b466dd5a3f43773e41cc9c0acb98f83640880a39a68080029")

	privKey := getPrivKey()

	gas := uint64(210000)
	gasLimit := gas
	tx := createTx(privKey, deployCode, gas, 10000000)
	mdb := buildStateDB(getAddr(privKey).String(), 500000000)
	ret, addr, leftGas, err, statedb := createContract(mdb, tx, 0)

	test := NewTester(t)
	test.assertNil(err)

	test.assertEqualsB(ret, execCode)
	test.assertBigger(int(gasLimit), int(leftGas))
	test.assertNotEqualsI(common.Address(addr), common.EmptyAddress())

	// 检查返回数据是否正确
	test.assertEqualsV(statedb.GetLastSnapshot().GetId(), 0)
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

	test.assertEqualsE(err, model.ErrOutOfGas)

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
	gasLimit := gas
	tx := createTx(privKey, deployCode, gas, 0)
	mdb := buildStateDB(getAddr(privKey).String(), 100000000)
	ret, _, leftGas, err, _ := createContract(mdb, tx, 0)

	test := NewTester(t)

	// 这是合约代码已经生成了，但是没有存储到StateDb
	test.assertEqualsB(ret, execCode)

	//合约gas不足
	test.assertEqualsE(err, model.ErrCodeStoreOutOfGas)

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
	gasLimit := gas
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
//contract MyStore {
//    uint value;
//    constructor() public{
//        value=9999999;
//    }
//    function set(uint x) public {
//        value = x;
//    }
//
//    function get() public constant returns (uint){
//        return value;
//    }
//}

func decodeHex(data string) []byte {
	str := data
	if data[0:2] == "0x" {
		str = data[2:]
	}
	deployCode, _ := hex.DecodeString(str)
	return deployCode
}

//func TestCreateTx(t *testing.T) {
//	caller := "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
//	to := "1ApLV3FEZiCseuvn7fMM1GpVtpZyS1r1St"
//	code := "0x3bd5c209"
//	//code := "4e71d92d"  // claim
//	//code := "1b9265b8"  // pay
//	//code := "541aea0f00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000021"  // put
//	//code := "9507d39a0000000000000000000000000000000000000000000000000000000000000003"  // get
//	deployCode := decodeHex(code)
//	fee := int64(3000000)
//
//	action := types.EVMContractAction{Amount:0, Code:deployCode}
//	tx := &types.Transaction{Execer: []byte("user.evm"), Payload: types.Encode(&action), Fee: int64(fee), To:to}
//
//	var err error
//	tx.Fee, err = tx.GetRealFee(types.MinBalanceTransfer)
//	if err != nil {
//		t.Error(err)
//		t.Fail()
//	}
//	tx.Fee += types.MinBalanceTransfer
//	tx.Fee += fee
//	random := rand.New(rand.NewSource(time.Now().UnixNano()))
//	tx.Nonce = random.Int63()
//	//tx.Sign(int32(wallet.SignType), privKey)
//	txHex := types.Encode(tx)
//	rawTx := hex.EncodeToString(txHex)
//
//	unsignedTx := &types.ReqSignRawTx{
//		Addr:caller,
//		Privkey:"CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944",
//		TxHex:rawTx,
//		Expire:"2h",
//	}
//
//	wal := &wallet.Wallet{}
//	signedTx, err := procSignRawTx(wal, unsignedTx, tx.Payload)
//	if err != nil{
//		t.Error(err)
//		t.Fail()
//	}else{
//		t.Log("begin to call cli")
//		params := jsonrpc.RawParm{
//			Data: signedTx,
//		}
//		var res string
//		ctx := commands.NewRpcCtx("http://localhost:8801", "Chain33.SendTransaction", params, &res)
//		ctx.Run()
//
//		t.Log(signedTx)
//	}
//
//	fmt.Println(common.Bytes2Hex(crypto.Keccak256([]byte("litian"))))
//}

func TestTmp(t *testing.T) {

	xx := time.Now().UnixNano()
	yy := time.Now().UTC().UnixNano()

	fmt.Println(xx)
	fmt.Println(yy)
	fmt.Println(time.Unix(0, xx).String())
	fmt.Println(time.Unix(0, yy).String())
}
