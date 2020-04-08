package store

import (
	"context"

	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

type StoreHelper struct {
}

func (s *StoreHelper) PutBlock(routing *dht.IpfsDHT, block *types.Block, blockHash []byte) error {
	key := MakeBlockHashAsKey(blockHash)
	value, err := proto.Marshal(block)
	if err != nil {
		return err
	}
	return routing.PutValue(context.Background(), key, value)
}

func (s *StoreHelper) GetBlockByHash(routing *dht.IpfsDHT, blockHash []byte) (*types.Block, error) {
	key := MakeBlockHashAsKey(blockHash)
	value, err := routing.GetValue(context.Background(), key)
	if err != nil {
		return nil, err
	}

	block := &types.Block{}
	err = proto.Unmarshal(value, block)
	if err != nil {
		return nil, err
	}

	return block, err
}
