package runtime

import (
	"math/big"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
)

// Destinations 存储合约以及代码对应的位向量对象
// Key为代码哈希，Value为对应的合约代码的位向量
// JUMPDEST指令会使用此对象进行跳转位置判断
type Destinations map[common.Hash]bitvec

// 检查PC只想的代码是否存在JUMPDEST指令，并且跳转目标有效
func (d Destinations) Has(codehash common.Hash, code []byte, dest *big.Int) bool {
	// 首先需要检查PC（指令指针），它不可能比代码长度还大，也不可能大于63位
	// 注意，这里的参数dest就是PC指针
	udest := dest.Uint64()
	if dest.BitLen() >= 63 || udest >= uint64(len(code)) {
		return false
	}
	// 查看丢应代码是否已经解析，如果没有则，立即执行解析
	m, analysed := d[codehash]
	if !analysed {
		m = codeBitmap(code)
		d[codehash] = m
	}

	// PC跳转位置，并且跳转的目的地为有效代码段
	return OpCode(code[udest]) == JUMPDEST && m.codeSegment(udest)
}

// 位向量，未设置的位表示操作码，设置的位表示数据
type bitvec []byte

func (bits *bitvec) set(pos uint64) {
	(*bits)[pos/8] |= 0x80 >> (pos % 8)
}
func (bits *bitvec) set8(pos uint64) {
	(*bits)[pos/8] |= 0xFF >> (pos % 8)
	(*bits)[pos/8+1] |= ^(0xFF >> (pos % 8))
}

// 检查pos对应的位置是否为有效代码段开始
func (bits *bitvec) codeSegment(pos uint64) bool {
	return ((*bits)[pos/8] & (0x80 >> (pos % 8))) == 0
}

// 检查区分代码中的指令和数据，并将数据位置1
func codeBitmap(code []byte) bitvec {
	// bitmap长度会多加上4，避免代码以push32结束（这种情况下需要在最后设置4个byte，也就是32个位）
	//
	bits := make(bitvec, len(code)/8+1+4)
	for pc := uint64(0); pc < uint64(len(code)); {
		op := OpCode(code[pc])

		// 从PUSH1到PUSH32的指令是连续的数字，下面才能这样处理
		if op >= PUSH1 && op <= PUSH32 {
			numbits := op - PUSH1 + 1
			pc++

			// 最小按照每8位进行设置
			for ; numbits >= 8; numbits -= 8 {
				bits.set8(pc) // 8
				pc += 8
			}

			// 然后再设置剩余的不足8位的bit
			for ; numbits > 0; numbits-- {
				bits.set(pc)
				pc++
			}
		} else {
			pc++
		}
	}
	return bits
}
