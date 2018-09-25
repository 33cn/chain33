// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package mpt

import (
	"fmt"
	"io"
	"strings"

	proto "github.com/golang/protobuf/proto"
)

var indices = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f", "T"}

type node interface {
	fstring(string) string
	cache() (hashNode, bool)
	create() *Node
	size() int
	canUnload(cachegen, cachelimit uint16) bool
}

const (
	TyFullNode  = 1
	TyShortNode = 2
	TyHashNode  = 3
	TyValueNode = 4
)

type (
	fullNode struct {
		Children [17]node // Actual trie node data to encode/decode (needs custom encoder)
		flags    nodeFlag
		ncache   *Node
	}
	shortNode struct {
		Key    []byte
		Val    node
		flags  nodeFlag
		ncache *Node
	}
)

type hashNode struct {
	*HashNode
	ncache *Node
}

type valueNode struct {
	*ValueNode
	ncache *Node
}

// EncodeRLP encodes a full node into the consensus RLP format.
func (n *fullNode) EncodeProto(w io.Writer) error {
	node := n.create()
	data, err := proto.Marshal(node)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (n *fullNode) create() *Node {
	nodes := make([]*Node, 0)
	for i, n := range n.Children {
		if n != nil {
			nn := n.create()
			nn.Ty = int32(i)<<4 | nn.Ty
			nodes = append(nodes, nn)
		}
	}
	n.ncache = &Node{Ty: TyFullNode, Value: &Node_Full{Full: &FullNode{Nodes: nodes}}}
	return n.ncache
}

func (n *fullNode) size() int {
	k := 0
	for _, n := range n.Children {
		if n != nil {
			k += n.size()
		}
	}
	return k
}

func (n *fullNode) copy() *fullNode   { copy := *n; return &copy }
func (n *shortNode) copy() *shortNode { copy := *n; return &copy }

// nodeFlag contains caching-related metadata about a node.
type nodeFlag struct {
	hash  hashNode // cached hash of the node (may be nil)
	gen   uint16   // cache generation counter
	dirty bool     // whether the node has changes that must be written to the database
}

// canUnload tells whether a node can be unloaded.
func (n *nodeFlag) canUnload(cachegen, cachelimit uint16) bool {
	return !n.dirty && cachegen-n.gen >= cachelimit
}

func (n *fullNode) canUnload(gen, limit uint16) bool  { return n.flags.canUnload(gen, limit) }
func (n *shortNode) canUnload(gen, limit uint16) bool { return n.flags.canUnload(gen, limit) }

func (n *shortNode) create() *Node {
	sn := &ShortNode{Key: n.Key, Val: n.Val.create()}
	n.ncache = &Node{Ty: TyShortNode, Value: &Node_Short{Short: sn}}
	return n.ncache
}

func (n *shortNode) size() int {
	return len(n.Key) + n.Val.size()
}

func (n hashNode) canUnload(uint16, uint16) bool  { return false }
func (n valueNode) canUnload(uint16, uint16) bool { return false }

func (n *fullNode) cache() (hashNode, bool)  { return n.flags.hash, n.flags.dirty }
func (n *shortNode) cache() (hashNode, bool) { return n.flags.hash, n.flags.dirty }
func (n hashNode) cache() (hashNode, bool)   { return hashNode{}, true }
func (n hashNode) create() *Node {
	hn := n.HashNode
	n.ncache = &Node{Ty: TyHashNode, Value: &Node_Hash{Hash: hn}}
	return n.ncache
}

func (n hashNode) size() int {
	return len(n.Hash)
}

func (n valueNode) cache() (hashNode, bool) { return hashNode{}, true }
func (n valueNode) create() *Node {
	vn := n.ValueNode
	n.ncache = &Node{Ty: TyValueNode, Value: &Node_Val{Val: vn}}
	return n.ncache
}

func (n valueNode) size() int {
	return len(n.ValueNode.Value)
}

// Pretty printing.
func (n *fullNode) String() string  { return n.fstring("") }
func (n *shortNode) String() string { return n.fstring("") }
func (n hashNode) String() string   { return n.fstring("") }
func (n valueNode) String() string  { return n.fstring("") }

func (n *fullNode) fstring(ind string) string {
	resp := fmt.Sprintf("[\n%s  ", ind)
	for i, node := range &n.Children {
		if node != nil {
			resp += fmt.Sprintf("%s: %v", indices[i], node.fstring(ind+"  "))
		}
	}
	return resp + fmt.Sprintf("\n%s] ", ind)
}
func (n *shortNode) fstring(ind string) string {
	return fmt.Sprintf("{%s: %v} ", hexToString(n.Key), n.Val.fstring(ind+"  "))
}
func (n hashNode) fstring(ind string) string {
	return fmt.Sprintf("<%x> ", n.Hash)
}
func (n valueNode) fstring(ind string) string {
	return fmt.Sprintf("%s ", string(n.Value))
}

func hexToString(hex []byte) string {
	s := ""
	for i := 0; i < len(hex); i++ {
		if hex[i] > 16 {
			hex = compactToHex(hex)
		}
	}
	for i := 0; i < len(hex); i++ {
		s += indices[hex[i]]
	}
	return s
}

func mustDecodeNode(hash, buf []byte, cachegen uint16) node {
	n, err := decodeNode(hash, buf, cachegen)
	if err != nil {
		panic(fmt.Sprintf("node %x: %v", hash, err))
	}
	return n
}

// decodeNode parses the RLP encoding of a trie node.
func decodeNode(hash, buf []byte, cachegen uint16) (node, error) {
	if len(buf) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	var node = &Node{}
	err := proto.Unmarshal(buf, node)
	if err != nil {
		return nil, err
	}
	return node.decode(hash, cachegen)
}

func createHashNode(hash []byte) (n hashNode) {
	if hash == nil {
		return hashNode{nil, nil}
	}
	n.HashNode = &HashNode{Hash: hash}
	return n
}

func createValueNode(val []byte) (n valueNode) {
	n.ValueNode = &ValueNode{Value: val}
	return n
}

func (n *Node) decode(hash []byte, cachegen uint16) (node, error) {
	if n == nil {
		return nil, nil
	}
	switch n.Ty & 0xF {
	case TyShortNode:
		n, err := decodeShort(hash, n.GetShort(), cachegen)
		return n, wrapError(err, "short")
	case TyFullNode:
		n, err := decodeFull(hash, n.GetFull(), cachegen)
		return n, wrapError(err, "full")
	case TyHashNode:
		return hashNode{n.GetHash(), nil}, nil
	case TyValueNode:
		return valueNode{n.GetVal(), nil}, nil
	default:
		return nil, fmt.Errorf("invalid proto")
	}
}

func decodeShort(hash []byte, sn *ShortNode, cachegen uint16) (*shortNode, error) {
	n := &shortNode{flags: nodeFlag{hash: createHashNode(hash), gen: cachegen}}
	n.Key = compactToHex(sn.Key)
	var err error
	n.Val, err = sn.Val.decode(nil, cachegen)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func decodeFull(hash []byte, fn *FullNode, cachegen uint16) (*fullNode, error) {
	n := &fullNode{flags: nodeFlag{hash: createHashNode(hash), gen: cachegen}}
	var err error
	for i := 0; i < len(fn.Nodes); i++ {
		if fn.Nodes[i] != nil && fn.Nodes[i].Ty&0xF > 0 {
			index := fn.Nodes[i].Ty >> 4
			n.Children[index], err = fn.Nodes[i].decode(nil, cachegen)
			if err != nil {
				return nil, err
			}
		}
	}
	return n, nil
}

// wraps a decoding error with information about the path to the
// invalid child node (for debugging encoding issues).
type decodeError struct {
	what  error
	stack []string
}

func wrapError(err error, ctx string) error {
	if err == nil {
		return nil
	}
	if decErr, ok := err.(*decodeError); ok {
		decErr.stack = append(decErr.stack, ctx)
		return decErr
	}
	return &decodeError{err, []string{ctx}}
}

func (err *decodeError) Error() string {
	return fmt.Sprintf("%v (decode path: %s)", err.what, strings.Join(err.stack, "<-"))
}
