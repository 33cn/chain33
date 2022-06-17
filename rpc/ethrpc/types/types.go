package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

//Header block header
/*type Header struct {
	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"`
	UncleHash   common.Hash    `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    string         `json:"miner"            gencodec:"required"`
	Root        common.Hash    `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash    `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash    `json:"receiptsRoot"     gencodec:"required"`
	Bloom       types.Bloom    `json:"logsBloom"        gencodec:"required"`
	Difficulty  hexutil.Big    `json:"difficulty"       gencodec:"required"`
	Number      hexutil.Big    `json:"number"           gencodec:"required"`
	GasLimit    hexutil.Uint64 `json:"gasLimit"         gencodec:"required"`
	GasUsed     hexutil.Uint64 `json:"gasUsed"          gencodec:"required"`
	Time        hexutil.Uint64 `json:"timestamp"        gencodec:"required"`
	Extra       []byte         `json:"extraData"        gencodec:"required"`
	MixDigest   string         `json:"mixHash"`
	Nonce       hexutil.Uint64 `json:"nonce"`

	// BaseFee was added by EIP-1559 and is ignored in legacy headers.
	BaseFee hexutil.Big `json:"baseFeePerGas" rlp:"optional"`
}*/

//Header block header
type Header struct {
	ParentHash  common.Hash      `json:"parentHash"       gencodec:"required"`
	UncleHash   common.Hash      `json:"sha3Uncles"   gencodec:"required"`
	Coinbase    common.Address   `json:"miner"            gencodec:"required"`
	Root        common.Hash      `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash      `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash      `json:"receiptsRoot"     gencodec:"required"`
	Bloom       types.Bloom      `json:"logsBloom"        gencodec:"required"`
	Difficulty  *hexutil.Big     `json:"difficulty"       gencodec:"required"`
	Number      *hexutil.Big     `json:"number"           gencodec:"required"`
	GasLimit    hexutil.Uint64   `json:"gasLimit"         gencodec:"required"`
	GasUsed     hexutil.Uint64   `json:"gasUsed"          gencodec:"required"`
	Time        hexutil.Uint64   `json:"timestamp"        gencodec:"required"`
	Extra       hexutil.Bytes    `json:"extraData"        gencodec:"required"`
	MixDigest   common.Hash      `json:"mixHash"`
	Nonce       types.BlockNonce `json:"nonce"`
	//BaseFee     *hexutil.Big     `json:"baseFeePerGas" rlp:"optional"`
	Hash common.Hash `json:"hash"`
}

//type Transaction struct {
//	BlockHash        common.Hash   `json:"blockHash,omitempty"`
//	BlockNumber      hexutil.Big   `json:"blockNumber,omitempty"`
//	From             string        `json:"from,omitempty"`
//	To               string        `json:"to,omitempty"`
//	Hash             string        `json:"hash,omitempty"`
//	Data             hexutil.Bytes `json:"input,omitempty"`
//	TransactionIndex hexutil.Uint  `json:"transactionIndex,omitempty"`
//	Value            hexutil.Big   `json:"value,omitempty"`
//	Type             string        `json:"type,omitempty"`
//	V                hexutil.Bytes `json:"v,omitempty"`
//	R                hexutil.Bytes `json:"r,omitempty"`
//	S                hexutil.Bytes `json:"s,omitempty"`
//}

