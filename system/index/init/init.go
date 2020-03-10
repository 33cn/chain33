package init

import (
	_ "github.com/33cn/chain33/system/index/addrindex" // register addrindex
	_ "github.com/33cn/chain33/system/index/fee"       // register fee
	_ "github.com/33cn/chain33/system/index/stat"      // register stat
	_ "github.com/33cn/chain33/system/index/txindex"   // register txindex
)
