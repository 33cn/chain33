package types

import proto "github.com/golang/protobuf/proto"
import "code.aliyun.com/chain33/chain33/common"

func (tx *Transaction) Hash() []byte {
	copytx := *tx
	copytx.Signature = nil
	data, err := proto.Marshal(&copytx)
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}

func (block *Block) Hash() []byte {
	head := &Header{}
	head.Version = block.Version
	head.ParentHash = block.ParentHash
	head.TxHash = block.TxHash
	head.BlockTime = block.BlockTime
	head.Height = block.Height
	data, err := proto.Marshal(head)
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}
