// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"runtime"
	"sync"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/crypto"
)

// Hash 获取block的hash值
func (block *Block) Hash(cfg *Chain33Config) []byte {
	if cfg.IsFork(block.Height, "ForkBlockHash") {
		return block.HashNew()
	}
	return block.HashOld()
}

// HashByForkHeight hash 通过自己设置的fork 高度计算 hash
func (block *Block) HashByForkHeight(forkheight int64) []byte {
	if block.Height >= forkheight {
		return block.HashNew()
	}
	return block.HashOld()
}

// HashNew 新版本的Hash
func (block *Block) HashNew() []byte {
	data := Encode(block.getHeaderHashNew())
	return common.Sha256(data)
}

// HashOld 老版本的hash
func (block *Block) HashOld() []byte {
	data := Encode(block.getHeaderHashOld())
	return common.Sha256(data)
}

// Size 获取block的Size
func (block *Block) Size() int {
	return Size(block)
}

// GetHeader 获取block的Header信息
func (block *Block) GetHeader(cfg *Chain33Config) *Header {
	head := &Header{}
	head.Version = block.Version
	head.ParentHash = block.ParentHash
	head.TxHash = block.TxHash
	head.BlockTime = block.BlockTime
	head.Height = block.Height
	head.Difficulty = block.Difficulty
	head.StateHash = block.StateHash
	head.TxCount = int64(len(block.Txs))
	head.Hash = block.Hash(cfg)
	return head
}

// SetHeader 设置header, 注意相关字段必须和GetHeader一致, 一起维护
func (block *Block) SetHeader(header *Header) {
	block.Version = header.Version
	block.ParentHash = header.ParentHash
	block.TxHash = header.TxHash
	block.BlockTime = header.BlockTime
	block.Height = header.Height
	block.Difficulty = header.Difficulty
	block.StateHash = header.StateHash
	block.Signature = header.Signature
}

func (block *Block) getHeaderHashOld() *Header {
	head := &Header{}
	head.Version = block.Version
	head.ParentHash = block.ParentHash
	head.TxHash = block.TxHash
	head.BlockTime = block.BlockTime
	head.Height = block.Height
	return head
}

func (block *Block) getHeaderHashNew() *Header {
	head := &Header{}
	head.Version = block.Version
	head.ParentHash = block.ParentHash
	head.TxHash = block.TxHash
	head.BlockTime = block.BlockTime
	head.Height = block.Height
	head.Difficulty = block.Difficulty
	head.StateHash = block.StateHash
	head.TxCount = int64(len(block.Txs))
	return head
}

// VerifySignature 验证区块和交易的签名,支持指定需要验证的交易
func VerifySignature(cfg *Chain33Config, block *Block, txs []*Transaction) bool {
	//检查区块的签名
	if !block.verifySignature(cfg) {
		return false
	}
	//检查交易的签名
	return verifyTxsSignature(txs, block.GetHeight())
}

// CheckSign 检测block的签名,以及交易的签名
func (block *Block) CheckSign(cfg *Chain33Config) bool {
	return VerifySignature(cfg, block, block.Txs)
}

func (block *Block) verifySignature(cfg *Chain33Config) bool {
	if block.GetSignature() == nil {
		return true
	}
	hash := block.Hash(cfg)
	return CheckSign(hash, "", block.GetSignature(), block.GetHeight())
}

func verifyTxsSignature(txs []*Transaction, blockHeight int64) bool {

	//没有需要要验签的交易，直接返回
	if len(txs) == 0 {
		return true
	}
	done := make(chan struct{})
	defer close(done)
	taskes := gen(done, txs)
	cpuNum := runtime.NumCPU()
	// Start a fixed number of goroutines to read and digest files.
	c := make(chan result) // HLc
	var wg sync.WaitGroup
	wg.Add(cpuNum)
	for i := 0; i < cpuNum; i++ {
		go func() {
			checksign(done, taskes, c, blockHeight) // HLc
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c) // HLc
	}()
	// End of pipeline. OMIT
	for r := range c {
		if !r.isok {
			return false
		}
	}
	return true
}

