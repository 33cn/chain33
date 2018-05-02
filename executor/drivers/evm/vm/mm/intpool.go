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
	"math/big"
)

var checkVal = big.NewInt(-42)

const poolLimit = 256

// IntPool is a pool of big integers that
// can be reused for all big.Int operations.
type IntPool struct {
	pool *Stack
}

func NewIntPool() *IntPool {
	return &IntPool{pool: NewStack()}
}

func (p *IntPool) Get() *big.Int {
	if p.pool.Len() > 0 {
		return p.pool.Pop()
	}
	return new(big.Int)
}
func (p *IntPool) Put(is ...*big.Int) {
	if len(p.pool.Items) > poolLimit {
		return
	}

	for _, i := range is {
		// VerifyPool is a build flag. Pool verification makes sure the integrity
		// of the integer pool by comparing values to a default value.
		if VerifyPool {
			i.Set(checkVal)
		}

		p.pool.Push(i)
	}
}