// Transaction LegacyTx is the transaction data of regular Ethereum transactions.
type Transaction struct {
	Type hexutil.Uint64 `json:"type"`

	// Common transaction fields:
	Nonce                *hexutil.Uint64 `json:"nonce"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	Value                *hexutil.Big    `json:"value"`
	Input                *hexutil.Bytes  `json:"input"`
	V                    *hexutil.Big    `json:"v"`
	R                    *hexutil.Big    `json:"r"`
	S                    *hexutil.Big    `json:"s"`
	To                   *common.Address `json:"to"`
	From                 *common.Address `json:"from"`
	// Access list transaction fields:
	ChainID *hexutil.Big `json:"chainId,omitempty"`
	//AccessList *AccessList  `json:"accessList,omitempty"`

	// Only used for encoding:
	Hash common.Hash `json:"hash"`

	BlockNumber *hexutil.Big `json:"blockNumber,omitempty"`
	BlockHash   common.Hash  `json:"blockHash,omitempty"`
}

//Transactions txs types
type Transactions []*Transaction

//Block ETH 交易结构体
type Block struct {
	*Header
	//Uncles []*Header `json:"uncles"`
	//TODO 保留ETH的交易结构类型还是替换为chain33的Transaction
	Transactions interface{} `json:"transactions"`
	Hash         string      `json:"hash"`
}

//Receipt tx Receipt
type Receipt struct {
	Type              hexutil.Uint64  `json:"type,omitempty"`
	PostState         hexutil.Bytes   `json:"root"`
	Status            hexutil.Uint64  `json:"status"`
	CumulativeGasUsed hexutil.Uint64  `json:"cumulativeGasUsed" gencodec:"required"`
	Bloom             types.Bloom     `json:"logsBloom"         gencodec:"required"`
	Logs              []*EvmLog       `json:"logs"              gencodec:"required"`
	TxHash            common.Hash     `json:"transactionHash" gencodec:"required"`
	ContractAddress   *common.Address `json:"contractAddress,omitempty"`
	GasUsed           hexutil.Uint64  `json:"gasUsed" gencodec:"required"`
	BlockHash         common.Hash     `json:"blockHash,omitempty"`
	BlockNumber       *hexutil.Big    `json:"blockNumber,omitempty"`
	TransactionIndex  hexutil.Uint    `json:"transactionIndex"`
	From              *common.Address `json:"from"`
}

//CallMsg eth api param
type CallMsg struct {
	From     string          `json:"from"`
	To       string          `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	// We accept "data" and "input" for backwards-compatibility reasons. "input" is the
	// newer name and should be preferred by clients.
	Data *hexutil.Bytes `json:"data"`
}

//EvmLog evm log
type EvmLog struct {
	Address     *common.Address `json:"address,omitempty"`
	Topics      []common.Hash   `json:"topics,omitempty"`
	Data        *hexutil.Bytes  `json:"data,omitempty"`
	BlockNumber *hexutil.Uint64 `json:"blockNumber,omitempty"`
	TxHash      *common.Hash    `json:"transactionHash,omitempty"`
	TxIndex     hexutil.Uint    `json:"transactionIndex,omitempty"`
	BlockHash   common.Hash     `json:"blockHash,omitempty"`
	Index       hexutil.Uint    `json:"logIndex,omitempty"`
	Removed     bool            `json:"removed,omitempty"`
}

//Peer peer info
type Peer struct {
	ID         string     `json:"id,omitempty"`
	Name       string     `json:"name,omitempty"`
	NetWork    *Network   `json:"netWork,omitempty"`
	Protocols  *Protocols `json:"protocols,omitempty"`
	Self       bool       `json:"self,omitempty"`
	Ports      *Ports     `json:"ports,omitempty"`
	Encode     string     `json:"encode,omitempty"`
	ListenAddr string     `json:"listenAddr,omitempty"`
}

//Network network info
type Network struct {
	LocalAddress  string `json:"localAddress,omitempty"`
	RemoteAddress string `json:"remoteAddress,omitempty"`
}

//Protocols  peer protocols
type Protocols struct {
	EthProto *EthProto `json:"eth,omitempty"`
}

//EthProto eth proto
type EthProto struct {
	Difficulty uint32 `json:"difficulty,omitempty"`
	Head       string `json:"head,omitempty"`
	Version    string `json:"version,omitempty"`
	NetworkID  int    `json:"network,omitempty"`
}

//Ports ...
type Ports struct {
	Discovery int32 `json:"discovery,omitempty"`
	Listener  int32 `json:"listener,omitempty"`
}

//SubLogs ...
//logs Subscription
type SubLogs struct {
	Address string   `json:"address,omitempty"`
	Topics  []string `json:"topics,omitempty"`
}

// EvmLogInfo  ...
type EvmLogInfo struct {
	Address          string      `json:"address,omitempty"`
	BlockHash        string      `json:"blockHash,omitempty"`
	BlockNumber      string      `json:"blockNumber,omitempty"`
	LogIndex         string      `json:"logIndex,omitempty"`
	Topics           interface{} `json:"topics,omitempty"`
	TransactionHash  string      `json:"transactionHash,omitempty"`
	TransactionIndex string      `json:"transactionIndex,omitempty"`
}

//HexRawTx return rawhextx and hash256
type HexRawTx struct {
	RawTx hexutil.Bytes `json:"rawTx,omitempty"`
	//sha3Hash
	Hash      hexutil.Bytes `json:"sha256Hash,omitempty"`
	Signature hexutil.Bytes `json:"signature,omitempty"`
}
