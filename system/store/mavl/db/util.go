// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mavl

import (
	"fmt"
	"strings"
)

// PrintNode the in-memory children recursively.
func PrintNode(node *Node) {
	fmt.Println("==== NODE")
	if node != nil {
		printNode(node, 0)
	}
	fmt.Println("==== END")
}

func printNode(node *Node, indent int) {
	indentPrefix := strings.Repeat("  ", indent)
	if node.rightNode != nil {
		printNode(node.rightNode, indent+1)
	}
	fmt.Printf("%s|-%s:%03d\n", indentPrefix, string(node.key), node.height)
	if node.leftNode != nil {
		printNode(node.leftNode, indent+1)
	}
}

func maxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
