package mm

import (
	"fmt"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"math/big"
)

// 内存操作封装，在EVM中使用此对象模拟物理内存
type Memory struct {
	// 内存中存储的数据
	Store []byte
	// 上次开辟内存消耗的Gas
	LastGasCost uint64
}

func NewMemory() *Memory {
	return &Memory{}
}

// 设置内存中的值， value => offset:offset + size
func (m *Memory) Set(offset, size uint64, value []byte) {
	// 偏移量+大小一定不会大于内存长度
	if offset + size > uint64(len(m.Store)) {
		panic("INVALID memory: Store empty")
	}

	if size > 0 {
		copy(m.Store[offset:offset+size], value)
	}
}

// 扩充内存到指定大小
func (m *Memory) Resize(size uint64) {
	if uint64(m.Len()) < size {
		m.Store = append(m.Store, make([]byte, size-uint64(m.Len()))...)
	}
}

// 获取内存中制定偏移量开始的指定长度的数据，返回数据的拷贝而非引用
func (m *Memory) Get(offset, size int64) (cpy []byte) {
	if size == 0 {
		return nil
	}

	if len(m.Store) > int(offset) {
		cpy = make([]byte, size)
		copy(cpy, m.Store[offset:offset+size])

		return
	}

	return
}


// 同Get操作，不过这里返回的是数据引用
func (m *Memory) GetPtr(offset, size int64) []byte {
	if size == 0 {
		return nil
	}

	if len(m.Store) > int(offset) {
		return m.Store[offset : offset+size]
	}

	return nil
}

// 返回内存中已开辟空间的大小（以字节计算）
func (m *Memory) Len() int {
	return len(m.Store)
}

// 返回内存中的原始数据引用
func (m *Memory) Data() []byte {
	return m.Store
}


// 打印内存中的数据（调试用）
func (m *Memory) Print() {
	fmt.Printf("### mem %d bytes ###\n", len(m.Store))
	if len(m.Store) > 0 {
		addr := 0
		for i := 0; i+32 <= len(m.Store); i += 32 {
			fmt.Printf("%03d: % x\n", addr, m.Store[i:i+32])
			addr++
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("####################")
}

// 计算所需的内存偏移量和数据大小，计算所需内存大小
func calcMemSize(off, l *big.Int) *big.Int {
	if l.Sign() == 0 {
		return common.Big0
	}

	return new(big.Int).Add(off, l)
}
