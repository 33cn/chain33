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

package mm

import (
	"fmt"
	"math/big"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/runtime"
)

// stack is an object for basic stack operations. Items popped to the stack are
// expected to be changed and modified. stack does not take care of adding newly
// initialised objects.
type Stack struct {
	Items []*big.Int
}

func NewStack() *Stack {
	return &Stack{Items: make([]*big.Int, 0, 1024)}
}

func (st *Stack) Data() []*big.Int {
	return st.Items
}

func (st *Stack) Push(d *big.Int) {
	// NOTE Push limit (1024) is checked in baseCheck
	//stackItem := new(big.Int).Set(d)
	//st.Items = append(st.Items, stackItem)
	st.Items = append(st.Items, d)
}
func (st *Stack) PushN(ds ...*big.Int) {
	st.Items = append(st.Items, ds...)
}

func (st *Stack) Pop() (ret *big.Int) {
	ret = st.Items[len(st.Items)-1]
	st.Items = st.Items[:len(st.Items)-1]
	return
}

func (st *Stack) Len() int {
	return len(st.Items)
}

func (st *Stack) Swap(n int) {
	st.Items[st.Len()-n], st.Items[st.Len()-1] = st.Items[st.Len()-1], st.Items[st.Len()-n]
}

func (st *Stack) Dup(pool *runtime.IntPool, n int) {
	st.Push(pool.Get().Set(st.Items[st.Len()-n]))
}

func (st *Stack) Peek() *big.Int {
	return st.Items[st.Len()-1]
}

// Back returns the n'th item in stack
func (st *Stack) Back(n int) *big.Int {
	return st.Items[st.Len()-n-1]
}

func (st *Stack) Require(n int) error {
	if st.Len() < n {
		return fmt.Errorf("stack underflow (%d <=> %d)", len(st.Items), n)
	}
	return nil
}

func (st *Stack) Print() {
	fmt.Println("### stack ###")
	if len(st.Items) > 0 {
		for i, val := range st.Items {
			fmt.Printf("%-3d  %v\n", i, val)
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("#############")
}
