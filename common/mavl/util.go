package mavl

import (
	"fmt"
)

// Prints the in-memory children recursively.
func PrintNode(node *Node) {
	fmt.Println("==== NODE")
	if node != nil {
		printNode(node, 0)
	}
	fmt.Println("==== END")
}

func printNode(node *Node, indent int) {
	indentPrefix := ""
	for i := 0; i < indent; i++ {
		indentPrefix += "    "
	}

	if node.rightNode != nil {
		printNode(node.rightNode, indent+1)
	} else if node.rightHash != nil {
		fmt.Printf("node.rightHash:%s    %X\n", indentPrefix, node.rightHash)
	}

	fmt.Printf("%s%v:%v\n", indentPrefix, node.key, node.height)

	//fmt.Printf("nodeinfo:key:%v,value:%v,height:%v,size:%v\n", node.key, node.value, node.height, node.size)

	fmt.Printf("nodeinfo:hash:%X,lefthash:%X,rightHash:%X,persisted:%v\n", node.hash, node.leftHash, node.rightHash, node.persisted)

	if node.leftNode != nil {
		printNode(node.leftNode, indent+1)
	} else if node.leftHash != nil {
		fmt.Printf("node.leftHash:%s    %X\n", indentPrefix, node.leftHash)
	}

}

func maxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
