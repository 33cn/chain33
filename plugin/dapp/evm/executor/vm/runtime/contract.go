package runtime

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
)

// 合约对象引用
type ContractRef interface {
	Address() common.Address
}

// 账户对象引用 （实现了合约对象引用ContractRef接口）
// 因为在合约调用过程中，调用者有可能是外部账户，也有可能是合约账户，所以两者的结构是互通的
type AccountRef common.Address

// 将账户引用转换为普通地址对象
func (ar AccountRef) Address() common.Address { return (common.Address)(ar) }

// 合约对象，它在内存中表示一个合约账户的代码即地址信息实体
// 每次合约调用都会创建一个新的合约对象
type Contract struct {
	// 调用者地址，应该为外部账户的地址
	// 如果是通过合约再调用合约时，会从上级合约中获取调用者地址进行赋值
	CallerAddress common.Address

	// 调用此合约的地址，有可能是外部地址（直接调用时），也有可能是合约地址（委托调用时）
	caller ContractRef

	// 一般情况下为合约自身地址
	// 但是，二般情况下（外部账户通过CallCode直接调用合约代码时，此地址会设置为外部账户的地址，就是和caller一样）
	self ContractRef

	// 存储跳转信息，供JUMP和JUMPI指令使用
	Jumpdests Destinations

	// 合约代码和代码哈希
	Code     []byte
	CodeHash common.Hash

	// 合约地址以及输入参数
	CodeAddr *common.Address
	Input    []byte

	// 此合约对象的可用Gas（合约执行过程中会修改此值）
	Gas uint64

	// 合约调用的同时，如果包含转账逻辑，则此处为转账金额
	value uint64

	// 委托调用时，此属性会被设置为true
	DelegateCall bool
}

// 创建一个新的合约调用对象
// 不管合约是否存在，每次调用时都会新创建一个合约对象交给解释器执行，对象持有合约代码和合约地址
func NewContract(caller ContractRef, object ContractRef, value uint64, gas uint64) *Contract {

	c := &Contract{CallerAddress: caller.Address(), caller: caller, self: object}

	// 如果是合约调用合约的情况，则直接赋值父合约的jumpdests
	// 否则，创建一个新的jumpdests
	if parent, ok := caller.(*Contract); ok {
		c.Jumpdests = parent.Jumpdests
	} else {
		c.Jumpdests = make(Destinations)
	}

	// 持有gas引用，方便正确计算消耗的gas
	c.Gas = gas
	c.value = value

	return c
}

// 设置当前的合约对象为被委托调用
// 返回当前合约对象的指针，以便在多层调用链模式下使用
func (c *Contract) AsDelegate() *Contract {
	c.DelegateCall = true

	// 在委托调用模式下，调用者必定为合约对象，而非外部对象
	parent := c.caller.(*Contract)

	// 在一个多层的委托调用链中，调用者地址始终为最初发起合约调用的外部账户地址
	c.CallerAddress = parent.CallerAddress

	// 其它数据正常传递
	c.value = parent.value
	return c
}

// 获取合约代码中制定位置的操作码
func (c *Contract) GetOp(n uint64) OpCode {
	return OpCode(c.GetByte(n))
}

// 获取合约代码中制定位置的字节值
func (c *Contract) GetByte(n uint64) byte {
	if n < uint64(len(c.Code)) {
		return c.Code[n]
	}

	return 0
}

// 返回合约的调用者
// 如果当前合约为委托调用，则调用它的不是外部账户，而是合约账户，所以此时的caller为调用此合约的合约的caller
// 这个关系可以一直递归向上，直到定位到caller为外部账户地址
func (c *Contract) Caller() common.Address {
	return c.CallerAddress
}

// 从合约的可用gas中进行gas消费
func (c *Contract) UseGas(gas uint64) (ok bool) {
	if c.Gas < gas {
		return false
	}
	c.Gas -= gas
	return true
}

// 返回上下文中合约自身的地址
// 注意，当合约通过CallCode调用时，这个地址并不是当前合约代码对应的地址，而是调用者的地址
func (c *Contract) Address() common.Address {
	return c.self.Address()
}

// 合约包含转账逻辑时，转账的金额
func (c *Contract) Value() uint64 {
	return c.value
}

// 设置合约代码内容
func (self *Contract) SetCode(hash common.Hash, code []byte) {
	self.Code = code
	self.CodeHash = hash
}

// 设置合约代码和代码哈希
func (self *Contract) SetCallCode(addr *common.Address, hash common.Hash, code []byte) {
	self.Code = code
	self.CodeHash = hash
	self.CodeAddr = addr
}
