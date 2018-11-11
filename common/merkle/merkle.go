package merkle

import (
	"bytes"
	"crypto/sha256"

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

/* This implements a constant-space merkle root/path calculator, limited to 2^32 leaves. */
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
			h = GetHashFromTwoHash(inner[level], h)
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
		h = GetHashFromTwoHash(h, h)
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
			h = GetHashFromTwoHash(inner[level], h)
			level++
		}
	}
	return h, mutated, branch
}

//计算左右节点hash的父hash
func GetHashFromTwoHash(left []byte, right []byte) []byte {
	if left == nil || right == nil {
		return nil
	}
	leftlen := len(left)
	rightlen := len(right)

	parent := make([]byte, leftlen+rightlen)

	copy(parent, left)
	copy(parent[leftlen:], right)
	hash := sha256.Sum256(parent)
	parenthash := sha256.Sum256(hash[:])
	return parenthash[:]
}

//获取merkle roothash
func GetMerkleRoot(leaves [][]byte) (roothash []byte) {
	if leaves == nil {
		return nil
	}
	proothash, _, _ := Computation(leaves, 1, 0)
	return proothash
}

//获取指定txindex的branch position 从0开始
func GetMerkleBranch(leaves [][]byte, position uint32) [][]byte {
	_, _, branchs := Computation(leaves, 2, position)
	return branchs
}

// 通过branch 获取对应的roothash 用于指定txhash的proof证明
func GetMerkleRootFromBranch(merkleBranch [][]byte, leaf []byte, Index uint32) []byte {
	hash := leaf
	for _, branch := range merkleBranch {
		if (Index & 1) != 0 {
			hash = GetHashFromTwoHash(branch, hash)
		} else {
			hash = GetHashFromTwoHash(hash, branch)
		}
		Index >>= 1
	}
	return hash
}

//获取merkle roothash 以及指定tx index的branch，注释：position从0开始
func GetMerkleRootAndBranch(leaves [][]byte, position uint32) (roothash []byte, branchs [][]byte) {
	roothash, _, branchs = Computation(leaves, 3, position)
	return
}

var zeroHash [32]byte

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
