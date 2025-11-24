package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Header block header
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

// Transaction represents a transaction that will serialize to the RPC representation of a transaction
type Transaction struct {
	BlockHash        *common.Hash      `json:"blockHash"`
	BlockNumber      *hexutil.Big      `json:"blockNumber"`
	From             common.Address    `json:"from"`
	Gas              hexutil.Uint64    `json:"gas"`
	GasPrice         *hexutil.Big      `json:"gasPrice"`
	GasFeeCap        *hexutil.Big      `json:"maxFeePerGas,omitempty"`
	GasTipCap        *hexutil.Big      `json:"maxPriorityFeePerGas,omitempty"`
	Hash             common.Hash       `json:"hash"`
	Input            hexutil.Bytes     `json:"input"`
	Nonce            hexutil.Uint64    `json:"nonce"`
	To               *common.Address   `json:"to"`
	TransactionIndex *hexutil.Uint64   `json:"transactionIndex"`
	Value            *hexutil.Big      `json:"value"`
	Type             hexutil.Uint64    `json:"type"`
	Accesses         *types.AccessList `json:"accessList,omitempty"`
	ChainID          *hexutil.Big      `json:"chainId,omitempty"`
	V                *hexutil.Big      `json:"v"`
	R                *hexutil.Big      `json:"r"`
	S                *hexutil.Big      `json:"s"`
}

// Transactions txs types
type Transactions []*Transaction

// Block ETH 交易结构体
type Block struct {
	*Header
	Uncles       []*Header   `json:"uncles"`
	Transactions interface{} `json:"transactions"`
	Hash         string      `json:"hash"`
}

// Receipt tx Receipt
type Receipt struct {
	Type              hexutil.Uint64  `json:"type"`
	Status            hexutil.Uint64  `json:"status"`
	CumulativeGasUsed hexutil.Uint64  `json:"cumulativeGasUsed" gencodec:"required"`
	Bloom             types.Bloom     `json:"logsBloom"         gencodec:"required"`
	Logs              []*EvmLog       `json:"logs"              gencodec:"required"`
	TxHash            common.Hash     `json:"transactionHash" gencodec:"required"`
	ContractAddress   *common.Address `json:"contractAddress"`
	GasUsed           hexutil.Uint64  `json:"gasUsed" gencodec:"required"`
	BlockHash         common.Hash     `json:"blockHash,omitempty"`
	BlockNumber       *hexutil.Big    `json:"blockNumber,omitempty"`
	TransactionIndex  hexutil.Uint    `json:"transactionIndex"`
	From              *common.Address `json:"from"`
}

// CallMsg eth api param
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

// EvmLog evm log
type EvmLog struct {
	Address     common.Address `json:"address"`
	Topics      []common.Hash  `json:"topics"`
	Data        hexutil.Bytes  `json:"data"`
	BlockNumber hexutil.Uint64 `json:"blockNumber"`
	TxHash      common.Hash    `json:"transactionHash"`
	TxIndex     hexutil.Uint   `json:"transactionIndex"`
	BlockHash   common.Hash    `json:"blockHash"`
	Index       hexutil.Uint   `json:"logIndex"`
	Removed     bool           `json:"removed"`
}

// Peer peer info
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

// Network network info
type Network struct {
	LocalAddress  string `json:"localAddress,omitempty"`
	RemoteAddress string `json:"remoteAddress,omitempty"`
}

// Protocols  peer protocols
type Protocols struct {
	EthProto *EthProto `json:"eth,omitempty"`
}

// EthProto eth proto
type EthProto struct {
	Difficulty uint32 `json:"difficulty,omitempty"`
	Head       string `json:"head,omitempty"`
	Version    string `json:"version,omitempty"`
	NetworkID  int    `json:"network,omitempty"`
}

// Ports ...
type Ports struct {
	Discovery int32 `json:"discovery,omitempty"`
	Listener  int32 `json:"listener,omitempty"`
}

// FilterQuery ...
// logs Subscription
type FilterQuery struct {
	Address   interface{}   `json:"address,omitempty"`
	Topics    []interface{} `json:"topics,omitempty"`
	BlockHash *common.Hash  // used by eth_getLogs, return logs only from block with this hash
	FromBlock string        `json:"fromBlock,omitempty"` // beginning of the queried range, nil means genesis block
	ToBlock   string        `json:"toBlock,omitempty"`
}
