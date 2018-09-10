package common

// 合约交易消息模型
// 在EVM执行器中传递此消息，由外部Tx等价构造
type Message struct {
	to       *Address
	from     Address
	alias    string
	nonce    int64
	amount   uint64
	gasLimit uint64
	gasPrice uint32
	data     []byte
}

func NewMessage(from Address, to *Address, nonce int64, amount uint64, gasLimit uint64, gasPrice uint32, data []byte, alias string) *Message {
	return &Message{
		from:     from,
		to:       to,
		nonce:    nonce,
		amount:   amount,
		gasLimit: gasLimit,
		gasPrice: gasPrice,
		data:     data,
		alias:    alias,
	}
}

func (m Message) From() Address    { return m.from }
func (m Message) To() *Address     { return m.to }
func (m Message) GasPrice() uint32 { return m.gasPrice }
func (m Message) Value() uint64    { return m.amount }
func (m Message) Nonce() int64     { return m.nonce }
func (m Message) Data() []byte     { return m.data }
func (m Message) GasLimit() uint64 { return m.gasLimit }
func (m Message) Alias() string    { return m.alias }
