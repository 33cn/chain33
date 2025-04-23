// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os/user"
	"path/filepath"
	"sync/atomic"
	"syscall"

	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

var (
	fileHeaderKey        = []byte("F:header:")
	endBlockKey          = []byte("F:endblock:")
	blockHeightKey       = []byte("F:blockH:")
	blockCount     int64 = 1024 //只处理当前高度减去1024之前的区块，避免导出侧链的数据
	dbCache        int32 = 4
	exportlog            = chainlog.New("submodule", "export")
)

// errors
var (
	ErrIsOrphan                   = errors.New("ErrIsOrphan")
	ErrIsSideChain                = errors.New("ErrIsSideChain")
	ErrBlockHeightDiscontinuous   = errors.New("ErrBlockHeightDiscontinuous")
	ErrCurHeightMoreThanEndHeight = errors.New("ErrCurHeightMoreThanEndHeight")
)

func calcblockHeightKey(height int64) []byte {
	return append(blockHeightKey, []byte(fmt.Sprintf("%012d", height))...)
}

// ExportBlockProc 通过title导出对应的区块到指定文件中
func (chain *BlockChain) ExportBlockProc(title string, dir string, startHeight int64) {
	//获取文件路径
	dataDir := getDataDir(dir)
	err := chain.ExportBlock(title, dataDir, startHeight)
	exportlog.Info("ExportBlockProc:complete", "title", title, "dir", dir, "err", err)
	syscall.Exit(0)
}

// ImportBlockProc 导入区块的处理，区块导入结束后退出整个系统
func (chain *BlockChain) ImportBlockProc(filename string, dir string) {

	//获取文件路径,空字符串默认当前目录下
	dataDir := getDataDir(dir)

	//从文件导入区块期间，执行区块设置成不刷磁盘，提高写入数据库的效率
	if !chain.cfgBatchSync {
		atomic.CompareAndSwapInt32(&chain.isbatchsync, 1, 0)
	}
	err := chain.ImportBlock(filename, dataDir)
	exportlog.Info("importBlock:complete", "filename", filename, "dir", dir, "err", err)
	syscall.Exit(0)
}

// ExportBlock 通过指定title和起始高度将block信息导出到一个指定文件中。
// title:chain33/bityuan
// startHeight:需要导入/导出的起始高度
// dbPath:存储到指定路径,默认当前目录下
func (chain *BlockChain) ExportBlock(title, dbPath string, startHeight int64) error {
	exportlog.Info("exportBlock", "title", title, "startHeight", startHeight, "dbPath", dbPath)

	if startHeight < 0 {
		return types.ErrInvalidParam
	}
	cfg := chain.client.GetConfig()
	cfgTitle := cfg.GetTitle()
	//title检测
	if len(title) == 0 {
		title = cfgTitle
	}
	if title != cfgTitle {
		exportlog.Error("exportBlock", "title", title, "configTitle", cfgTitle)
		return types.ErrInvalidParam
	}
	//获取blockstore中当前区块的最新高度
	curheight, err := LoadBlockStoreHeight(chain.blockStore.db)
	if err != nil || curheight < 0 {
		exportlog.Error("exportBlock:LoadBlockStoreHeight", "curheight", curheight, "err", err.Error())
		return err
	}

	if startHeight+blockCount >= curheight {
		exportlog.Info("exportBlock:startHeight+blockCount >= height", "curheight", curheight)
		return types.ErrInvalidParam
	}

	exportdb := dbm.NewDB(title, chain.cfg.Driver, dbPath, dbCache)
	defer exportdb.Close()

	var fileExist bool

	//判断导出文件是否已经存在，校验已经存在的导出文件头信息
	oldfileHeader, err := getFileHeader(exportdb)
	if err == nil && oldfileHeader != nil {
		newfileHeader := types.FileHeader{
			Title:   title,
			Driver:  chain.cfg.Driver,
			TestNet: cfg.IsTestNet(),
		}
		if !isValidFileHeader(oldfileHeader, &newfileHeader) {
			exportlog.Error("exportBlock:inValidFileHeader", "oldfileHeader", oldfileHeader, "newfileHeader", newfileHeader.String())
			return types.ErrInValidFileHeader
		}
		//需要在已有导出的endheight区块继续接着导出，需要校验endHeight
		endBlock, err := getEndBlock(exportdb)
		if err != nil {
			exportlog.Error("exportBlock:EndHeight", "error", err)
			return err
		}
		if endBlock.Height < startHeight || endBlock.Height > curheight-blockCount {
			exportlog.Error("exportBlock:endHeight<startHeight", "endHeight", endBlock.Height, "startHeight", startHeight, "error", err)
			return types.ErrBlockHeight
		}

		// 需要校验block是否连续
		block, err := chain.blockStore.LoadBlock(endBlock.Height+1, nil)
		if err != nil {
			exportlog.Error("exportBlock:LoadBlock", "Height", endBlock.Height+1, "error", err)
			return err
		}
		parentHash := block.Block.ParentHash
		if !bytes.Equal(endBlock.Hash, parentHash) {
			exportlog.Error("exportBlock:block discontinuous", "endHeight", endBlock.Height, "hash", common.ToHex(endBlock.Hash), "nextHeight", endBlock.Height+1, "parentHash", common.ToHex(parentHash), "hash", common.ToHex(block.Block.Hash(cfg)))
			return types.ErrBlockHashNoMatch
		}
		fileExist = true
		startHeight = endBlock.Height + 1
	}

	batch := exportdb.NewBatch(false)
	// 设置导出文件头信息
	if !fileExist {
		fileHeader := types.FileHeader{
			Title:       title,
			Driver:      chain.cfg.Driver,
			TestNet:     cfg.IsTestNet(),
			StartHeight: startHeight,
		}
		setFileHeader(batch, &fileHeader)

		endBlock := types.EndBlock{
			Height: startHeight,
			Hash:   zeroHash[:],
		}
		setEndBlock(batch, &endBlock)

		err = batch.Write()
		if err != nil {
			exportlog.Error("exportBlock:batch.Write()", "error", err)
			return err
		}
		batch.Reset()
	}
	return chain.exportMainBlock(startHeight, curheight-blockCount, batch)
}

