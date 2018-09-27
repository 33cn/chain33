package mavl

import (
	"bytes"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/types"
	"fmt"
)

//const proofLimit = 1 << 16 // 64 KB

//merkle avl tree proof证明结构体
type Proof struct {
	LeafHash   []byte
	InnerNodes []*types.InnerNode
	RootHash   []byte
}

// key:value 的proof确认
func (proof *Proof) Verify(key []byte, value []byte, root []byte) bool {
	if !bytes.Equal(proof.RootHash, root) {
		return false
	}
	leafNode := types.LeafNode{Key: key, Value: value, Height: 0, Size: 1}
	leafHash := leafNode.Hash()


	if enableHashPrefix {
		hashKey := bytes.TrimSuffix(proof.LeafHash, leafHash)
		hashKey = append(hashKey, leafHash...)
		leafHash = hashKey
	}

	if !bytes.Equal(leafHash, proof.LeafHash) {
		return false
	}
	hash := leafHash
	for i, branch := range proof.InnerNodes {
		//hash = branch.ProofHash(hash)
		hash = InnerNodeProofHash(hash, branch)
		if enableHashPrefix && i != len(proof.InnerNodes) - 1 {
			node := proof.InnerNodes[i + 1]
			var hashKey []byte
			if len(node.LeftHash) == 0 {
				hashKey = node.RightHash
			} else {
				hashKey = node.LeftHash
			}
			elem := bytes.Split(hashKey, []byte("-"))
			if len(elem) >= 3 {
				hashKey = []byte(fmt.Sprintf("%s-%s-", hashNodePrefix, string(elem[2])))
			}
			hashKey = append(hashKey, hash...)
			hash = hashKey
		}
	}
	return bytes.Equal(proof.RootHash, hash)
}

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

func (node *Node) constructProof(t *Tree, key []byte, valuePtr *[]byte, proof *Proof) (exists bool) {
	if node.height == 0 {
		if bytes.Equal(node.key, key) {
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
	} else {
		return nil, nil
	}
}
