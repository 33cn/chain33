package mm

import (
	"math/big"
	"sync"
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

// 默认容量
const poolDefaultCap = 25

// 用于管理IntPool的Pool
type IntPoolPool struct {
	pools []*IntPool
	lock  sync.Mutex
}

var PoolOfIntPools = &IntPoolPool{
	pools: make([]*IntPool, 0, poolDefaultCap),
}

// get is looking for an available pool to return.
func (ipp *IntPoolPool) Get() *IntPool {
	ipp.lock.Lock()
	defer ipp.lock.Unlock()

	if len(PoolOfIntPools.pools) > 0 {
		ip := ipp.pools[len(ipp.pools)-1]
		ipp.pools = ipp.pools[:len(ipp.pools)-1]
		return ip
	}
	return NewIntPool()
}

// put a pool that has been allocated with get.
func (ipp *IntPoolPool) Put(ip *IntPool) {
	ipp.lock.Lock()
	defer ipp.lock.Unlock()

	if len(ipp.pools) < cap(ipp.pools) {
		ipp.pools = append(ipp.pools, ip)
	}
}
