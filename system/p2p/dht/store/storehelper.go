package store

import (
	"context"

	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

//Helper for testing
type Helper struct {
}

func (s *Helper) putBlock(routing *dht.IpfsDHT, block *types.Block, blockHash []byte) error {
	key := makeBlockHashAsKey(blockHash)
	value, err := proto.Marshal(block)
	if err != nil {
		return err
	}
	return routing.PutValue(context.Background(), key, value)
}

func (s *Helper) getBlockByHash(routing *dht.IpfsDHT, blockHash []byte) (*types.Block, error) {
	key := makeBlockHashAsKey(blockHash)
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
