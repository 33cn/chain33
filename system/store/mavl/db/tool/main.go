// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build go1.8

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"bufio"
	"os/signal"
	"syscall"

	dbm "github.com/33cn/chain33/common/db"
	clog "github.com/33cn/chain33/common/log"
	log "github.com/33cn/chain33/common/log/log15"
	mavl "github.com/33cn/chain33/system/store/mavl/db"
	"github.com/33cn/chain33/types"
)

func main() {
	stdin := bufio.NewReader(os.Stdin)
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(dir)

	str := ""
	fmt.Println("请输入打印日志级别(debug(dbug)/info/warn/error(eror)/crit)")
	fmt.Fscan(stdin, &str)
	stdin.ReadString('\n')
	if str == "" {
		str = "info"
	}

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

	log.Info("test", dir)
	db := dbm.NewDB("store", "leveldb", dir, 100)

	a := 0
	fmt.Println("是否需要查询叶子节点索引计数")
	fmt.Fscan(stdin, &a)
	stdin.ReadString('\n')
	if a > 0 {
		mavl.PruningTreePrintDB(db, []byte("..mk.."))
	}
	a = 0
	fmt.Println("是否需要查询hash节点计数")
	fmt.Fscan(stdin, &a)
	stdin.ReadString('\n')
	if a > 0 {
		mavl.PruningTreePrintDB(db, []byte("_mh_"))
	}
	a = 0
	fmt.Println("是否需要查询叶子节点计数")
	fmt.Fscan(stdin, &a)
	stdin.ReadString('\n')
	if a > 0 {
		mavl.PruningTreePrintDB(db, []byte("_mb_"))
	}
	a = 0
	fmt.Println("是否需要查询全部节点计数")
	fmt.Fscan(stdin, &a)
	stdin.ReadString('\n')
	if a > 0 {
		mavl.PruningTreePrintDB(db, nil)
	}
	a = 0
	fmt.Println("是否需要查询节点删除pool")
	fmt.Fscan(stdin, &a)
	stdin.ReadString('\n')
	if a > 0 {
		mavl.PruningTreePrintDB(db, []byte("_..md.._"))
	}
	a = 0
	fmt.Println("是否需要查询二级叶子节点索引计数")
	fmt.Fscan(stdin, &a)
	stdin.ReadString('\n')
	if a > 0 {
		mavl.PruningTreePrintDB(db, []byte("..mok.."))
	}
	a = 0
	fmt.Println("是否需要裁剪树,请输入最大裁剪数高度")
	fmt.Fscan(stdin, &a)
	stdin.ReadString('\n')
	if a > 0 {
		mavl.PruningTree(db, int64(a))
	}
	fmt.Println("over")
	exit := make(chan os.Signal, 10)                     //初始化一个channel
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM) //notify方法用来监听收到的信号
	sig := <-exit
	fmt.Println(sig.String())
}
