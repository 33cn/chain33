package download

import (
	"fmt"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

// F-DL-002: downloadBlockFromPeerOld panics when peer returns empty/malicious response.
// resp.Message.Items[0].Value.(*types.InvData_Block).Block has no length check or comma-ok.

func extractBlockFromRespBuggy(resp *types.MessageGetBlocksResp) (*types.Block, error) {
	// This is the exact logic from download.go:150 (buggy version)
	block := resp.Message.Items[0].Value.(*types.InvData_Block).Block
	return block, nil
}

func extractBlockFromRespFixed(resp *types.MessageGetBlocksResp) (*types.Block, error) {
	if resp.Message == nil || len(resp.Message.Items) == 0 {
		return nil, fmt.Errorf("empty block response from peer")
	}
	blockData, ok := resp.Message.Items[0].Value.(*types.InvData_Block)
	if !ok || blockData == nil || blockData.Block == nil {
		return nil, fmt.Errorf("invalid block data in response")
	}
	return blockData.Block, nil
}

func TestDownloadBlockFromPeerOld_EmptyResponse_Panic(t *testing.T) {
	resp := &types.MessageGetBlocksResp{
		Message: &types.InvDatas{Items: nil},
	}
	// Buggy version panics
	assert.Panics(t, func() {
		extractBlockFromRespBuggy(resp)
	}, "buggy version should panic on empty Items")
	// Fixed version returns error
	_, err := extractBlockFromRespFixed(resp)
	assert.Error(t, err)
}

func TestDownloadBlockFromPeerOld_NilMessage_Panic(t *testing.T) {
	resp := &types.MessageGetBlocksResp{Message: nil}
	assert.Panics(t, func() {
		extractBlockFromRespBuggy(resp)
	}, "buggy version should panic on nil Message")
	_, err := extractBlockFromRespFixed(resp)
	assert.Error(t, err)
}

func TestDownloadBlockFromPeerOld_WrongType_Panic(t *testing.T) {
	resp := &types.MessageGetBlocksResp{
		Message: &types.InvDatas{
			Items: []*types.InvData{{Ty: 1, Value: &types.InvData_Tx{}}},
		},
	}
	assert.Panics(t, func() {
		extractBlockFromRespBuggy(resp)
	}, "buggy version should panic on wrong type")
	_, err := extractBlockFromRespFixed(resp)
	assert.Error(t, err)
}
