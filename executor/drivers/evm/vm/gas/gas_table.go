package gas

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/params"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/mm"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/model"
)

type (
	GasFunc func(GasTable, *params.EVMParam, *params.GasParam, *mm.Stack, *mm.Memory, uint64) (uint64, error) // last parameter is the requested memory size as a uint64
)

// 此文件中定义各种指令操作花费的Gas计算
// Gas定价表结构
type GasTable struct {
	ExtcodeSize uint64
	ExtcodeCopy uint64
	Balance     uint64
	SLoad       uint64
	Calls       uint64
	Suicide     uint64

	ExpByte uint64

	// 奖励合约账户已经自杀的情况下，需要多少计费
	// 此属性可以设置为0，不设置，此时不计费
	CreateBySuicide uint64
}

var (
	// 定义各种操作的Gas定价
	GasTableHomestead = GasTable{
		ExtcodeSize: 20,
		ExtcodeCopy: 20,
		Balance:     20,
		SLoad:       50,
		Calls:       40,
		Suicide:     0,
		ExpByte:     10,
	}
)

// memoryGasCosts calculates the quadratic gas for memory expansion. It does so
// only for the memory region that is expanded, not the total memory.
func memoryGasCost(mem *mm.Memory, newMemSize uint64) (uint64, error) {

	if newMemSize == 0 {
		return 0, nil
	}
	// The maximum that will fit in a uint64 is max_word_count - 1
	// anything above that will result in an overflow.
	// Additionally, a newMemSize which results in a
	// newMemSizeWords larger than 0x7ffffffff will cause the square operation
	// to overflow.
	// The constant 0xffffffffe0 is the highest number that can be used without
	// overflowing the gas calculation
	if newMemSize > 0xffffffffe0 {
		return 0, model.ErrGasUintOverflow
	}

	newMemSizeWords := common.ToWordSize(newMemSize)
	newMemSize = newMemSizeWords * 32

	if newMemSize > uint64(mem.Len()) {
		square := newMemSizeWords * newMemSizeWords
		linCoef := newMemSizeWords * params.MemoryGas
		quadCoef := square / params.QuadCoeffDiv
		newTotalFee := linCoef + quadCoef

		fee := newTotalFee - mem.LastGasCost
		mem.LastGasCost = newTotalFee

		return fee, nil
	}
	return 0, nil
}

func ConstGasFunc(gas uint64) GasFunc {
	return func(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
		return gas, nil
	}
}

