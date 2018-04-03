package mavl

import (
	"bytes"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/types"
)

const proofLimit = 1 << 16 // 64 KB

//merkle avl tree proof证明结构体
type MerkleAvlProof struct {
	LeafHash   []byte
	InnerNodes []*types.InnerNode
	RootHash   []byte
}

// key:value 的proof确认
func (proof *MerkleAvlProof) Verify(key []byte, value []byte, root []byte) bool {
	if !bytes.Equal(proof.RootHash, root) {
		return false
	}
	leafNode := types.LeafNode{Key: key, Value: value, Height: 0, Size: 1}
	leafHash := leafNode.Hash()

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

func (proof *MerkleAvlProof) Root() []byte {
	return proof.RootHash
}

// ReadProof will deserialize a MAVLProof from bytes
func ReadProof(roothash []byte, leafhash []byte, data []byte) (*MerkleAvlProof, error) {
	var mavlproof types.MAVLProof
	err := proto.Unmarshal(data, &mavlproof)
	if err != nil {
		treelog.Error("Unmarshal err!", "err", err)
		return nil, err
	}
	var merkleAvlProof MerkleAvlProof
	merkleAvlProof.InnerNodes = mavlproof.InnerNodes
	merkleAvlProof.LeafHash = leafhash
	merkleAvlProof.RootHash = roothash
	return &merkleAvlProof, nil
}

//计算inner节点的hash
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

func (node *MAVLNode) constructProof(t *MAVLTree, key []byte, valuePtr *[]byte, proof *MerkleAvlProof) (exists bool) {
	if node.height == 0 {
		if bytes.Compare(node.key, key) == 0 {
			*valuePtr = node.value
			proof.LeafHash = node.hash
			return true
		} else {
			return false
		}
	} else {
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
		} else {
			exists := node.getRightNode(t).constructProof(t, key, valuePtr, proof)
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
	}
}

// Returns nil, nil if key is not in tree.
func (t *MAVLTree) ConstructProof(key []byte) (value []byte, proof *MerkleAvlProof) {
	if t.root == nil {
		return nil, nil
	}
	t.root.Hash(t) // Ensure that all hashes are calculated.
	proof = &MerkleAvlProof{
		RootHash: t.root.hash,
	}
	exists := t.root.constructProof(t, key, &value, proof)
	if exists {
		return value, proof
	} else {
		return nil, nil
	}
}
