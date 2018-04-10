package adapter

const (
	MaxBitLength = 63
)

// 模仿big.Int行为，底层实现可变，封装起来不直接使用big.Int，是为了提升性能
type BigInt struct {
	val int64
}

func NewInt(val int64) *BigInt {
	return &BigInt{val}
}


func (x *BigInt) Sign() int {
	if x.val ==0 {
		return 0
	} else if x.val >0{
		return 1
	}else{
		return -1
	}
}

// Add sets z to the sum x+y and returns z.
func (z *BigInt) Add(x, y *BigInt) *BigInt {
	z.val = x.val+y.val
	return z
}

func (x *BigInt) Uint64() uint64 {
	return uint64(x.val)
}

func (x *BigInt) BitLen() int {
	return MaxBitLength
}