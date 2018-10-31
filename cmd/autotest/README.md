
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
需要自定义配置用例文件
```
$ ./autotest -f autotest.toml -l autotest.log
```

-f，-l分别指定配置文件和日志文件，
不指定默认为autotest.toml，autotest.log


#### **通过chain33 makefile**
chain33开发人员修改框架或dapp代码，验证测试
```

//启动单节点solo版本chain33，运行默认配置autotest，dapp用于指定需要跑的配置用例
$ make autotest dapp=bty
$ make autotest dapp="bty token"

//dapp=all，执行所有预配置用例
$ make autotest dapp=all
```
目前autotest支持的dapp有，bty(coins） token trade privacy，后续有待扩展


### 配置文件

配置文件为toml格式，用于指定具体的测试用例文件
```
# 指定内部调用chain33-cli程序文件
cliCmd = "./chain33-cli"

# 进行用例执行结果check，等待回执时协程睡眠时间(秒/s)，大概配置为单个区块生成时间
checkSleepTime = 10

# 进行用例check时，主要根据交易hash查询回执，查询失败的超时次数
checkTimeout = 10

# 测试用例配置文件，根据dapp分类
[[TestCaseFile]]
contract = "bty"    # coins合约
filename = "bty.toml" # 用例文件路径

[[TestCaseFile]]
contract = "token"  # token合约
filename = "token.toml"
```


### 用例文件
用例文件用于配置具体的测试用例，autotest代码目录下预配置了基本用例文件，可以作为参考，如bty.toml:
```
[[TransferCase]]
id = "btyTrans1"
command = "send bty transfer -a 10 -t 1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt -k 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
from = "12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
to = "1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt"
amount = "10"
checkItem = ["balance"]
repeat = 1

[[TransferCase]]
id = "btyTrans2"
command = "send bty transfer -a 1 -t 17UZr5eJVxDRW1gs7rausQwaSUPtvcpxGT -k 1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt"
from = "1D9xKRnLvV2zMtSxSx33ow1GF4pcbLcNRt"
to = "17UZr5eJVxDRW1gs7rausQwaSUPtvcpxGT"
amount = "1"
checkItem = ["balance"]
repeat = 5
dep = ["btyTrans1"]

[[WithdrawCase]]
id = "failWithdraw"     #带有fail前缀表示用例本身是失败的，在autotest程序中集成了这个特性
command = "send bty withdraw -a 1.1 -e token -k 17UZr5eJVxDRW1gs7rausQwaSUPtvcpxGT"
addr = "17UZr5eJVxDRW1gs7rausQwaSUPtvcpxGT"
amount = "1.1"
checkItem = ["balance"]
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
	Repeat    int      `toml:"repeat,omitempty"`    //重复次数
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
可以根据测试需求，开发自定义测试用例，只需继承实现相关的接口
```go
doSendCommand(id string)    //实现用例执行命令的行为

//实现用例check回执行为，正常的交易类型继承即可，无须重写。特殊需要可以重写
doCheckResult(CheckHandlerMap) (bool, bool)

//需要实现用例checkItem每一项的check行为，并用该接口返回FunctionMap
getCheckHandlerMap() CheckHandlerMap

```
根据需要重写以上接口，灵活定义用例行为