// 导出主链block信息到文件中
func (chain *BlockChain) exportMainBlock(startHeight, endheight int64, batch dbm.Batch) error {
	cfg := chain.client.GetConfig()
	var count = 0
	for height := startHeight; height <= endheight; height++ {
		block, err := chain.blockStore.LoadBlock(height, nil)
		if err != nil {
			exportlog.Error("exportMainBlock:LoadBlock", "height", height, "error", err)
			return err
		}
		count += block.Size()
		blockinfo := types.Encode(block.Block)
		batch.Set(calcblockHeightKey(height), blockinfo)

		endBlock := types.EndBlock{
			Height: height,
			Hash:   block.Block.Hash(cfg),
		}
		setEndBlock(batch, &endBlock)

		if count > types.MaxBlockSizePerTime {
			exportlog.Info("exportBlock", "height", height)
			err := batch.Write()
			if err != nil {
				storeLog.Error("exportMainBlock:batch.Write()", "height", height, "error", err)
				return err
			}
			batch.Reset()
			count = 0
		}
	}
	if count > 0 {
		err := batch.Write()
		if err != nil {
			exportlog.Error("exportMainBlock:batch.Write()", "height", endheight, "error", err)
			return err
		}
		exportlog.Info("exportBlock:complete!", "endheight", endheight)
		batch.Reset()
	}
	return nil
}

// ImportBlock 通过指定文件导入block
func (chain *BlockChain) ImportBlock(filename, dbPath string) error {
	cfg := chain.client.GetConfig()
	if len(filename) == 0 {
		filename = cfg.GetTitle()
	}

	db := dbm.NewDB(filename, chain.cfg.Driver, dbPath, dbCache)
	defer db.Close()

	//获取文件头信息并做基本的校验
	newfileHeader := types.FileHeader{
		Title:   cfg.GetTitle(),
		Driver:  chain.cfg.Driver,
		TestNet: cfg.IsTestNet(),
	}
	fileHeader, err := getFileHeader(db)

	if err != nil || fileHeader.StartHeight < 0 || !isValidFileHeader(fileHeader, &newfileHeader) {
		exportlog.Error("importBlock:fileHeader", "filename", filename, "dbPath", dbPath, "fileHeader", fileHeader, "cfg.fileHeader", newfileHeader.String(), "err", err)
		return types.ErrInValidFileHeader
	}
	startHeight := fileHeader.StartHeight

	endBlock, err := getEndBlock(db)
	if err != nil {
		exportlog.Error("importBlock:getEndBlock", "error", err)
		return err
	}
	endHeight := endBlock.Height

	// 获取当前链的区块高度,并校验当前高度和要导入的文件中的startHeight高度
	//当前高度小于startHeight时，区块高度无法连续返回错误
	//从当前高度继续读取区块并处理
	curheight := chain.GetBlockHeight()

	if curheight < startHeight {
		exportlog.Error("importBlock", "curheight", curheight, "startHeight", startHeight)
		return ErrBlockHeightDiscontinuous
	}
	if curheight > endHeight {
		exportlog.Error("importBlock", "curheight", curheight, "endHeight", endHeight)
		return ErrCurHeightMoreThanEndHeight
	}
	if curheight >= startHeight {
		startHeight = curheight + 1
	}

	//从文件开始导入block
	for i := startHeight; i <= endHeight; i++ {
		block, err := getBlock(db, i)
		if err != nil {
			exportlog.Error("importBlock:getBlock", "Height", i, "err", err)
			return err
		}
		err = chain.mainChainImport(block)
		if err != nil {
			exportlog.Error("importBlock:mainChainImport", "Height", i, "err", err)
			return err
		}
	}
	return nil
}

