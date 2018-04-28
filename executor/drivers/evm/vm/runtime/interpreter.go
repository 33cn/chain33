// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package runtime

import (
	"fmt"
	"sync/atomic"

	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/gas"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/codes"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/mm"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/model"
)

// Config are the configuration options for the Interpreter
type Config struct {
	// Debug enabled debugging Interpreter options
	Debug bool
	// EnableJit enabled the JIT VM
	EnableJit bool
	// ForceJit forces the JIT VM
	ForceJit bool
	// Tracer is the op code logger
	Tracer Tracer
	// NoRecursion disabled Interpreter codes.CALL, codes.CALLcode,
	// delegate codes.CALL and create.
	NoRecursion bool
	// Enable recording of SHA3/keccak preimages
	EnablePreimageRecording bool
	// JumpTable contains the EVM instruction table. This
	// may be left uninitialised and will be set to the default
	// table.
	JumpTable [256]codes.Operation
}

// Interpreter is used to run Ethereum based contracts and will utilise the
// passed evmironment to query external sources for state information.
// The Interpreter will run the byte code VM or JIT VM based on the passed
// configuration.
type Interpreter struct {
	evm      *EVM
	cfg      Config
	gasTable gas.GasTable
	IntPool  *IntPool

	readOnly   bool   // Whether to throw on stateful modifications
	ReturnData []byte // Last codes.CALL's return data for subsequent reuse
}

// NewInterpreter returns a new instance of the Interpreter.
func NewInterpreter(evm *EVM, cfg Config) *Interpreter {
	// 使用是否包含第一个STOP指令判断jump table是否完成初始化
	// 需要注意，后继如果新增指令，需要在这里判断硬分叉，指定不同的指令集
	if !cfg.JumpTable[codes.STOP].Valid {
		cfg.JumpTable = codes.ConstantinopleInstructionSet
	}

	return &Interpreter{
		evm:      evm,
		cfg:      cfg,
		gasTable: evm.ChainConfig().GasTable(evm.BlockNumber),
		IntPool:  NewIntPool(),
	}
}

func (in *Interpreter) enforceRestrictions(op codes.OpCode, operation codes.Operation, stack *mm.Stack) error {
	if in.evm.chainRules.IsByzantium {
		if in.readOnly {
			// If the Interpreter is operating in readonly mode, make sure no
			// state-modifying operation is performed. The 3rd stack item
			// for a codes.CALL operation is the value. Transferring value from one
			// account to the others means the state is modified and should also
			// return with an error.
			if operation.Writes || (op == codes.CALL && stack.Back(2).BitLen() > 0) {
				return model.ErrWriteProtection
			}
		}
	}
	return nil
}

// Run loops and evaluates the contract's code with the given input data and returns
// the return byte-slice and an error if one occurred.
//
// It's important to note that any errors returned by the Interpreter should be
// considered a revert-and-consume-all-gas operation except for
// errExecutionReverted which means revert-and-keep-gas-left.
func (in *Interpreter) Run(contract *codes.Contract, input []byte) (ret []byte, err error) {
	// Increment the codes.CALL depth which is restricted to 1024
	in.evm.depth++
	defer func() { in.evm.depth-- }()

	// Reset the previous codes.CALL's return data. It's unimportant to preserve the old buffer
	// as every returning codes.CALL will return new data anyway.
	in.ReturnData = nil

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	var (
		op    codes.OpCode        // current codes.OpCode
		mem   = mm.NewMemory() // bound memory
		stack = mm.NewStack()  // local stack
		// For optimisation reason we're using uint64 as the program counter.
		// It's theoreticodes.CALLy possible to go above 2^64. The YP defines the PC
		// to be uint256. Practicodes.CALLy much less so feasible.
		pc   = uint64(0) // program counter
		cost uint64
		// copies used by tracer
		pcCopy  uint64 // needed for the deferred Tracer
		gasCopy uint64 // for Tracer to log gas remaining before execution
		logged  bool   // deferred Tracer should ignore already logged steps
	)
	contract.Input = input

	if in.cfg.Debug {
		defer func() {
			if err != nil {
				if !logged {
					in.cfg.Tracer.CaptureState(in.evm, pcCopy, op, gasCopy, cost, mem, stack, contract, in.evm.depth, err)
				} else {
					in.cfg.Tracer.CaptureFault(in.evm, pcCopy, op, gasCopy, cost, mem, stack, contract, in.evm.depth, err)
				}
			}
		}()
	}
	// The Interpreter main run loop (contextual). This loop runs until either an
	// explicit STOP, RETURN or SELFDESTRUCT is executed, an error occurred during
	// the execution of one of the operations or until the done flag is set by the
	// parent context.
	for atomic.LoadInt32(&in.evm.abort) == 0 {
		if in.cfg.Debug {
			// Capture pre-execution values for tracing.
			logged, pcCopy, gasCopy = false, pc, contract.Gas
		}

		// Get the operation from the jump table and validate the stack to ensure there are
		// enough stack items available to perform the operation.
		op = contract.GetOp(pc)
		operation := in.cfg.JumpTable[op]
		if !operation.Valid {
			return nil, fmt.Errorf("invalid codes.OpCode 0x%x", int(op))
		}
		if err := operation.ValidateStack(stack); err != nil {
			return nil, err
		}
		// If the operation is valid, enforce and write restrictions
		if err := in.enforceRestrictions(op, operation, stack); err != nil {
			return nil, err
		}

		var memorySize uint64
		// calculate the new memory size and expand the memory to fit
		// the operation
		if operation.MemorySize != nil {
			memSize, overflow := common.BigUint64(operation.MemorySize(stack))
			if overflow {
				return nil, model.ErrGasUintOverflow
			}
			// memory is expanded in words of 32 bytes. Gas
			// is also calculated in words.
			if memorySize, overflow = common.SafeMul(common.ToWordSize(memSize), 32); overflow {
				return nil, model.ErrGasUintOverflow
			}
		}
		// consume the gas and return an error if not enough gas is available.
		// cost is explicitly set so that the capture state defer method cas get the proper cost
		cost, err = operation.GasCost(in.gasTable, in.evm, contract, stack, mem, memorySize)
		if err != nil || !contract.UseGas(cost) {
			return nil, model.ErrOutOfGas
		}
		if memorySize > 0 {
			mem.Resize(memorySize)
		}

		if in.cfg.Debug {
			in.cfg.Tracer.CaptureState(in.evm, pc, op, gasCopy, cost, mem, stack, contract, in.evm.depth, err)
			logged = true
		}

		// execute the operation
		res, err := operation.Execute(&pc, in.evm, contract, mem, stack)
		// verifyPool is a build flag. Pool verification makes sure the integrity
		// of the integer pool by comparing values to a default value.
		if verifyPool {
			verifyIntegerPool(in.IntPool)
		}
		// if the operation clears the return data (e.g. it has returning data)
		// set the last return to the result of the operation.
		if operation.Returns {
			in.ReturnData = res
		}

		switch {
		case err != nil:
			return nil, err
		case operation.Reverts:
			return res, model.ErrExecutionReverted
		case operation.Halts:
			return res, nil
		case !operation.Jumps:
			pc++
		}
	}
	return nil, nil
}
