package runtime

import (
	"math/big"
	"time"

	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/mm"
)


// Tracer接口用来在合约执行过程中收集跟踪数据。
// CaptureState 会在EVM解释每条指令时调用。
// 需要注意的是，传入的引用参数不允许修改，否则会影响EVM解释执行；如果需要使用其中的数据，请复制后使用。
type Tracer interface {
	CaptureStart(from common.Address, to common.Address, call bool, input []byte, gas uint64, value *big.Int) error
	CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *mm.Memory, stack *mm.Stack, contract *Contract, depth int, err error) error
	CaptureFault(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *mm.Memory, stack *mm.Stack, contract *Contract, depth int, err error) error
	CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) error
}
