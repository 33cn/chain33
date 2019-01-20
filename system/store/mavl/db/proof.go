// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mavl

import (
	"bytes"

	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
)

//const proofLimit = 1 << 16 // 64 KB

// Proof merkle avl tree proof证明结构体
type Proof struct {
	LeafHash   []byte
	InnerNodes []*types.InnerNode
	RootHash   []byte
}

var sha256Len = 32

// Verify key:value 的proof确认
func (proof *Proof) Verify(key []byte, value []byte, root []byte) bool {
	if !bytes.Equal(proof.RootHash, root) {
		return false
	}
	leafNode := types.LeafNode{Key: key, Value: value, Height: 0, Size: 1}
	leafHash := leafNode.Hash()

	hashLen := sha256Len
	if len(proof.LeafHash) > hashLen {
		proof.LeafHash = proof.LeafHash[len(proof.LeafHash)-hashLen:]
	}

	if !bytes.Equal(leafHash, proof.LeafHash) {
		return false
	}
	hash := leafHash
	for _, branch := range proof.InnerNodes {
		//hash = branch.ProofHash(hash)
		hash = InnerNodeProofHash(hash, branch)
	}
	return bytes.Equal(proof.RootHash, hash)
}

// Root 证明节点的root hash
func (proof *Proof) Root() []byte {
	return proof.RootHash
}

// ReadProof will deserialize a MAVLProof from bytes
func ReadProof(roothash []byte, leafhash []byte, data []byte) (*Proof, error) {
	var mavlproof types.MAVLProof
	err := proto.Unmarshal(data, &mavlproof)
	if err != nil {
		treelog.Error("Unmarshal err!", "err", err)
		return nil, err
	}
	var merkleAvlProof Proof
	merkleAvlProof.InnerNodes = mavlproof.InnerNodes
	merkleAvlProof.LeafHash = leafhash
	merkleAvlProof.RootHash = roothash
	return &merkleAvlProof, nil
}

// InnerNodeProofHash 计算inner节点的hash
func InnerNodeProofHash(childHash []byte, branch *types.InnerNode) []byte {
	var innernode types.InnerNode

	innernode.Height = branch.Height
	innernode.Size = branch.Size

	// left is nil
	if len(branch.LeftHash) == 0 {
		innernode.LeftHash = childHash
		innernode.RightHash = branch.RightHash
	} else {
		innernode.LeftHash = branch.LeftHash
		innernode.RightHash = childHash
	}
	return innernode.Hash()
}

func (node *Node) constructProof(t *Tree, key []byte, valuePtr *[]byte, proof *Proof) (exists bool) {
	if node.height == 0 {
		if bytes.Equal(node.key, key) {
			*valuePtr = node.value
			proof.LeafHash = node.hash
			return true
		}
		return false
	}
	if bytes.Compare(key, node.key) < 0 {
		exists := node.getLeftNode(t).constructProof(t, key, valuePtr, proof)
		if !exists {
			return false
		}
		branch := types.InnerNode{
			Height:    node.height,
			Size:      node.size,
			LeftHash:  nil,
			RightHash: node.getRightNode(t).hash,
		}
		proof.InnerNodes = append(proof.InnerNodes, &branch)
		return true
	}
	exists = node.getRightNode(t).constructProof(t, key, valuePtr, proof)
	if !exists {
		return false
	}
	branch := types.InnerNode{
		Height:    node.height,
		Size:      node.size,
		LeftHash:  node.getLeftNode(t).hash,
		RightHash: nil,
	}
	proof.InnerNodes = append(proof.InnerNodes, &branch)
	return true
}

// ConstructProof Returns nil, nil if key is not in tree.
func (t *Tree) ConstructProof(key []byte) (value []byte, proof *Proof) {
	if t.root == nil {
		return nil, nil
	}
	t.root.Hash(t) // Ensure that all hashes are calculated.
	proof = &Proof{
		RootHash: t.root.hash,
	}
	exists := t.root.constructProof(t, key, &value, proof)
	if exists {
		return value, proof
	}
	return nil, nil
}
