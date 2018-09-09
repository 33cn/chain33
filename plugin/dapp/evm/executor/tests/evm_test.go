package tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/db"
	evm "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common/crypto"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/runtime"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/state"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestVM(t *testing.T) {

	basePath := "testdata/"

	// 生成测试用例
	genTestCase(basePath)

	t.Parallel()

	// 执行测试用例
	runTestCase(t, basePath)

	//清空测试用例
	defer clearTestCase(basePath)
}

func TestTmp(t *testing.T) {
	//addr := common.StringToAddress("19i4kLkSrAr4ssvk1pLwjkFAnoXeJgvGvj")
	//fmt.Println(hex.EncodeToString(addr.Bytes()))
	tt := types.Now().Unix()
	fmt.Println(time.Unix(tt, 0).String())
}

type CaseFilter struct{}

var testCaseFilter = &CaseFilter{}

// 满足过滤条件的用例将被执行
func (filter *CaseFilter) filter(num int) bool {
	return num >= 0
}

// 满足过滤条件的用例将被执行
func (filter *CaseFilter) filterCaseName(name string) bool {
	//return name == "selfdestruct"
	return name != ""
}

func runTestCase(t *testing.T, basePath string) {
	filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if path == basePath || !info.IsDir() {
			return nil
		}
		runDir(t, path)
		return nil
	})
}

func runDir(tt *testing.T, basePath string) {
	filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		baseName := info.Name()

		if baseName[:5] == "data_" || baseName[:4] == "tpl_" || filepath.Ext(path) != ".json" {
			return nil
		}

		raw, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Println(err.Error())
			tt.FailNow()
		}

		var data interface{}
		json.Unmarshal(raw, &data)

		cases := parseData(data.(map[string]interface{}))
		for _, c := range cases {
			// 每个测试用例，单独起子任务测试
			tt.Run(c.name, func(t *testing.T) {
				runCase(t, c, baseName)
			})
		}

		return nil
	})
}

func runCase(tt *testing.T, c VMCase, file string) {
	tt.Logf("runing test case:%s in file:%s", c.name, file)

	// 1 构建预置环境 pre
	inst := evm.NewEVMExecutor()
	inst.SetEnv(c.env.currentNumber, c.env.currentTimestamp, uint64(c.env.currentDifficulty))
	inst.CheckInit()
	statedb := inst.GetMStateDB()
	mdb := createStateDB(statedb, c)
	statedb.StateDB = mdb
	statedb.CoinsAccount = account.NewCoinsAccount()
	statedb.CoinsAccount.SetDB(statedb.StateDB)

	// 2 创建交易信息 create
	vmcfg := inst.GetVMConfig()
	msg := buildMsg(c)
	context := inst.NewEVMContext(msg)
	context.Coinbase = common.StringToAddress(c.env.currentCoinbase)

	// 3 调用执行逻辑 call
	env := runtime.NewEVM(context, statedb, *vmcfg)
	var (
		ret []byte
		//addr common.Address
		//leftGas uint64
		err error
	)

	if len(c.exec.address) > 0 {
		ret, _, _, err = env.Call(runtime.AccountRef(msg.From()), *common.StringToAddress(c.exec.address), msg.Data(), msg.GasLimit(), msg.Value())
	} else {
		addr := crypto.RandomContractAddress()
		ret, _, _, err = env.Create(runtime.AccountRef(msg.From()), *addr, msg.Data(), msg.GasLimit(), "testExecName", "")
	}

	if err != nil {
		// 合约执行出错的情况下，判断错误是否相同，如果相同，则返回，不判断post
		if len(c.err) > 0 && c.err == err.Error() {
			return
		} else {
			// 非意料情况下的出错，视为错误
			tt.Errorf("test case:%s, failed:%s", c.name, err)
			tt.Fail()
			return
		}
	}
	// 4 检查执行结果 post (注意，这里不检查Gas具体扣费数额，因为计费规则不一样，值检查执行结果是否正确)
	t := NewTester(tt)
	// 4.1 返回结果
	t.assertEqualsB(ret, getBin(c.out))

	// 4.2 账户余额以及数据
	for k, v := range c.post {
		addrStr := (*common.StringToAddress(k)).String()
		t.assertEqualsV(int(statedb.GetBalance(addrStr)), int(v.balance))

		t.assertEqualsB(statedb.GetCode(addrStr), getBin(v.code))

		for a, b := range v.storage {
			if len(a) < 1 || len(b) < 1 {
				continue
			}
			hashKey := common.BytesToHash(getBin(a))
			hashVal := common.BytesToHash(getBin(b))
			t.assertEqualsB(statedb.GetState(addrStr, hashKey).Bytes(), hashVal.Bytes())
		}
	}
}

// 使用预先设置的数据构建测试环境数据库
func createStateDB(msdb *state.MemoryStateDB, c VMCase) *db.GoMemDB {
	// 替换statedb中的数据库，获取测试需要的数据
	mdb, _ := db.NewGoMemDB("test", "", 0)
	// 构建预置的账户信息
	for k, v := range c.pre {
		// 写coins账户
		ac := &types.Account{Addr: c.exec.caller, Balance: v.balance}
		addAccount(mdb, k, ac)

		// 写合约账户
		addContractAccount(msdb, mdb, k, v, c.exec.caller)
	}

	// 清空MemoryStateDB中的日志
	msdb.ResetDatas()

	return mdb
}

// 使用测试输入信息构建交易
func buildMsg(c VMCase) *common.Message {
	code, _ := hex.DecodeString(c.exec.code)
	addr1 := common.StringToAddress(c.exec.caller)
	addr2 := common.StringToAddress(c.exec.address)
	gasLimit := uint64(210000000)
	gasPrice := c.exec.gasPrice
	return common.NewMessage(*addr1, addr2, int64(1), uint64(c.exec.value), gasLimit, uint32(gasPrice), code, "")
}
