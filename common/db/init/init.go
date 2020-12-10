package init

import (
	_ "github.com/33cn/chain33/common/db/badger"  //auto gen
	_ "github.com/33cn/chain33/common/db/level"   //auto gen
	_ "github.com/33cn/chain33/common/db/local"   //auto gen
	_ "github.com/33cn/chain33/common/db/mem"     //auto gen
	_ "github.com/33cn/chain33/common/db/mvcc"    //auto gen
	_ "github.com/33cn/chain33/common/db/pegasus" //auto gen
	_ "github.com/33cn/chain33/common/db/ssdb"    //auto gen
)
