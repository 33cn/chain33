package relayd

import (
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	currentBlockHashKey      = []byte("4062c3e6baa0ef4bc5917572c3b9aeaa5dc24f8ef008f38bcd401af6e80f257c")
	currentBtcBlockheightKey = []byte("000007bc077cc540821f06280c4c2f04e036931f21ea3b1bf509b972cbbef5ca")
	executor                 = []byte("relay")
	zeroBlockHeader          = []byte("")
)

type LatestBlock struct {
	Hash       string   `json:"hash"`
	Time       int64    `json:"time"`
	BlockIndex uint64   `json:"block_index"`
	Height     uint64   `json:"height"`
	TxIndexes  []uint64 `json:"txIndexes"`
}

type Header struct {
	Hash         string  `json:"hash"`
	Ver          uint64  `json:"ver"`
	PrevBlock    string  `json:"prev_block"`
	MerkleRoot   string  `json:"mrkl_root"`
	Time         int64   `json:"time"`
	Bits         int64   `json:"bits"`
	Fee          float64 `json:"fee,omitempty"`
	Nonce        int64   `json:"nonce"`
	TxNum        uint64  `json:"n_tx"`
	Size         uint64  `json:"size"`
	BlockIndex   uint64  `json:"block_index"`
	MainChain    bool    `json:"main_chain"`
	Height       uint64  `json:"height"`
	ReceivedTime int64   `json:"received_time"`
	RelayedBy    string  `json:"relayed_by"`
}

func (b *Block) BtcHeader() *types.BtcHeader {
	return &types.BtcHeader{
		Hash:         b.Hash,
		Height:       b.Height,
		MerkleRoot:   b.MerkleRoot,
		Time:         b.Time,
		Nonce:        b.Nonce,
		Bits:         b.Bits,
		PreviousHash: b.PrevBlock,
	}
}

type Block struct {
	// Header
	Hash         string               `json:"hash"`
	Ver          uint64               `json:"ver"`
	PrevBlock    string               `json:"prev_block"`
	MerkleRoot   string               `json:"mrkl_root"`
	Time         int64                `json:"time"`
	Bits         string               `json:"bits"`
	Nonce        uint64               `json:"nonce"`
	Fee          float64              `json:"fee,omitempty"`
	TxNum        uint64               `json:"n_tx"`
	Size         uint64               `json:"size"`
	BlockIndex   uint64               `json:"block_index"`
	MainChain    bool                 `json:"main_chain"`
	Height       uint64               `json:"height"`
	ReceivedTime int64                `json:"received_time"`
	RelayedBy    string               `json:"relayed_by"`
	Tx           []TransactionDetails `json:"tx"`
}

type TransactionDetails struct {
	LockTime  int64     `json:"lock_time"`
	Ver       uint64    `json:"ver"`
	Size      uint64    `json:"size"`
	Inputs    []TxInput `json:"inputs"`
	Weight    int64     `json:"weight"`
	Time      int64     `json:"time"`
	TxIndex   uint64    `json:"tx_index"`
	VinSz     uint64    `json:"vin_sz"`
	Hash      string    `json:"hash"`
	VoutSz    uint64    `json:"vout_sz"`
	RelayedBy string    `json:"relayed_by"`
	Outs      []TxOut   `json:"out"`
}

type TxInput struct {
	Sequence int64  `json:"sequence"`
	Witness  string `json:"witness"`
	Script   string `json:"script"`
}

type TxOut struct {
	Spent   bool   `json:"spent"`
	TxIndex uint64 `json:"tx_index"`
	Type    int    `json:"type"`
	Address string `json:"addr"`
	Value   uint64 `json:"value"`
	N       uint64 `json:"n"`
	Script  string `json:"script"`
}

type Blocks struct {
	Blocks []Block `json:"blocks"`
}

type TransactionResult struct {
	Ver         uint     `json:"ver"`
	Inputs      []Inputs `json:"inputs"`
	Weight      int64    `json:"weight"`
	BlockHeight uint64   `json:"block_height"`
	RelayedBy   string   `json:"relayed_by"`
	Out         []TxOut  `json:"out"`
	LockTime    int64    `json:"lock_time"`
	Size        uint64   `json:"size"`
	DoubleSpend bool     `json:"double_spend"`
	Time        int64    `json:"time"`
	TxIndex     uint64   `json:"tx_index"`
	VinSz       uint64   `json:"vin_sz"`
	Hash        string   `json:"hash"`
	VoutSz      uint64   `json:"vout_sz"`
}

type Inputs struct {
	Sequence uint   `json:"sequence"`
	Witness  string `json:"witness"`
	PrevOut  TxOut  `json:"prev_out"`
	Script   string `json:"script"`
}

func (t *TransactionResult) BtcTransaction() *types.BtcTransaction {
	btcTx := &types.BtcTransaction{}
	btcTx.Hash = t.Hash
	btcTx.Time = t.Time
	btcTx.BlockHeight = t.BlockHeight

	vin := make([]*types.Vin, len(t.Inputs))
	for index, in := range t.Inputs {
		var v types.Vin
		v.Value = in.PrevOut.Value
		v.Address = in.PrevOut.Address
		vin[index] = &v
	}
	btcTx.Vin = vin

	vout := make([]*types.Vout, len(t.Out))
	for index, in := range t.Out {
		var out types.Vout
		out.Value = in.Value
		out.Address = in.Address
		vout[index] = &out
		// TODO
		// vout[index].Coinbase
	}
	btcTx.Vout = vout
	return btcTx
}
