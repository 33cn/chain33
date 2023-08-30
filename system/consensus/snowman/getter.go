package snowman

import "github.com/33cn/chain33/types"

type getter struct {
}

func (g *getter) getLastAcceptBlock() *types.Block {

	return nil
}

func (g *getter) getBlock(hash []byte) (*types.Block, error) {

	return nil, nil
}