func gen(done <-chan struct{}, task []*Transaction) <-chan *Transaction {
	ch := make(chan *Transaction)
	go func() {
		defer func() {
			close(ch)
		}()
		for i := 0; i < len(task); i++ {
			select {
			case ch <- task[i]:
			case <-done:
				return
			}
		}
	}()
	return ch
}

type result struct {
	isok bool
}

func check(data *Transaction, blockHeight int64) bool {
	return data.CheckSign(blockHeight)
}

func checksign(done <-chan struct{}, taskes <-chan *Transaction, c chan<- result, blockHeight int64) {
	for task := range taskes {
		select {
		case c <- result{check(task, blockHeight)}:
		case <-done:
			return
		}
	}
}

// CheckSign 检测签名
func CheckSign(data []byte, execer string, sign *Signature, blockHeight int64) bool {
	//GetDefaultSign: 系统内置钱包，非插件中的签名
	c, err := crypto.Load(GetSignName(execer, int(sign.Ty)), blockHeight)
	if err != nil {
		return false
	}
	return c.Validate(data, sign.Pubkey, sign.Signature) == nil
}

// FilterParaTxsByTitle 过滤指定title的平行链交易
// 1，单笔平行连交易
// 2,交易组中的平行连交易，需要将整个交易组都过滤出来
// 目前暂时不返回单个交易的proof证明路径，
// 后面会将平行链的交易组装到一起，构成一个子roothash。会返回子roothash的proof证明路径
func (blockDetail *BlockDetail) FilterParaTxsByTitle(cfg *Chain33Config, title string) *ParaTxDetail {
	var paraTx ParaTxDetail
	paraTx.Header = blockDetail.Block.GetHeader(cfg)

	for i := 0; i < len(blockDetail.Block.Txs); i++ {
		tx := blockDetail.Block.Txs[i]
		if IsSpecificParaExecName(title, string(tx.Execer)) {

			//过滤交易组中的para交易，需要将整个交易组都过滤出来
			if tx.GroupCount >= 2 {
				txDetails, endIdx := blockDetail.filterParaTxGroup(tx, i)
				paraTx.TxDetails = append(paraTx.TxDetails, txDetails...)
				i = endIdx - 1
				continue
			}

			//单笔para交易
			var txDetail TxDetail
			txDetail.Tx = tx
			txDetail.Receipt = blockDetail.Receipts[i]
			txDetail.Index = uint32(i)
			paraTx.TxDetails = append(paraTx.TxDetails, &txDetail)

		}
	}
	return &paraTx
}

// filterParaTxGroup 获取para交易所在交易组信息
func (blockDetail *BlockDetail) filterParaTxGroup(tx *Transaction, index int) ([]*TxDetail, int) {
	var headIdx int
	var txDetails []*TxDetail

	for i := index; i >= 0; i-- {
		if bytes.Equal(tx.Header, blockDetail.Block.Txs[i].Hash()) {
			headIdx = i
			break
		}
	}

	endIdx := headIdx + int(tx.GroupCount)
	for i := headIdx; i < endIdx; i++ {
		var txDetail TxDetail
		txDetail.Tx = blockDetail.Block.Txs[i]
		txDetail.Receipt = blockDetail.Receipts[i]
		txDetail.Index = uint32(i)
		txDetails = append(txDetails, &txDetail)
	}
	return txDetails, endIdx
}

// Size 获取blockDetail的Size
func (blockDetail *BlockDetail) Size() int {
	return Size(blockDetail)
}

// Size 获取header的Size
func (header *Header) Size() int {
	return Size(header)
}

// Size 获取paraTxDetail的Size
func (paraTxDetail *ParaTxDetail) Size() int {
	return Size(paraTxDetail)
}
