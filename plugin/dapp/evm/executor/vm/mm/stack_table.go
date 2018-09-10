package mm

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/params"
)

type (
	// 校验栈中数据是否满足计算要求
	StackValidationFunc func(*Stack) error
)

// 栈校验的通用逻辑封装（主要就是检查栈的深度和空间是否够用）
func MakeStackFunc(pop, push int) StackValidationFunc {
	return func(stack *Stack) error {
		if err := stack.Require(pop); err != nil {
			return err
		}

		if stack.Len()+push-pop > int(params.StackLimit) {
			return fmt.Errorf("stack limit reached %d (%d)", stack.Len(), params.StackLimit)
		}
		return nil
	}
}

func MakeDupStackFunc(n int) StackValidationFunc {
	return MakeStackFunc(n, n+1)
}

func MakeSwapStackFunc(n int) StackValidationFunc {
	return MakeStackFunc(n, n)
}
