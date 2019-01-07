// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
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
