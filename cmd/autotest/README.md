
# autotest
自动化系统回归测试工具，外部支持输入测试用例配置文件，
输出测试用例执行结果并记录详细执行日志。
内部代码支持用例扩展开发，继承并实现通用接口，即可自定义实现用例类型。

### 编译
通过chain33 makefile
```
$ make autotest
```

### 运行

#### **直接执行**
已启动chain33服务并需要自定义配置用例文件
```
$ ./autotest -f autotest.toml -l autotest.log
```

-f，-l分别指定配置文件和日志文件，
不指定默认为autotest.toml，autotest.log


#### **通过chain33 makefile**
chain33开发人员修改框架或dapp代码，验证测试
```

//启动单节点solo版本chain33，运行默认配置autotest，dapp用于指定需要跑的配置用例
$ make autotest dapp=coins
$ make autotest dapp="coins token"

//dapp=all，执行所有预配置用例
$ make autotest dapp=all
```
目前autotest支持的dapp有，coins token trade privacy，后续有待扩展


### 配置文件

配置文件为toml格式，用于指定具体的测试用例文件
```
# 指定内部调用chain33-cli程序文件
cliCmd = "./chain33-cli"

# 进行用例check时，主要根据交易hash查询回执，多次查询失败总超时，单位秒
checkTimeout = 60

# 测试用例配置文件，根据dapp分类
[[TestCaseFile]]
contract = "bty"    # coins合约
filename = "bty.toml" # 用例文件路径

[[TestCaseFile]]
contract = "token"  # token合约
filename = "token.toml"
```


### 用例文件
用例文件用于配置具体的测试用例,采用toml格式，dapp的autotest目录下预配置了跑ci的用例文件，如coins.toml:
```
[[TransferCase]]
id = "btyTrans1"
command = "send bty transfer -a 10 -t 1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
from = "12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
to = "1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt"
amount = "10"
checkItem = ["balance"]
repeat = 1

```


### 用例类型
#### 基本类型
**BaseCase**，所有用例的基类类型
```go
type BaseCase struct {
	ID        string   `toml:"id"`  //用例id
	Command   string   `toml:"command"` //执行的cli命令
	Dep       []string `toml:"dep,omitempty"`   //依赖的用例id数组
	CheckItem []string `toml:"checkItem,omitempty"` //回执中需要check项
	Repeat    int      `toml:"repeat,omitempty"`    //重复执行次数
	Fail      bool     'toml:"fail,omitempty"'  //错误标志，置true时表示用例本身为错误用例，默认不配置为false
}
```

**CheckItem**，默认会检查交易回执执行结果，并根据用例不同支持以下字段
* **balance**，交易双方余额校验
* **frozon**， 涉及冻结余额操作校验
* **utxo**， 隐私相关交易，校验utxo余额


#### Coins合约
* **TransferCase**，转账
* **WithdrawCase**，从合约取出

#### Token合约
* **TokePreCreateCase**，token预创建
* **TokenFinishCreateCase**，token完成创建

#### Trade合约
* **SellCase**，token出售
* **DependBuyCase**，token买入，需要在dep字段指定依赖的**SellCase**的id

#### Privacy合约
* **PubToPrivCase**，公对私转账
* **PrivToPubCase**，私对公转账
* **PrivToPrivCase**，私对私转账

#### ...

### 日志分析


### 扩展开发
分为以下几个步骤
> 注册dapp的AutoTest类型，以chain33/system/dapp/coins为例
增加autotest目录，并新建coins.go文件
```go
package autotest

//导入autotest开发依赖，主要是types包
import (
	"reflect"
	. "github.com/33cn/chain33/cmd/autotest/types"
)

//声明coins的AutoTest结构，其成员皆为coins将实现的用例类型
type coinsAutoTest struct {
	SimpleCaseArr   []SimpleCase   `toml:"SimpleCase,omitempty"`
	TransferCaseArr []TransferCase `toml:"TransferCase,omitempty"`
	WithdrawCaseArr []WithdrawCase `toml:"WithdrawCase,omitempty"`
}

//注册AutoTest类型
//coinsAutoest结构需要实现AutoTest接口，
//才能进行注册，该接口共有两个函数
func init() {

	RegisterAutoTest(coinsAutoTest{})

}

//返回dapp名字
func (config coinsAutoTest) GetName() string {

	return "coins"
}

//返回AutoTest的类型
func (config coinsAutoTest) GetTestConfigType() reflect.Type {

	return reflect.TypeOf(config)
}
```


> 实现用例的测试行为
```go
SendCommand(id string)    //实现用例执行命令的行为

//实现用例check回执行为，正常的交易类型继承即可，无须重写。特殊需要可以重写
CheckResult(CheckHandlerMap) (bool, bool)

//需要实现用例checkItem每一项的check行为，并用该接口返回FunctionMap
getCheckHandlerMap() interface{} 返回CheckHandlerMap类型

```
根据需要重写以上接口，灵活定义用例行为