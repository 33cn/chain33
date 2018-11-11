// +build go1.8

package main

import (
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/util/cli"
)

func main() {
	cli.RunChain33("")
}
