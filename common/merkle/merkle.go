// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package merkle 实现默克尔树相关的hash计算
package merkle

import (
	"bytes"
	"runtime"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
)

/*     WARNING! If you're reading this because you're learning about crypto
and/or designing a new system that will use merkle trees, keep in mind
that the following merkle tree algorithm has a serious flaw related to
duplicate txids, resulting in a vulnerability (CVE-2012-2459).

The reason is that if the number of hashes in the list at a given time
is odd, the last one is duplicated before computing the next level (which
is unusual in Merkle trees). This results in certain sequences of
transactions leading to the same merkle root. For example, these two
trees:

A               A
/  \            /   \
B     C         B       C
/ \    |        / \     / \
D   E   F       D   E   F   F
/ \ / \ / \     / \ / \ / \ / \
1 2 3 4 5 6     1 2 3 4 5 6 5 6

for transaction lists [1,2,3,4,5,6] and [1,2,3,4,5,6,5,6] (where 5 and
6 are repeated) result in the same root hash A (because the hash of both
of (F) and (F,F) is C).

The vulnerability results from being able to send a block with such a
transaction list, with the same merkle root, and the same block hash as
the original without duplication, resulting in failed validation. If the
receiving node proceeds to mark that block as permanently invalid
however, it will fail to accept further unmodified (and thus potentially
valid) versions of the same block. We defend against this by detecting
the case where we would hash two identical hashes at the end of the list
together, and treating that identically to the block having an invalid
merkle root. Assuming no double-SHA256 collisions, this will detect all
known ways of changing the transactions without affecting the merkle
root.
*/

/*GetMerkleRoot This implements a constant-space merkle root/path calculator, limited to 2^32 leaves. */
//flage =1 只计算roothash  flage =2 只计算branch  flage =3 计算roothash 和 branch
func getMerkleRoot(hashes [][]byte) []byte {
	cache := make([]byte, 64)
	level := 0
	for len(hashes) > 1 {
		if len(hashes)&1 != 0 { //奇数
			hashes = append(hashes, hashes[len(hashes)-1])
		}
		index := 0
		for i := 0; i < len(hashes); i += 2 {
			hashes[index] = GetHashFromTwoHash(cache, hashes[i], hashes[i+1])
			index++
		}
		level++
		hashes = hashes[0:index]
	}
	if len(hashes) == 0 {
		return nil
	}
	return hashes[0]
}

func log2(data int) int {
	level := 1
	if data <= 0 {
		return 0
	}
	for {
		data = data / 2
		if data <= 1 {
			return level
		}
		level++
	}
}

func pow2(d int) (p int) {
	if d <= 0 {
		return 1
	}
	p = 1
	for i := 0; i < d; i++ {
		p *= 2
	}
	return p
}

func calcLevel(n int) int {
	if n == 1 {
		return 1
	}
	level := 0
	for n > 1 {
		if n&1 != 0 {
			n++
		}
		n = n / 2
		level++
	}
	return level
}

func getMerkleRootPad(hashes [][]byte, step int) []byte {
	level1 := calcLevel(len(hashes))
	level2 := log2(step)
	var root []byte
	cache := make([]byte, 64)
	if len(hashes) == 1 {
		root = GetHashFromTwoHash(cache, hashes[0], hashes[0])
	} else {
		root = getMerkleRoot(hashes)
	}
	for i := 0; i < level2-level1; i++ {
		root = GetHashFromTwoHash(cache, root, root)
	}
	return root
}

type childstate struct {
	hash  []byte
	index int
}

//GetMerkleRoot 256构成一个组，进行计算
// n * step = hashes
// (hashes / n)
func GetMerkleRoot(hashes [][]byte) []byte {
	ncpu := runtime.NumCPU()
	if len(hashes) <= 80 || ncpu <= 1 {
		return getMerkleRoot(hashes)
	}
	step := log2(len(hashes) / ncpu)
	if step < 1 {
		step = 1
	}
	step = pow2(step)
	if step > 256 {
		step = 256
	}
	ch := make(chan *childstate, 10)
	//pad to step
	rem := len(hashes) % step
	l := len(hashes) / step
	if rem != 0 {
		l++
	}
	for i := 0; i < l; i++ {
		end := (i + 1) * step
		if end > len(hashes) {
			end = len(hashes)
		}
		child := hashes[i*step : end]
		go func(index int, h [][]byte) {
			var subhash []byte
			if len(h) != step {
				subhash = getMerkleRootPad(h, step)
			} else {
				subhash = getMerkleRoot(h)
			}
			ch <- &childstate{
				hash:  subhash,
				index: index,
			}
		}(i, child)
	}
	childlist := make([][]byte, l)
	for i := 0; i < l; i++ {
		sub := <-ch
		childlist[sub.index] = sub.hash
	}
	return getMerkleRoot(childlist)
}

