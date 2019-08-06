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
	proto "github.com/golang/protobuf/proto"
)

// Hash 获取block的hash值
func (block *Block) Hash() []byte {
	if IsFork(block.Height, "ForkBlockHash") {
		return block.HashNew()
	}
	return block.HashOld()
}

//HashByForkHeight hash 通过自己设置的fork 高度计算 hash
func (block *Block) HashByForkHeight(forkheight int64) []byte {
	if block.Height >= forkheight {
		return block.HashNew()
	}
	return block.HashOld()
}

//HashNew 新版本的Hash
func (block *Block) HashNew() []byte {
	data, err := proto.Marshal(block.getHeaderHashNew())
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}

//HashOld 老版本的hash
func (block *Block) HashOld() []byte {
	data, err := proto.Marshal(block.getHeaderHashOld())
	if err != nil {
		panic(err)
	}
	return common.Sha256(data)
}

// Size 获取block的Size
func (block *Block) Size() int {
	return Size(block)
}

// GetHeader 获取block的Header信息
func (block *Block) GetHeader() *Header {
	head := &Header{}
	head.Version = block.Version
	head.ParentHash = block.ParentHash
	head.TxHash = block.TxHash
	head.BlockTime = block.BlockTime
	head.Height = block.Height
	head.Difficulty = block.Difficulty
	head.StateHash = block.StateHash
	head.TxCount = int64(len(block.Txs))
	head.Hash = block.Hash()
	return head
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

// CheckSign 检测block的签名
func (block *Block) CheckSign() bool {
	//检查区块的签名
	if block.Signature != nil {
		hash := block.Hash()
		sign := block.GetSignature()
		if !CheckSign(hash, "", sign) {
			return false
		}
	}
	//检查交易的签名
	cpu := runtime.NumCPU()
	ok := checkAll(block.Txs, cpu)
	return ok
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

func check(data *Transaction) bool {
	return data.CheckSign()
}

func checksign(done <-chan struct{}, taskes <-chan *Transaction, c chan<- result) {
	for task := range taskes {
		select {
		case c <- result{check(task)}:
		case <-done:
			return
		}
	}
}

func checkAll(task []*Transaction, n int) bool {
	done := make(chan struct{})
	defer close(done)

	taskes := gen(done, task)

	// Start a fixed number of goroutines to read and digest files.
	c := make(chan result) // HLc
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			checksign(done, taskes, c) // HLc
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

// CheckSign 检测签名
func CheckSign(data []byte, execer string, sign *Signature) bool {
	//GetDefaultSign: 系统内置钱包，非插件中的签名
	c, err := crypto.New(GetSignName(execer, int(sign.Ty)))
	if err != nil {
		return false
	}
	pub, err := c.PubKeyFromBytes(sign.Pubkey)
	if err != nil {
		return false
	}
	signbytes, err := c.SignatureFromBytes(sign.Signature)
	if err != nil {
		return false
	}
	return pub.VerifyBytes(data, signbytes)
}

//FilterParaTxsByTitle 过滤指定title的平行链交易
//1，单笔平行连交易
//2,交易组中的平行连交易，需要将整个交易组都过滤出来
//目前暂时不返回单个交易的proof证明路径，
//后面会将平行链的交易组装到一起，构成一个子roothash。会返回子roothash的proof证明路径
func (blockDetail *BlockDetail) FilterParaTxsByTitle(title string) *ParaTxDetail {
	var paraTx ParaTxDetail
	paraTx.Header = blockDetail.Block.GetHeader()

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

//filterParaTxGroup 获取para交易所在交易组信息
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
