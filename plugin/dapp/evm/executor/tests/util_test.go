package tests

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	evm "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	crypto2 "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common/crypto"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/runtime"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/state"
	evmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func getBin(data string) (ret []byte) {
	ret, err := hex.DecodeString(data)
	if err != nil {
		fmt.Println(err)
	}
	return
}

func parseData(data map[string]interface{}) (cases []VMCase) {
	for k, v := range data {
		ut := VMCase{name: k}
		parseVMCase(v, &ut)
		cases = append(cases, ut)
	}
	return
}

func parseVMCase(data interface{}, ut *VMCase) {
	m := data.(map[string]interface{})
	for k, v := range m {
		switch k {
		case "env":
			ut.env = parseEnv(v)
		case "exec":
			ut.exec = parseExec(v)
		case "pre":
			ut.pre = parseAccount(v)
		case "post":
			ut.post = parseAccount(v)
		case "gas":
			ut.gas = toint64(unpre(v.(string)))
		case "logs":
			ut.logs = unpre(v.(string))
		case "out":
			ut.out = unpre(v.(string))
		case "err":
			ut.err = v.(string)
			if ut.err == "{{.Err}}" {
				ut.err = ""
			}
		default:
			//fmt.Println(k, "is of a type I don't know how to handle")
		}
	}
}

func parseEnv(data interface{}) EnvJson {
	m := data.(map[string]interface{})
	ut := EnvJson{}
	for k, v := range m {
		switch k {
		case "currentCoinbase":
			ut.currentCoinbase = unpre(v.(string))
		case "currentDifficulty":
			ut.currentDifficulty = toint64(unpre(v.(string)))
		case "currentGasLimit":
			ut.currentGasLimit = toint64(unpre(v.(string)))
		case "currentNumber":
			ut.currentNumber = toint64(unpre(v.(string)))
		case "currentTimestamp":
			ut.currentTimestamp = toint64(unpre(v.(string)))
		default:
			fmt.Println(k, "is of a type I don't know how to handle")
		}
	}
	return ut
}

func parseExec(data interface{}) ExecJson {
	m := data.(map[string]interface{})
	ut := ExecJson{}
	for k, v := range m {
		switch k {
		case "address":
			ut.address = unpre(v.(string))
		case "caller":
			ut.caller = unpre(v.(string))
		case "code":
			ut.code = unpre(v.(string))
		case "data":
			ut.data = unpre(v.(string))
		case "gas":
			ut.gas = toint64(unpre(v.(string)))
		case "gasPrice":
			ut.gasPrice = toint64(unpre(v.(string)))
		case "origin":
			ut.origin = unpre(v.(string))
		case "value":
			ut.value = toint64(unpre(v.(string))) / 100000000
		default:
			fmt.Println(k, "is of a type I don't know how to handle")
		}
	}
	return ut
}

func parseAccount(data interface{}) map[string]AccountJson {
	ret := make(map[string]AccountJson)
	m := data.(map[string]interface{})
	for k, v := range m {
		ret[unpre(k)] = parseAccount2(v)
	}
	return ret
}

func parseAccount2(data interface{}) AccountJson {
	m := data.(map[string]interface{})
	ut := AccountJson{}
	for k, v := range m {
		switch k {
		case "balance":
			ut.balance = toint64(unpre(v.(string))) / 100000000
		case "code":
			ut.code = unpre(v.(string))
		case "nonce":
			ut.nonce = toint64(unpre(v.(string)))
		case "storage":
			ut.storage = parseStorage(v)
		default:
			fmt.Println(k, "is of a type I don't know how to handle")
		}
	}
	return ut
}

func parseStorage(data interface{}) map[string]string {
	ret := make(map[string]string)
	m := data.(map[string]interface{})
	for k, v := range m {
		ret[unpre(k)] = unpre(v.(string))
	}
	return ret
}

func toint64(data string) int64 {
	if len(data) == 0 {
		return 0
	}

	val, err := strconv.ParseInt(data, 16, 64)
	if err != nil {
		fmt.Println(err)
		return 0
	}
	return val
}

// 去掉十六进制字符串前面的0x
func unpre(data string) string {
	if len(data) > 1 && data[:2] == "0x" {
		return data[2:]
	}
	return data
}

func getPrivKey() crypto.PrivKey {
	c, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		return nil
	}
	key, err := c.GenKey()
	if err != nil {
		return nil
	}
	return key
}

func getAddr(privKey crypto.PrivKey) *address.Address {
	return address.PubKeyToAddress(privKey.PubKey().Bytes())
}

func createTx(privKey crypto.PrivKey, code []byte, fee uint64, amount uint64) types.Transaction {

	action := evmtypes.EVMContractAction{Amount: amount, Code: code}
	tx := types.Transaction{Execer: []byte("evm"), Payload: types.Encode(&action), Fee: int64(fee), To: address.ExecAddress("evm")}
	tx.Sign(types.SECP256K1, privKey)
	return tx
}

func addAccount(mdb *db.GoMemDB, execAddr string, acc1 *types.Account) {
	acc := account.NewCoinsAccount()
	var set []*types.KeyValue
	if len(execAddr) > 0 {
		set = acc.GetExecKVSet(execAddr, acc1)
	} else {
		set = acc.GetKVSet(acc1)
	}

	for i := 0; i < len(set); i++ {
		mdb.Set(set[i].GetKey(), set[i].Value)
	}
}

func addContractAccount(db *state.MemoryStateDB, mdb *db.GoMemDB, addr string, a AccountJson, creator string) {
	acc := state.NewContractAccount(addr, db)
	acc.SetCreator(creator)
	code, err := hex.DecodeString(a.code)
	if err != nil {
		fmt.Println(err)
	}
	acc.SetCode(code)
	acc.SetNonce(uint64(a.nonce))
	for k, v := range a.storage {
		key, _ := hex.DecodeString(k)
		value, _ := hex.DecodeString(v)
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
	mdb, _ := db.NewGoMemDB("test", "", 0)

	// 将调用者账户设置进去，并给予金额，方便发起合约调用
	ac := &types.Account{Addr: addr, Balance: balance}
	addAccount(mdb, "", ac)

	return mdb
}

func createContract(mdb *db.GoMemDB, tx types.Transaction, maxCodeSize int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error, statedb *state.MemoryStateDB) {
	inst := evm.NewEVMExecutor()
	inst.CheckInit()
	msg, _ := inst.GetMessage(&tx)

	inst.SetEnv(10, 0, uint64(10))
	statedb = inst.GetMStateDB()

	statedb.StateDB = mdb

	statedb.CoinsAccount = account.NewCoinsAccount()
	statedb.CoinsAccount.SetDB(statedb.StateDB)

	vmcfg := inst.GetVMConfig()

	context := inst.NewEVMContext(msg)

	// 创建EVM运行时对象
	env := runtime.NewEVM(context, statedb, *vmcfg)
	if maxCodeSize != 0 {
		env.SetMaxCodeSize(maxCodeSize)
	}

	addr := *crypto2.RandomContractAddress()
	ret, _, leftGas, err := env.Create(runtime.AccountRef(msg.From()), addr, msg.Data(), msg.GasLimit(), fmt.Sprintf("%s%s", evmtypes.EvmPrefix, common.BytesToHash(tx.Hash()).Hex()), "")

	return ret, addr, leftGas, err, statedb
}