/*Computation This implements a constant-space merkle root/path calculator, limited to 2^32 leaves. */
//flage =1 只计算roothash  flage =2 只计算branch  flage =3 计算roothash 和 branch
func Computation(leaves [][]byte, flage int, branchpos uint32) (roothash []byte, mutated bool, pbranch [][]byte) {
	if len(leaves) == 0 {
		return nil, false, nil
	}
	if flage < 1 || flage > 3 {
		return nil, false, nil
	}

	var count int
	var branch [][]byte
	var level uint32
	var h []byte
	inner := make([][]byte, 32)
	var matchlevel uint32 = 0xff
	mutated = false
	var matchh bool
	cache := make([]byte, 64)
	for count, h = range leaves {

		if (uint32(count) == branchpos) && (flage&2) != 0 {
			matchh = true
		} else {
			matchh = false
		}
		count++
		// 1左移level位
		for level = 0; 0 == ((count) & (1 << level)); level++ {
			//需要计算branch
			if (flage & 2) != 0 {
				if matchh {
					branch = append(branch, inner[level])
				} else if matchlevel == level {
					branch = append(branch, h)
					matchh = true
				}
			}
			if bytes.Equal(inner[level], h) {
				mutated = true
			}
			//计算inner[level] + h 的hash值
			h = GetHashFromTwoHash(cache, inner[level], h)
		}
		inner[level] = h
		if matchh {
			matchlevel = level
		}
	}

	for level = 0; 0 == (count & (1 << level)); level++ {
	}
	h = inner[level]
	matchh = matchlevel == level
	for count != (1 << level) {
		if (flage&2) != 0 && matchh {
			branch = append(branch, h)
		}
		h = GetHashFromTwoHash(cache, h, h)
		count += (1 << level)
		level++
		// And propagate the result upwards accordingly.
		for 0 == (count & (1 << level)) {
			if (flage & 2) != 0 {
				if matchh {
					branch = append(branch, inner[level])
				} else if matchlevel == level {
					branch = append(branch, h)
					matchh = true
				}
			}
			h = GetHashFromTwoHash(cache, inner[level], h)
			level++
		}
	}
	return h, mutated, branch
}

//GetHashFromTwoHash 计算左右节点hash的父hash
func GetHashFromTwoHash(parent []byte, left []byte, right []byte) []byte {
	if left == nil || right == nil {
		return nil
	}
	copy(parent, left)
	copy(parent[32:], right)
	return common.Sha2Sum(parent)
}

//GetMerkleBranch 获取指定txindex的branch position 从0开始
func GetMerkleBranch(leaves [][]byte, position uint32) [][]byte {
	_, _, branchs := Computation(leaves, 2, position)
	return branchs
}

//GetMerkleRootFromBranch 通过branch 获取对应的roothash 用于指定txhash的proof证明
func GetMerkleRootFromBranch(merkleBranch [][]byte, leaf []byte, Index uint32) []byte {
	hash := leaf
	hashcache := make([]byte, 64)
	for _, branch := range merkleBranch {
		if (Index & 1) != 0 {
			hash = GetHashFromTwoHash(hashcache, branch, hash)
		} else {
			hash = GetHashFromTwoHash(hashcache, hash, branch)
		}
		Index >>= 1
	}
	return hash
}

//GetMerkleRootAndBranch 获取merkle roothash 以及指定tx index的branch，注释：position从0开始
func GetMerkleRootAndBranch(leaves [][]byte, position uint32) (roothash []byte, branchs [][]byte) {
	roothash, _, branchs = Computation(leaves, 3, position)
	return
}

var zeroHash [32]byte

//CalcMerkleRoot 计算merkle树根
func CalcMerkleRoot(txs []*types.Transaction) []byte {
	var hashes [][]byte
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash())
	}
	if hashes == nil {
		return zeroHash[:]
	}
	merkleroot := GetMerkleRoot(hashes)
	if merkleroot == nil {
		panic("calc merkle root error")
	}
	return merkleroot
}

//CalcMerkleRootCache 计算merkle树根缓存
func CalcMerkleRootCache(txs []*types.TransactionCache) []byte {
	var hashes [][]byte
	for _, tx := range txs {
		hashes = append(hashes, tx.Hash())
	}
	if hashes == nil {
		return zeroHash[:]
	}
	merkleroot := GetMerkleRoot(hashes)
	if merkleroot == nil {
		panic("calc merkle root error")
	}
	return merkleroot
}
