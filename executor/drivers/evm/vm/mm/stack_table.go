// Copyright 2017 The go-ethereum Authors
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

package mm

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/params"
)

type (
	StackValidationFunc func(*Stack) error
)

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