// mainChainImport 主链的导入区块处理函数
func (chain *BlockChain) mainChainImport(block *types.Block) error {
	cfg := chain.client.GetConfig()
	blockDetail := types.BlockDetail{
		Block: block,
	}
	exportlog.Info("mainChainImport", "height", block.Height, "Hash", common.ToHex(block.Hash(cfg)), "ParentHash", common.ToHex(block.ParentHash))

	_, isMainChain, isOrphan, err := chain.ProcessBlock(false, &blockDetail, "import", true, -1)
	if err == types.ErrBlockExist {
		return nil
	} else if err != nil {
		return err
	}
	if !isMainChain {
		return ErrIsSideChain
	}
	if isOrphan {
		return ErrIsOrphan
	}
	return nil
}

// isValidFileHeader 校验文件头信息
func isValidFileHeader(oldFileHeader, newFileHeader *types.FileHeader) bool {
	if oldFileHeader.Title != newFileHeader.Title ||
		oldFileHeader.Driver != newFileHeader.Driver ||
		oldFileHeader.TestNet != newFileHeader.TestNet {
		return false
	}
	return true
}

// getFileHeader获取文件头信息
func getFileHeader(db dbm.DB) (*types.FileHeader, error) {
	headertitle, err := db.Get(fileHeaderKey)
	if err != nil {
		return nil, err
	}
	var fileHeader types.FileHeader
	err = types.Decode(headertitle, &fileHeader)
	if err != nil {
		exportlog.Error("getFileHeader", "headertitle", string(headertitle), "err", err)
		return nil, err
	}
	return &fileHeader, nil
}

// setFileHeader 设置文件头信息到数据库中
func setFileHeader(batch dbm.Batch, fileHeader *types.FileHeader) {
	fileHeaderinfo := types.Encode(fileHeader)
	batch.Set(fileHeaderKey, fileHeaderinfo)
}

// getEndBlock 获取endblock的信息
func getEndBlock(db dbm.DB) (*types.EndBlock, error) {
	var endBlock types.EndBlock

	storeEndHeight, err := db.Get(endBlockKey)
	if err != nil {
		exportlog.Error("getEndBlock", "error", err)
		return nil, err
	}
	err = types.Decode(storeEndHeight, &endBlock)
	if err != nil || endBlock.Height < 0 {
		exportlog.Error("getEndBlock:Unmarshal", "storeEndHeight", string(storeEndHeight), "error", err)
		return nil, err
	}
	return &endBlock, nil
}

// setEndBlock 设置endblock的信息
func setEndBlock(batch dbm.Batch, endBlock *types.EndBlock) {
	endBlockinfo := types.Encode(endBlock)
	batch.Set(endBlockKey, endBlockinfo)
}

// 从数据库中获取对应高度的block信息
func getBlock(db dbm.DB, height int64) (*types.Block, error) {
	data, err := db.Get(calcblockHeightKey(height))
	if err != nil {
		exportlog.Error("getBlock:storeblock", "Height", height, "err", err)
		return nil, err
	}
	var block types.Block
	err = types.Decode(data, &block)
	if err != nil {
		exportlog.Error("getBlock:Decode", "err", err)
		return nil, err
	}
	return &block, nil
}

// getDataDir 获取文件路径 "/something/~/something/"
func getDataDir(datadir string) string {

	if len(datadir) >= 2 && datadir[:2] == "~/" {
		usr, err := user.Current()
		if err != nil {
			panic(err)
		}
		dir := usr.HomeDir
		datadir = filepath.Join(dir, datadir[2:])
	}
	if len(datadir) >= 6 && datadir[:6] == "$TEMP/" {
		dir, err := ioutil.TempDir("", "chain33datadir-")
		if err != nil {
			panic(err)
		}
		datadir = filepath.Join(dir, datadir[6:])
	}
	return datadir
}