func GasCallDataCopy(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = common.SafeAdd(gas, GasFastestStep); overflow {
		return 0, model.ErrGasUintOverflow
	}

	words, overflow := common.BigUint64(stack.Back(2))
	if overflow {
		return 0, model.ErrGasUintOverflow
	}

	if words, overflow = common.SafeMul(common.ToWordSize(words), params.CopyGas); overflow {
		return 0, model.ErrGasUintOverflow
	}

	if gas, overflow = common.SafeAdd(gas, words); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasReturnDataCopy(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = common.SafeAdd(gas, GasFastestStep); overflow {
		return 0, model.ErrGasUintOverflow
	}

	words, overflow := common.BigUint64(stack.Back(2))
	if overflow {
		return 0, model.ErrGasUintOverflow
	}

	if words, overflow = common.SafeMul(common.ToWordSize(words), params.CopyGas); overflow {
		return 0, model.ErrGasUintOverflow
	}

	if gas, overflow = common.SafeAdd(gas, words); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasSStore(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	var (
		y, x = stack.Back(1), stack.Back(0)
		val  = evm.StateDB.GetState(contractGas.Address, common.BigToHash(x))
	)
	// This checks for 3 scenario's and calculates gas accordingly
	// 1. From a zero-value address to a non-zero value         (NEW VALUE)
	// 2. From a non-zero value address to a zero-value address (DELETE)
	// 3. From a non-zero to a non-zero                         (CHANGE)
	if common.EmptyHash(val) && !common.EmptyHash(common.BigToHash(y)) {
		// 0 => non 0
		return params.SstoreSetGas, nil
	} else if !common.EmptyHash(val) && common.EmptyHash(common.BigToHash(y)) {
		evm.StateDB.AddRefund(params.SstoreRefundGas)

		return params.SstoreClearGas, nil
	} else {
		// non 0 => non 0 (or 0 => 0)
		return params.SstoreResetGas, nil
	}
}

func MakeGasLog(n uint64) GasFunc {
	return func(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
		requestedSize, overflow := common.BigUint64(stack.Back(1))
		if overflow {
			return 0, model.ErrGasUintOverflow
		}

		gas, err := memoryGasCost(mem, memorySize)
		if err != nil {
			return 0, err
		}

		if gas, overflow = common.SafeAdd(gas, params.LogGas); overflow {
			return 0, model.ErrGasUintOverflow
		}
		if gas, overflow = common.SafeAdd(gas, n*params.LogTopicGas); overflow {
			return 0, model.ErrGasUintOverflow
		}

		var memorySizeGas uint64
		if memorySizeGas, overflow = common.SafeMul(requestedSize, params.LogDataGas); overflow {
			return 0, model.ErrGasUintOverflow
		}
		if gas, overflow = common.SafeAdd(gas, memorySizeGas); overflow {
			return 0, model.ErrGasUintOverflow
		}
		return gas, nil
	}
}

func GasSha3(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	if gas, overflow = common.SafeAdd(gas, params.Sha3Gas); overflow {
		return 0, model.ErrGasUintOverflow
	}

	wordGas, overflow := common.BigUint64(stack.Back(1))
	if overflow {
		return 0, model.ErrGasUintOverflow
	}
	if wordGas, overflow = common.SafeMul(common.ToWordSize(wordGas), params.Sha3WordGas); overflow {
		return 0, model.ErrGasUintOverflow
	}
	if gas, overflow = common.SafeAdd(gas, wordGas); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasCodeCopy(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = common.SafeAdd(gas, GasFastestStep); overflow {
		return 0, model.ErrGasUintOverflow
	}

	wordGas, overflow := common.BigUint64(stack.Back(2))
	if overflow {
		return 0, model.ErrGasUintOverflow
	}
	if wordGas, overflow = common.SafeMul(common.ToWordSize(wordGas), params.CopyGas); overflow {
		return 0, model.ErrGasUintOverflow
	}
	if gas, overflow = common.SafeAdd(gas, wordGas); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasExtCodeCopy(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}

	var overflow bool
	if gas, overflow = common.SafeAdd(gas, gt.ExtcodeCopy); overflow {
		return 0, model.ErrGasUintOverflow
	}

	wordGas, overflow := common.BigUint64(stack.Back(3))
	if overflow {
		return 0, model.ErrGasUintOverflow
	}

	if wordGas, overflow = common.SafeMul(common.ToWordSize(wordGas), params.CopyGas); overflow {
		return 0, model.ErrGasUintOverflow
	}

	if gas, overflow = common.SafeAdd(gas, wordGas); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasMLoad(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, model.ErrGasUintOverflow
	}
	if gas, overflow = common.SafeAdd(gas, GasFastestStep); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasMStore8(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, model.ErrGasUintOverflow
	}
	if gas, overflow = common.SafeAdd(gas, GasFastestStep); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasMStore(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, model.ErrGasUintOverflow
	}
	if gas, overflow = common.SafeAdd(gas, GasFastestStep); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasCreate(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	var overflow bool
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	if gas, overflow = common.SafeAdd(gas, params.CreateGas); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasBalance(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	return gt.Balance, nil
}

func GasExtCodeSize(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	return gt.ExtcodeSize, nil
}

func GasSLoad(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	return gt.SLoad, nil
}

func GasExp(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	expByteLen := uint64((stack.Items[stack.Len()-2].BitLen() + 7) / 8)

	var (
		gas      = expByteLen * gt.ExpByte // no overflow check required. Max is 256 * ExpByte gas
		overflow bool
	)
	if gas, overflow = common.SafeAdd(gas, GasSlowStep); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasCall(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	var (
		gas            = gt.Calls
		transfersValue = stack.Back(2).Sign() != 0
		address        = common.BigToAddress(stack.Back(1))
		eip158         = evm.ChainConfig.IsEIP158(evm.BlockNumber)
	)
	if eip158 {
		if transfersValue && evm.StateDB.Empty(address) {
			gas += params.CallNewAccountGas
		}
	} else if !evm.StateDB.Exist(address) {
		gas += params.CallNewAccountGas
	}
	if transfersValue {
		gas += params.CallValueTransferGas
	}
	memoryGas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = common.SafeAdd(gas, memoryGas); overflow {
		return 0, model.ErrGasUintOverflow
	}

	evm.CallGasTemp, err = callGas(gt, contractGas.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, err
	}
	if gas, overflow = common.SafeAdd(gas, evm.CallGasTemp); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasCallCode(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	gas := gt.Calls
	if stack.Back(2).Sign() != 0 {
		gas += params.CallValueTransferGas
	}
	memoryGas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = common.SafeAdd(gas, memoryGas); overflow {
		return 0, model.ErrGasUintOverflow
	}

	evm.CallGasTemp, err = callGas(gt, contractGas.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, err
	}
	if gas, overflow = common.SafeAdd(gas, evm.CallGasTemp); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasReturn(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	return memoryGasCost(mem, memorySize)
}

func GasRevert(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	return memoryGasCost(mem, memorySize)
}

func GasSuicide(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	var gas uint64
	// EIP150 homestead gas reprice fork:
	if evm.ChainConfig.IsEIP150(evm.BlockNumber) {
		gas = gt.Suicide
		var (
			address = common.BigToAddress(stack.Back(0))
			eip158  = evm.ChainConfig.IsEIP158(evm.BlockNumber)
		)

		if eip158 {
			// if empty and transfers value
			if evm.StateDB.Empty(address) && evm.StateDB.GetBalance(contractGas.Address).Sign() != 0 {
				gas += gt.CreateBySuicide
			}
		} else if !evm.StateDB.Exist(address) {
			gas += gt.CreateBySuicide
		}
	}

	if !evm.StateDB.HasSuicided(contractGas.Address) {
		evm.StateDB.AddRefund(params.SuicideRefundGas)
	}
	return gas, nil
}

func GasDelegateCall(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = common.SafeAdd(gas, gt.Calls); overflow {
		return 0, model.ErrGasUintOverflow
	}

	evm.CallGasTemp, err = callGas(gt, contractGas.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, err
	}
	if gas, overflow = common.SafeAdd(gas, evm.CallGasTemp); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasStaticCall(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = common.SafeAdd(gas, gt.Calls); overflow {
		return 0, model.ErrGasUintOverflow
	}

	evm.CallGasTemp, err = callGas(gt, contractGas.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, err
	}
	if gas, overflow = common.SafeAdd(gas, evm.CallGasTemp); overflow {
		return 0, model.ErrGasUintOverflow
	}
	return gas, nil
}

func GasPush(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	return GasFastestStep, nil
}

func GasSwap(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	return GasFastestStep, nil
}

func GasDup(gt GasTable, evm *params.EVMParam, contractGas *params.GasParam, stack *mm.Stack, mem *mm.Memory, memorySize uint64) (uint64, error) {
	return GasFastestStep, nil
}
