// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build go1.8

// package main 用于测试数据库中的MAVL节点数目
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	clog "github.com/33cn/chain33/common/log"
	log "github.com/33cn/chain33/common/log/log15"
	mavl "github.com/33cn/chain33/system/store/mavl/db"
	"github.com/33cn/chain33/types"
)

var (
	queryHashCount = flag.Bool("qh", false, "查询hash节点计数")
	queryLeafCount = flag.Bool("ql", false, "查询叶子节点计数")
	queryAllCount  = flag.Bool("qa", false, "查询全部节点计数")

	queryLeafCountCount    = flag.Bool("qlc", false, "查询叶子节点索引计数")
	queryOldLeafCountCount = flag.Bool("qolc", false, "查询叶子节点二级索引计数")
	pruneTreeHeight        = flag.Int64("pth", 0, "裁剪树高度")

	querySameLeafCountKeyPri = flag.String("qslc", "", "查询所有相同前缀的key的索引节点")
	queryHashNode            = flag.String("qhn", "", "查询hash节点是否存在且list左右子节点")

	//queryLeafNodeParent
	key    = flag.String("k", "", "查询叶子节点父节点所需要：key")
	hash   = flag.String("h", "", "查询叶子节点父节点所需要：hash")
	height = flag.Int64("hei", 0, "查询叶子节点父节点所需要：height")
)

func main() {
	flag.Parse()
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(dir)
	str := "dbug"

	log1 := &types.Log{
		Loglevel:        str,
		LogConsoleLevel: "info",
		LogFile:         "logs/syc.log",
		MaxFileSize:     400,
		MaxBackups:      100,
		MaxAge:          28,
		LocalTime:       true,
		Compress:        false,
	}
	clog.SetFileLog(log1)

	log.Info("dir", "test", dir)
	db := dbm.NewDB("store", "leveldb", dir, 100)

	if *queryHashCount {
		mavl.PruningTreePrintDB(db, []byte("_mh_"))
	}

	if *queryLeafCount {
		mavl.PruningTreePrintDB(db, []byte("_mb_"))
	}

	if *queryAllCount {
		mavl.PruningTreePrintDB(db, nil)
	}

	if *queryLeafCountCount {
		mavl.PruningTreePrintDB(db, []byte("..mk.."))
	}

	if *queryOldLeafCountCount {
		mavl.PruningTreePrintDB(db, []byte("..mok.."))
	}

	if *pruneTreeHeight > 0 {
		treeCfg := &mavl.TreeConfig{
			PruneHeight: mavl.DefaultPruneHeight,
		}
		mavl.PruningTree(db, *pruneTreeHeight, treeCfg)
	}

	if len(*querySameLeafCountKeyPri) > 0 {
		mavl.PrintSameLeafKey(db, *querySameLeafCountKeyPri)
	}

	if len(*key) > 0 && len(*hash) > 0 && *height > 0 {
		hashb, err := common.FromHex(*hash)
		if err != nil {
			fmt.Println("common.FromHex err", *hash)
			return
		}
		mavl.PrintLeafNodeParent(db, []byte(*key), hashb, *height)
	}

	if len(*queryHashNode) > 0 {
		key, err := common.FromHex(*queryHashNode)
		if err == nil {
			mavl.PrintNodeDb(db, key)
		}
	}
}
