package params

import (
	"math/big"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/state"
)

// 解释器中调用Gas计算时所传递的合约中和Gas相关的参数结构
// 之所以单独封装，是为了将解释器、指令集、Gas计算三者解耦
// 此结构包含的属性均为只读
type GasParam struct {
	// 此合约对象的可用Gas（合约执行过程中会修改此值）
	Gas uint64

	// 上下文中合约自身的地址
	// 注意，当合约通过CallCode调用时，这个地址并不是当前合约代码对应的地址，而是调用者的地址
	Address common.Address
}

// 解释器中调用Gas计算时所传递的合约中和EVM相关的参数结构
// 之所以单独封装，是为了将解释器、Gas计算两者解耦
// 此结构包含的属性中StateDB和CallGasTemp可写，解释器调用Gas计算的步骤如下：
// 1. 使用EVM构造EVMParam（属性除CallGasTemp外，全部传递引用）；
// 2. 以EVMParam为参数，调用Gas计算；
// 3. 计算结束后，使用EVMParam中的值回填到EVM中；
type EVMParam struct {

	// 状态数据操作入口
	StateDB state.StateDB

	// 此属性用于临时存储计算出来的Gas消耗值
	// 在指令执行时，会调用指令的gasCost方法，计算出此指令需要消耗的Gas，并存放在此临时属性中
	// 然后在执行opCall时，从此属性获取消耗的Gas值
	CallGasTemp uint64

	// NUMBER 指令，当前区块高度
	BlockNumber *big.Int
}
