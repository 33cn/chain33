package gas

import (
	"math/big"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/model"
)

const (
	GasQuickStep   uint64 = 2
	GasFastestStep uint64 = 3
	GasFastStep    uint64 = 5
	GasMidStep     uint64 = 8
	GasSlowStep    uint64 = 10
	GasExtStep     uint64 = 20

	// 允许开辟的最大内存空间大小，超过此值时将会导致溢出
	MaxNewMemSize uint64 = 0xffffffffe0
)

// 返回真实花费的Gas
//  availableGas - base * 63 / 64.
func callGas(gasTable GasTable, availableGas, base uint64, callCost *big.Int) (uint64, error) {
	if availableGas == callCost.Uint64() {
		availableGas = availableGas - base
		gas := availableGas - availableGas/64

		// 如果传入的callCost超大，我们认为是因为分叉引起，依然返回计算出的gas
		if callCost.BitLen() > 64 || gas < callCost.Uint64() {
			return gas, nil
		}
	}

	// 如果Gas超大，则返回错误
	if callCost.BitLen() > 64 {
		return 0, model.ErrGasUintOverflow
	}

	return callCost.Uint64(), nil
}
