// NCNL is No Copyright and No License

package merkle

import (
	"bytes"
	"crypto/sha256"
	"math"
)

//var zeroHash crypto.Hash

func calculateLeafNumber(n int) int {
	if n&(n-1) == 0 {
		return n
	}
	return 1 << (uint(math.Log2(float64(n))) + 1)
}

func sumChildHash(left, right []byte) []byte {
	leftlen := len(left)
	rightlen := len(right)
	data := make([]byte, leftlen+rightlen)
	copy(data, left[:])
	copy(data[leftlen:], right[:])
	h := sha256.Sum256(data)
	return h[:]
}
func SumChildHash(left, right []byte) []byte {
	leftlen := len(left)
	rightlen := len(right)
	data := make([]byte, leftlen+rightlen)
	copy(data, left[:])
	copy(data[leftlen:], right[:])
	h := sha256.Sum256(data)
	return h[:]
}

//
// func generateMerkle generate a merkle tree like
// 				 root
//			   /	  \
//			n1			n2
//		  /    \      /    \
//		l1		l2	l3		l4
//
// the tree return use a slice: [root][n1][n2][l1][l2][l3][l4]
//
func GenerateMerkle(hashs [][]byte) [][]byte {
	n := len(hashs)
	if n == 0 {
		return nil
	}

	treeSize := calculateLeafNumber(n)*2 - 1
	merkle := make([][]byte, treeSize)

	p := treeSize / 2
	for i, h := range hashs {
		index := p + i
		merkle[index] = h
	}

	for i := 4; i <= treeSize+1; i *= 2 {
		p := treeSize / i
		for j := p; j < p*2+1; j++ {
			left := j*2 + 1
			right := left + 1
			if merkle[left] == nil {
				merkle[j] = nil
				continue
			}
			if merkle[right] == nil {
				merkle[j] = merkle[left]
				continue
			}
			merkle[j] = sumChildHash(merkle[left], merkle[right])
		}
	}
	return merkle
}

// calculate tx and branch root
// br is tx's branch and i is index of tx in the tx list
func BranchRoot(br [][]byte, tx []byte, i int) []byte {
	r := tx
	for _, h := range br {
		if h != nil {
			if i%2 == 0 {
				r = sumChildHash(r, h)
			} else {
				r = sumChildHash(h, r)
			}
		}
		i >>= 1
	}
	return r
}

// 				 root
//			   /	  \
//			n1			n2
//		  /    \      /    \
//		l1		l2	l3		l4
//
// l2't branch is [l1][n2], l3's branch is [l4][n1]
// merkle is slice of merkle tree, i is index of tx in the merkle tree
func MerkleBranch(merkle [][]byte, i int) [][]byte {
	var hashs [][]byte
	for i > 0 {
		if i%2 != 0 {
			hashs = append(hashs, merkle[i+1]) // right
		} else {
			hashs = append(hashs, merkle[i-1]) // left
		}
		i = (i - 1) / 2
	}
	return hashs
}

// 				 root
//			   /	  \
//			n1			n2
//		  /    \      /    \
//		l1		l2	l3		l4
//
// l2't branch is [l1][n2], l3's branch is [l4][n1]
// txs is hashs of tx list, i is index of the tx in the txs
func ComputeMerkleBranch(txs [][]byte, i int) ([][]byte, int) {
	m := GenerateMerkle(txs)
	index := len(m)/2 + i
	return MerkleBranch(m, index), index
}

func Unmatch(m1, m2 [][]byte, i int) (hs [][]byte) {
	if i >= len(m1) {
		return
	}
	//if m1[i].String() != m2[i].String() {
	if !bytes.Equal(m1[i], m2[i]) {
		if i >= len(m1)/2 {
			hs = append(hs, m1[i])
		} else {
			lhs := Unmatch(m1, m2, i*2+1) // left child
			hs = append(hs, lhs...)
			rhs := Unmatch(m1, m2, i*2+2) // right child
			hs = append(hs, rhs...)
		}
	}
	return
}
