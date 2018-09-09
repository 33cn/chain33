package runtime

import (
	"math/big"
	"time"

	"encoding/json"
	"io"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/mm"
)

// Tracer接口用来在合约执行过程中收集跟踪数据。
// CaptureState 会在EVM解释每条指令时调用。
// 需要注意的是，传入的引用参数不允许修改，否则会影响EVM解释执行；如果需要使用其中的数据，请复制后使用。
type Tracer interface {
	CaptureStart(from common.Address, to common.Address, call bool, input []byte, gas uint64, value uint64) error
	CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *mm.Memory, stack *mm.Stack, contract *Contract, depth int, err error) error
	CaptureFault(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *mm.Memory, stack *mm.Stack, contract *Contract, depth int, err error) error
	CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) error
}

// 使用json格式打印日志
type JSONLogger struct {
	encoder *json.Encoder
}

// 指令执行状态信息
type StructLog struct {
	Pc         uint64                      `json:"pc"`
	Op         string                      `json:"op"`
	Gas        uint64                      `json:"gas"`
	GasCost    uint64                      `json:"gasCost"`
	Memory     []string                    `json:"memory"`
	MemorySize int                         `json:"memSize"`
	Stack      []string                    `json:"stack"`
	Storage    map[common.Hash]common.Hash `json:"-"`
	Depth      int                         `json:"depth"`
	Err        error                       `json:"-"`
}

func NewJSONLogger(writer io.Writer) *JSONLogger {
	return &JSONLogger{json.NewEncoder(writer)}
}

func (logger *JSONLogger) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value uint64) error {
	return nil
}

// CaptureState outputs state information on the logger.
func (logger *JSONLogger) CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *mm.Memory, stack *mm.Stack, contract *Contract, depth int, err error) error {
	log := StructLog{
		Pc:         pc,
		Op:         op.String(),
		Gas:        gas,
		GasCost:    cost,
		MemorySize: memory.Len(),
		Storage:    nil,
		Depth:      depth,
		Err:        err,
	}
	log.Memory = formatMemory(memory.Data())
	log.Stack = formatStack(stack.Data())
	return logger.encoder.Encode(log)
}

func formatStack(data []*big.Int) (res []string) {
	for _, v := range data {
		res = append(res, v.Text(16))
		v.String()
	}
	return
}

func formatMemory(data []byte) (res []string) {
	for idx := 0; idx < len(data); idx += 32 {
		res = append(res, common.Bytes2HexTrim(data[idx:idx+32]))
	}
	return
}

// CaptureFault outputs state information on the logger.
func (logger *JSONLogger) CaptureFault(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *mm.Memory, stack *mm.Stack, contract *Contract, depth int, err error) error {
	return nil
}

// CaptureEnd is triggered at end of execution.
func (logger *JSONLogger) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) error {
	type endLog struct {
		Output  string        `json:"output"`
		GasUsed int64         `json:"gasUsed"`
		Time    time.Duration `json:"time"`
		Err     string        `json:"error,omitempty"`
	}

	if err != nil {
		return logger.encoder.Encode(endLog{common.Bytes2Hex(output), int64(gasUsed), t, err.Error()})
	}
	return logger.encoder.Encode(endLog{common.Bytes2Hex(output), int64(gasUsed), t, ""})
}
