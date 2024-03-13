package dht

import (
	_ "github.com/33cn/chain33/system/p2p/dht/protocol/broadcast" //register init package
	_ "github.com/33cn/chain33/system/p2p/dht/protocol/download"  //register init package
	_ "github.com/33cn/chain33/system/p2p/dht/protocol/p2pstore"  //register init package
	_ "github.com/33cn/chain33/system/p2p/dht/protocol/peer"      //register init package
	_ "github.com/33cn/chain33/system/p2p/dht/protocol/snow"      // load package
)
