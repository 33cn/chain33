// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"os"
	"testing"
	"time"

	dbm "github.com/33cn/chain33/common/db"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportBlockProc(t *testing.T) {
	mock33 := testnode.New("", nil)
	blockchain := mock33.GetBlockChain()
	chainlog.Info("TestExportBlockProc begin --------------------")

	//构造1030个区块
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 1030
	var err error
	cfg := mock33.GetClient().GetConfig()
	for {
		_, err = addMainTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI())
		require.NoError(t, err)

		curheight = blockchain.GetBlockHeight()
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}

	//导出block信息,startheight从1开始
	title := mock33.GetCfg().Title
	driver := mock33.GetCfg().BlockChain.Driver
	dbPath := ""
	blockchain.ExportBlock(title, dbPath, 0)
	require.NoError(t, err)
	//读取文件头信息
	db := dbm.NewDB(title, driver, dbPath, 4)
	headertitle, err := db.Get([]byte("F:header:"))
	require.NoError(t, err)

	var fileHeader types.FileHeader
	err = types.Decode(headertitle, &fileHeader)
	require.NoError(t, err)
	assert.Equal(t, driver, fileHeader.Driver)
	assert.Equal(t, int64(0), fileHeader.GetStartHeight())
	assert.Equal(t, title, fileHeader.GetTitle())
	db.Close()
	mock33.Close()

	testImportBlockProc(t)
	chainlog.Info("TestExportBlock end --------------------")
}

func testImportBlockProc(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	blockchain := mock33.GetBlockChain()
	chainlog.Info("TestImportBlockProc begin --------------------")
	title := mock33.GetCfg().Title
	dbPath := ""
	driver := mock33.GetCfg().BlockChain.Driver

	//读取文件头信息
	db := dbm.NewDB(title, driver, dbPath, 4)
	var endBlock types.EndBlock
	endHeight, err := db.Get([]byte("F:endblock:"))
	require.NoError(t, err)
	err = types.Decode(endHeight, &endBlock)
	require.NoError(t, err)
	db.Close()

	err = blockchain.ImportBlock(title, dbPath)
	require.NoError(t, err)
	curHeader, err := blockchain.ProcGetLastHeaderMsg()
	assert.Equal(t, curHeader.Height, endBlock.GetHeight())
	assert.Equal(t, curHeader.GetHash(), endBlock.GetHash())
	file := title + ".db"
	os.RemoveAll(file)

	chainlog.Info("TestImportBlockProc end --------------------")
}
