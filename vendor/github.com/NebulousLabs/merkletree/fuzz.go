// +build gofuzz

package merkletree

import (
	"bytes"
	"crypto/sha256"
	"math"
)

// Fuzz is called by go-fuzz to look for inputs to BuildReaderProof that will
// not verify correctly.
func Fuzz(data []byte) int {
	// Use the first two bytes to determine the proof index.
	if len(data) < 2 {
		return -1
	}
	index := 256*uint64(data[0]) + uint64(data[1])
	data = data[2:]

	// Build a reader proof for index 'index' using the remaining data as input
	// to the reader. '64' is chosen as the only input size because that is the
	// size relevant to the Sia project.
	merkleRoot, proofSet, numLeaves, err := BuildReaderProof(bytes.NewReader(data), sha256.New(), 64, index)
	if err != nil {
		return 0
	}
	if !VerifyProof(sha256.New(), merkleRoot, proofSet, index, numLeaves) {
		panic("verification failed!")
	}

	// Output is more interesting when there is enough data to contain the
	// index.
	if uint64(len(data)) > 64*index {
		return 1
	}
	return 0
}

// FuzzReadSubTreesWithProof can be used by go-fuzz to test creating a merkle
// tree from cached subTrees and creating/proving a merkle proof on this tree.
func FuzzReadSubTreesWithProof(data []byte) int {
	// We want at least 2 bytes for the index and 1 for a subTree.
	if len(data) < 3 {
		return -1
	}
	index := 256*uint64(data[0]) + uint64(data[1])
	data = data[2:]
	cachedTree, numLeaves := buildAndCompareTreesFromFuzz(data, index)

	// Create and verify the proof.
	merkleRoot, proofSet, _, numLeaves := cachedTree.Prove()
	if proofSet == nil {
		return 0
	}
	if !VerifyProof(sha256.New(), merkleRoot, proofSet, index, numLeaves) {
		panic("verification failed!")
	}

	// Output is more interesting when there is enough data to contain the
	// index.
	if numLeaves > index {
		return 1
	}
	return 0
}

// FuzzReadSubTreesNoProof can be used by go-fuzz to test creating a merkle
// tree from cached subTrees.
func FuzzReadSubTreesNoProof(data []byte) int {
	buildAndCompareTreesFromFuzz(data, math.MaxUint64)
	if len(data) > 2 {
		return 1
	}
	return 0
}

// buildAndCompareTreesFromFuzz will read the input data and create a subTree
// or leaf for each byte of the input data. It returns the cached tree.
func buildAndCompareTreesFromFuzz(data []byte, proofIndex uint64) (cachedTree *Tree, numLeaves uint64) {
	hash := sha256.New()
	tree := New(hash)
	cachedTree = New(hash)
	if proofIndex != math.MaxUint64 {
		if err := cachedTree.SetIndex(proofIndex); err != nil {
			panic(err)
		}
	}

	for _, b := range data {
		b = b % 6 // should be in range [0;5]

		// if b == 0 we add the data as a leaf.
		if b == 0 {
			data := hash.Sum([]byte{byte(numLeaves)})
			tree.Push(data)
			cachedTree.Push(data)
			numLeaves++
			continue
		}
		// else we create a cached subTree of height b-1
		height := int(b - 1) // should be in range [0:4]
		subTree := New(hash)
		for i := uint64(0); i < 1<<uint64(height); i++ {
			data := hash.Sum([]byte{byte(numLeaves)})
			tree.Push(data)
			subTree.Push(data)
			numLeaves++
		}
		if err := cachedTree.PushSubTree(height, subTree.Root()); err != nil {
			return
		}
	}
	if !bytes.Equal(tree.Root(), cachedTree.Root()) {
		panic("tree roots don't match")
	}
	return
}
