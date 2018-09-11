package executor

/*
测试hashlocking需要测试三种状态：lock ，unlock ，send

首先lock状态：测试账户是否被锁定
所有的coins都是从一个私人密钥priv产生的
只显示或是记录上一个区块header，其中包含Version ，ParentHash，TxHash ，StateHash，Height，BlockTime，TxCount，Hash，Signature
根据时间差生成一个标签label并将其转换成string类型
用genaddress()方法生成一个公钥addrto和一私钥privkey，
privkey是 type PrivKey interface {
	Bytes() []byte
	Sign(msg []byte) Signature
	PubKey() PubKey
	Equals(PrivKey) bool
}接口类型
公钥addrto根据私钥PubKey()进行哈希生成公钥，
addrto是 type Address struct {
	Version  byte
	Hash160  [20]byte // For a stealth address: it's HASH160
	Checksum []byte   // Unused for a stealth address
	Pubkey   []byte   // Unused for a stealth address
	Enc58str string
}结构体类型

将私钥privkey 和标志label作为参数导入到钱包wallet中，其返回值Account类型，
type Account struct {
	Currency int32  `protobuf:"varint,1,opt,name=currency" json:"currency,omitempty"`
	Balance  int64  `protobuf:"varint,2,opt,name=balance" json:"balance,omitempty"`
	Frozen   int64  `protobuf:"varint,3,opt,name=frozen" json:"frozen,omitempty"`
	Addr     string `protobuf:"bytes,4,opt,name=addr" json:"addr,omitempty"`
}
同上再生成另外一个账户的地址，显示其公钥addrto_b

之后用sendtoaddress（）从priv这个初始的账户里向第一个生成的账户公钥地址打入1e10个币

sendtoaddress（）。。。。。是将一系列信息包括交易币amount手续费fee等信息放在交易列表tx中，完成交易，返回空nil
之后展示addrto账户各各信息，accs是WalletAccounts指针类型，其中有一个Wallets是WalletAccount结构体类型的集合，Wallets中有
Acc和Label两个元素变量，Acc是Account类型其中包含了
type Account struct {
	Currency int32  `protobuf:"varint,1,opt,name=currency" json:"currency,omitempty"`
	Balance  int64  `protobuf:"varint,2,opt,name=balance" json:"balance,omitempty"`
	Frozen   int64  `protobuf:"varint,3,opt,name=frozen" json:"frozen,omitempty"`
	Addr     string `protobuf:"bytes,4,opt,name=addr" json:"addr,omitempty"`
}

然后sendtolock（）发送并且锁定，先根据hashlock生成一个合约地址，然后从第一个账户向这个合约地址打入一定数量的coins,同时调用
showAccount（）函数，展示账户信息，最后开始执行lock.

总结：在测试lock状态是否可运行：
首先从一个公共初始私钥地址出向一新生成的地址a打钱，打完之后查看账户信息
其次从a向hashlock的合约地址打钱，收一定的手续费，查看账户信息
最后根据一系列数据进行lock


*/
