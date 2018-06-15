package params

const (
	MaxCodeSize            = 24576 // 合约允许的最大字节数
	CallCreateDepth uint64 = 1024  // 合约递归调用最大深度
	StackLimit      uint64 = 1024  // 栈允许的最大深度

	// 各种操作对应的Gas定价
	CreateDataGas        uint64 = 200   // 创建合约时，按字节计费
	CallStipend          uint64 = 2300  // 每次CALL调用之前，给予一定额度的免费Gas
	CallValueTransferGas uint64 = 9000  // 转账操作
	CallNewAccountGas    uint64 = 25000 // 操作目标地址事先不存在
	QuadCoeffDiv         uint64 = 512   // 计算开辟内存花费时，在计算出的内存大小平方基础上除此值
	CopyGas              uint64 = 3     // 内存数据复制时，按字计费

	Sha3Gas          uint64 = 30    // SHA3操作
	Sha3WordGas      uint64 = 6     // SHA3操作的数据按字计费
	SstoreSetGas     uint64 = 20000 // SSTORE 从零值地址到非零值地址存储
	SstoreResetGas   uint64 = 5000  // SSTORE 从非零值地址到非零值地址存储
	SstoreClearGas   uint64 = 5000  // SSTORE 从非零值地址到零值地址存储
	SstoreRefundGas  uint64 = 15000 // SSTORE 删除值时给予的奖励
	JumpdestGas      uint64 = 1     // JUMPDEST 指令
	LogGas           uint64 = 375   // LOGN 操作计费
	LogDataGas       uint64 = 8     // LOGN生成的数据，每个字节的计费价格
	LogTopicGas      uint64 = 375   // LOGN 生成日志时，使用N*此值计费
	CreateGas        uint64 = 32000 // CREATE 指令
	SuicideRefundGas uint64 = 24000 // SUICIDE 操作时给予的奖励
	MemoryGas        uint64 = 3     // 开辟新内存时按字收费

	// 预编译合约的Gas定价
	EcrecoverGas            uint64 = 3000   // ecrecover 指令
	Sha256BaseGas           uint64 = 60     // SHA256 基础计费
	Sha256PerWordGas        uint64 = 12     // SHA256 按字长计费 （总计费等于两者相加）
	Ripemd160BaseGas        uint64 = 600    // RIPEMD160 基础计费
	Ripemd160PerWordGas     uint64 = 120    // RIPEMD160 按字长计费 （总计费等于两者相加）
	IdentityBaseGas         uint64 = 15     // dataCopy 基础计费
	IdentityPerWordGas      uint64 = 3      // dataCopy 按字长计费（总计费等于两者相加）
	ModExpQuadCoeffDiv      uint64 = 20     // 大整数取模运算时计算出的费用除此数
	Bn256AddGas             uint64 = 500    // Bn256Add 计费
	Bn256ScalarMulGas       uint64 = 40000  // Bn256ScalarMul 计费
	Bn256PairingBaseGas     uint64 = 100000 // bn256Pairing 基础计费
	Bn256PairingPerPointGas uint64 = 80000  // bn256Pairing 按point计费（总计费等于两者相加）
)
