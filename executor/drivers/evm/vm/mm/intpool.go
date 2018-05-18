package mm

import (
	"math/big"
)

// 整数池允许的最大长度
const poolLimit = 256

// big.Int组成的内存池
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
		p.pool.Push(i)
	}
}

// 返回一个零值的big.Int
func (p *IntPool) GetZero() *big.Int {
	if p.pool.Len() > 0 {
		return p.pool.Pop().SetUint64(0)
	}
	return new(big.Int)
}
