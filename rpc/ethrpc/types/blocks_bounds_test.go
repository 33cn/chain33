package types

import (
	"testing"

	ctypes "github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

func TestBlockDetailToEthBlockRejectsMissingBlock(t *testing.T) {
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())

	_, err := BlockDetailToEthBlock(&ctypes.BlockDetails{}, cfg, false)
	require.Error(t, err)

	_, err = BlockDetailToEthBlock(&ctypes.BlockDetails{
		Items: []*ctypes.BlockDetail{{}},
	}, cfg, false)
	require.Error(t, err)
}
